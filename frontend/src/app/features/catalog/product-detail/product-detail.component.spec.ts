import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { Observable, of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { ActivatedRoute, ParamMap, Router, convertToParamMap } from '@angular/router';
import { ProductDetailComponent } from './product-detail.component';
import { ShowcaseApiService } from '@features/catalog/showcase-api.service';
import { AuthStore } from '@core/auth/auth.store';
import { CartService } from '@features/cart/cart.service';
import { CartStore } from '@features/cart/cart.store';
import { ToastService } from '@core/toast/toast.service';
import { WishlistStore } from '@features/wishlist/wishlist.store';
import { Product } from '@shared/models/catalog.models';

const product: Product = {
  id: 'p1',
  name: 'Widget',
  priceCents: 1299,
  stock: 3,
  totalReviews: 4,
  totalRating: 16,
  images: [
    {
      id: 'i1',
      productId: 'p1',
      url: 'a.png',
      contentType: 'image/png',
      sizeBytes: 1,
      position: 0,
    },
    {
      id: 'i2',
      productId: 'p1',
      url: 'b.png',
      contentType: 'image/png',
      sizeBytes: 1,
      position: 1,
    },
  ],
  details: Array.from({ length: 12 }, (_, i) => ({ name: `L${i}`, value: `V${i}` })),
};

describe('ProductDetailComponent', () => {
  let api: {
    get: ReturnType<typeof vi.fn>;
    getReviews: ReturnType<typeof vi.fn>;
    createReview: ReturnType<typeof vi.fn>;
  };
  let route: { paramMap: Observable<ParamMap>; snapshot: { paramMap: ParamMap } };
  let router: { navigate: ReturnType<typeof vi.fn> };
  let auth: { isAuthenticated: ReturnType<typeof signal> };
  let cartService: { addItem: ReturnType<typeof vi.fn> };
  let cartStore: { setCart: ReturnType<typeof vi.fn> };
  let toast: { success: ReturnType<typeof vi.fn>; error: ReturnType<typeof vi.fn> };
  let wishlist: { isInWishlist: ReturnType<typeof vi.fn>; toggle: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    api = {
      get: vi.fn(() => of(product)),
      getReviews: vi.fn(() =>
        of({
          data: [
            {
              id: 'r1',
              productId: 'p1',
              customerId: 'c1',
              description: 'ok',
              rating: 5,
              status: 'approved',
              createdAt: '2026-01-01T00:00:00Z',
            },
          ],
          meta: { page: 1, pageSize: 20, total: 1, totalPages: 1 },
        }),
      ),
      createReview: vi.fn(() => of(void 0)),
    };
    route = {
      paramMap: of(convertToParamMap({ id: 'p1' })),
      snapshot: { paramMap: convertToParamMap({ id: 'p1' }) },
    };
    router = { navigate: vi.fn(() => Promise.resolve(true)) };
    auth = { isAuthenticated: signal(true) };
    cartService = { addItem: vi.fn(() => of({ items: [] })) };
    cartStore = { setCart: vi.fn() };
    toast = { success: vi.fn(), error: vi.fn() };
    wishlist = { isInWishlist: vi.fn(() => false), toggle: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [ProductDetailComponent],
      providers: [
        { provide: ShowcaseApiService, useValue: api },
        { provide: ActivatedRoute, useValue: route },
        { provide: Router, useValue: router },
        { provide: AuthStore, useValue: auth },
        { provide: CartService, useValue: cartService },
        { provide: CartStore, useValue: cartStore },
        { provide: ToastService, useValue: toast },
        { provide: WishlistStore, useValue: wishlist },
      ],
    }).compileComponents();
  });

  it('loads product and reviews on init', () => {
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.product()).toEqual(product);
    expect(f.componentInstance.reviews().length).toBe(1);
    expect(f.componentInstance.loading()).toBe(false);
  });

  it('sets error when product load fails', () => {
    api.get.mockReturnValue(throwError(() => ({ message: 'gone' })));
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.error()).toBe('gone');
    expect(f.componentInstance.product()).toBeNull();
  });

  it('defaults reviews to [] when review load fails', () => {
    api.getReviews.mockReturnValue(throwError(() => new Error('x')));
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.reviews()).toEqual([]);
  });

  it('computes stock-related signals', () => {
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.outOfStock()).toBe(false);
    expect(f.componentInstance.maxQuantity()).toBe(3);
    expect(f.componentInstance.stockLabel()).toBe('3 em estoque');
  });

  it('stockLabel is Em estoque when stock undefined', () => {
    api.get.mockReturnValue(of({ ...product, stock: undefined }));
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.stockLabel()).toBe('Em estoque');
  });

  it('stockLabel is Esgotado when stock zero', () => {
    api.get.mockReturnValue(of({ ...product, stock: 0 }));
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.stockLabel()).toBe('Esgotado');
    expect(f.componentInstance.outOfStock()).toBe(true);
  });

  it('maxQuantity clamps undefined stock to 99', () => {
    api.get.mockReturnValue(of({ ...product, stock: undefined }));
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.maxQuantity()).toBe(99);
  });

  it('rating signals', () => {
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.hasRating()).toBe(true);
    expect(f.componentInstance.ratingLabel()).toBe('4,0');
    expect(f.componentInstance.stars()).toEqual([true, true, true, true, false]);
  });

  it('gallery and activeImageUrl', () => {
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.gallery().length).toBe(2);
    expect(f.componentInstance.activeImageUrl()).toBe('a.png');
    f.componentInstance.setMainImage(1);
    expect(f.componentInstance.activeImageUrl()).toBe('b.png');
  });

  it('activeImageUrl falls back to product.imageUrl', () => {
    api.get.mockReturnValue(of({ ...product, images: [], imageUrl: 'main.png' }));
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.activeImageUrl()).toBe('main.png');
  });

  it('paginates details', () => {
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    expect(f.componentInstance.specsTotalPages()).toBe(3);
    expect(f.componentInstance.paginatedDetails().length).toBe(5);
    f.componentInstance.specsPage.set(3);
    expect(f.componentInstance.paginatedDetails().length).toBe(2);
  });

  it('onTabChange resets specs page', () => {
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    f.componentInstance.specsPage.set(2);
    f.componentInstance.onTabChange();
    expect(f.componentInstance.specsPage()).toBe(1);
  });

  it('toggleWishlist toggles when product present', () => {
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    f.componentInstance.toggleWishlist();
    expect(wishlist.toggle).toHaveBeenCalledWith('p1');
  });

  it('toggleWishlist no-ops without product', () => {
    api.get.mockReturnValue(throwError(() => new Error('x')));
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    f.componentInstance.toggleWishlist();
    expect(wishlist.toggle).not.toHaveBeenCalled();
  });

  it('addToCart adds item and toasts success', () => {
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    f.componentInstance.quantity.set(2);
    f.componentInstance.addToCart();
    expect(cartService.addItem).toHaveBeenCalledWith('p1', 2);
    expect(cartStore.setCart).toHaveBeenCalled();
    expect(toast.success).toHaveBeenCalled();
  });

  it('addToCart is blocked when unauthenticated', () => {
    auth.isAuthenticated.set(false);
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    f.componentInstance.addToCart();
    expect(cartService.addItem).not.toHaveBeenCalled();
  });

  it('addToCart toasts error on failure', () => {
    cartService.addItem.mockReturnValue(throwError(() => ({ message: 'nope' })));
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    f.componentInstance.addToCart();
    expect(toast.error).toHaveBeenCalledWith('nope');
    expect(f.componentInstance.addingToCart()).toBe(false);
  });

  it('onReviewSubmitted submits, closes form and reloads', () => {
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    f.componentInstance.showReviewForm.set(true);
    f.componentInstance.onReviewSubmitted({ description: 'great', rating: 5 });
    expect(api.createReview).toHaveBeenCalledWith('p1', { description: 'great', rating: 5 });
    expect(toast.success).toHaveBeenCalled();
    expect(f.componentInstance.showReviewForm()).toBe(false);
    expect(f.componentInstance.submittingReview()).toBe(false);
  });

  it('onReviewSubmitted toasts error on failure', () => {
    api.createReview.mockReturnValue(throwError(() => ({ message: 'bad' })));
    const f = TestBed.createComponent(ProductDetailComponent);
    f.detectChanges();
    f.componentInstance.onReviewSubmitted({ description: 'x', rating: 1 });
    expect(toast.error).toHaveBeenCalledWith('bad');
  });
});
