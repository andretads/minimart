import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { HomeComponent } from './home.component';
import { ShowcaseApiService } from '@features/catalog/showcase-api.service';
import { CategoriesApiService } from '@features/catalog/categories-api.service';
import { AuthStore } from '@core/auth/auth.store';
import { CartService } from '@features/cart/cart.service';
import { CartStore } from '@features/cart/cart.store';
import { ToastService } from '@core/toast/toast.service';
import { WishlistStore } from '@features/wishlist/wishlist.store';

describe('HomeComponent', () => {
  let api: { list: ReturnType<typeof vi.fn> };
  let categoriesApi: { list: ReturnType<typeof vi.fn> };
  let router: { navigate: ReturnType<typeof vi.fn>; url: string };
  let authStore: { isAuthenticated: ReturnType<typeof vi.fn> };
  let cartService: { addItem: ReturnType<typeof vi.fn> };
  let cartStore: { setCart: ReturnType<typeof vi.fn> };
  let toast: { success: ReturnType<typeof vi.fn>; error: ReturnType<typeof vi.fn> };

  const page = {
    data: [{ id: 'p1', name: 'W', priceCents: 100 }],
    meta: { pageSize: 20, nextCursor: null, hasMore: false },
  };

  beforeEach(async () => {
    vi.useFakeTimers();
    api = { list: vi.fn(() => of(page)) };
    categoriesApi = { list: vi.fn(() => of([])) };
    router = { navigate: vi.fn(() => Promise.resolve(true)), url: '/home' };
    authStore = { isAuthenticated: vi.fn(() => true) };
    cartService = { addItem: vi.fn(() => of({ items: [] })) };
    cartStore = { setCart: vi.fn() };
    toast = { success: vi.fn(), error: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [HomeComponent],
      providers: [
        { provide: ShowcaseApiService, useValue: api },
        { provide: CategoriesApiService, useValue: categoriesApi },
        { provide: ActivatedRoute, useValue: { queryParams: of({} as Params) } },
        { provide: Router, useValue: router },
        { provide: AuthStore, useValue: authStore },
        { provide: CartService, useValue: cartService },
        { provide: CartStore, useValue: cartStore },
        { provide: ToastService, useValue: toast },
        { provide: WishlistStore, useValue: { isInWishlist: vi.fn(() => false), toggle: vi.fn() } },
      ],
    }).compileComponents();
  });

  afterEach(() => vi.useRealTimers());

  it('loads products and categories on init', () => {
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    expect(api.list).toHaveBeenCalledWith({ pageSize: 20, categoryId: undefined });
    expect(f.componentInstance.products()).toEqual(page.data);
    expect(f.componentInstance.hasMore()).toBe(false);
    expect(f.componentInstance.loading()).toBe(false);
  });

  it('reads category from query params', () => {
    TestBed.overrideProvider(ActivatedRoute, {
      useValue: { queryParams: of({ category_id: 'c1' } as Params) },
    });
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    expect(f.componentInstance.categoryId()).toBe('c1');
    expect(api.list).toHaveBeenCalledWith({ pageSize: 20, categoryId: 'c1' });
  });

  it('sets error when product load fails', () => {
    api.list.mockReturnValue(throwError(() => ({ message: 'boom' })));
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    expect(f.componentInstance.error()).toBe('boom');
    expect(f.componentInstance.products()).toEqual([]);
  });

  it('sets categories error on failure', () => {
    categoriesApi.list.mockReturnValue(throwError(() => ({ message: 'cat-fail' })));
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    expect(f.componentInstance.categoriesError()).toBe('cat-fail');
    expect(f.componentInstance.categoriesLoading()).toBe(false);
  });

  it('nextSlide cycles 0->1->2->0', () => {
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    expect(f.componentInstance.currentSlide()).toBe(0);
    f.componentInstance.nextSlide();
    expect(f.componentInstance.currentSlide()).toBe(1);
    f.componentInstance.nextSlide();
    expect(f.componentInstance.currentSlide()).toBe(2);
    f.componentInstance.nextSlide();
    expect(f.componentInstance.currentSlide()).toBe(0);
  });

  it('goToSlide sets slide', () => {
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    f.componentInstance.goToSlide(2);
    expect(f.componentInstance.currentSlide()).toBe(2);
  });

  it('auto slide advances after 5s and is cleared on destroy', () => {
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    vi.advanceTimersByTime(5000);
    expect(f.componentInstance.currentSlide()).toBe(1);
    f.destroy();
    vi.advanceTimersByTime(5000);
    expect(f.componentInstance.currentSlide()).toBe(1);
  });

  it('loadMore appends items and advances cursor', () => {
    const nextPage = {
      data: [{ id: 'p2', name: 'W2', priceCents: 200 }],
      meta: { pageSize: 20, nextCursor: 'cur2', hasMore: false },
    };
    api.list.mockReturnValueOnce(of(page)).mockReturnValueOnce(of(nextPage));
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    // simulate hasMore true
    f.componentInstance.hasMore.set(true);
    f.componentInstance.nextCursor.set('cur1');
    f.componentInstance.loadMore();
    expect(api.list).toHaveBeenLastCalledWith({
      pageSize: 20,
      cursor: 'cur1',
      categoryId: undefined,
    });
    expect(f.componentInstance.products()).toEqual([...page.data, ...nextPage.data]);
    expect(f.componentInstance.hasMore()).toBe(false);
    expect(f.componentInstance.loadingMore()).toBe(false);
  });

  it('onAddToCart redirects to login when unauthenticated', () => {
    authStore.isAuthenticated.mockReturnValue(false);
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    f.componentInstance.onAddToCart('p1');
    expect(router.navigate).toHaveBeenCalledWith(['/login'], {
      queryParams: { returnUrl: '/home' },
    });
    expect(cartService.addItem).not.toHaveBeenCalled();
  });

  it('onAddToCart adds item and toasts success', () => {
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    f.componentInstance.onAddToCart('p1');
    expect(cartService.addItem).toHaveBeenCalledWith('p1', 1);
    expect(cartStore.setCart).toHaveBeenCalled();
    expect(toast.success).toHaveBeenCalled();
  });

  it('onAddToCart toasts error on failure', () => {
    cartService.addItem.mockReturnValue(throwError(() => ({ message: 'nope' })));
    const f = TestBed.createComponent(HomeComponent);
    f.detectChanges();
    f.componentInstance.onAddToCart('p1');
    expect(toast.error).toHaveBeenCalledWith('nope');
  });
});
