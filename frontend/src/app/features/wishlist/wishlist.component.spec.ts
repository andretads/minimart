import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { provideRouter, Router } from '@angular/router';
import { WishlistComponent } from './wishlist.component';
import { WishlistStore } from './wishlist.store';
import { AuthStore } from '@core/auth/auth.store';
import { ShowcaseApiService } from '@features/catalog/showcase-api.service';
import { CartService } from '@features/cart/cart.service';
import { CartStore } from '@features/cart/cart.store';
import { ToastService } from '@core/toast/toast.service';
import { Wishlist } from '@shared/models/wishlist.models';
import { Product } from '@shared/models/catalog.models';

const product = (id: string): Product => ({ id, name: id, priceCents: 100 });

describe('WishlistComponent', () => {
  let store: {
    wishlist: ReturnType<typeof signal>;
    loading: ReturnType<typeof signal>;
    load: ReturnType<typeof vi.fn>;
    clear: ReturnType<typeof vi.fn>;
    isInWishlist: ReturnType<typeof vi.fn>;
    toggle: ReturnType<typeof vi.fn>;
  };
  let api: { get: ReturnType<typeof vi.fn> };
  let cartService: { addItem: ReturnType<typeof vi.fn> };
  let cartStore: { setCart: ReturnType<typeof vi.fn> };
  let toast: { success: ReturnType<typeof vi.fn>; error: ReturnType<typeof vi.fn> };
  let authSignal = signal(true);

  beforeEach(async () => {
    store = {
      wishlist: signal<Wishlist | null>(null),
      loading: signal(false),
      load: vi.fn(),
      clear: vi.fn(),
      isInWishlist: vi.fn(() => false),
      toggle: vi.fn(),
    };
    api = { get: vi.fn((id: string) => of(product(id))) };
    authSignal = signal(true);
    cartService = { addItem: vi.fn(() => of({ items: [] })) };
    cartStore = { setCart: vi.fn() };
    toast = { success: vi.fn(), error: vi.fn() };
    await TestBed.configureTestingModule({
      imports: [WishlistComponent],
      providers: [
        provideRouter([]),
        { provide: WishlistStore, useValue: store },
        {
          provide: AuthStore,
          useValue: { isAuthenticated: authSignal, userId: vi.fn(() => 'u1') },
        },
        { provide: ShowcaseApiService, useValue: api },
        { provide: CartService, useValue: cartService },
        { provide: CartStore, useValue: cartStore },
        { provide: ToastService, useValue: toast },
      ],
    }).compileComponents();
  });

  it('calls store.load on init', () => {
    const f = TestBed.createComponent(WishlistComponent);
    f.detectChanges();
    expect(store.load).toHaveBeenCalled();
  });

  it('fetches products for wishlist items', () => {
    store.wishlist.set({
      id: 'w1',
      customerId: 'c1',
      items: [
        { productId: 'p1', addedAt: 't' },
        { productId: 'p2', addedAt: 't' },
      ],
      createdAt: 't',
      updatedAt: 't',
    });
    const f = TestBed.createComponent(WishlistComponent);
    f.detectChanges();
    TestBed.flushEffects();
    expect(f.componentInstance.products().map((p) => p.id)).toEqual(['p1', 'p2']);
  });

  it('drops failed product fetches', () => {
    api.get.mockImplementation((id: string) =>
      id === 'p2' ? throwError(() => new Error('x')) : of(product(id)),
    );
    store.wishlist.set({
      id: 'w1',
      customerId: 'c1',
      items: [
        { productId: 'p1', addedAt: 't' },
        { productId: 'p2', addedAt: 't' },
      ],
      createdAt: 't',
      updatedAt: 't',
    });
    const f = TestBed.createComponent(WishlistComponent);
    f.detectChanges();
    TestBed.flushEffects();
    expect(f.componentInstance.products().map((p) => p.id)).toEqual(['p1']);
  });

  it('empty wishlist yields no products', () => {
    store.wishlist.set({ id: 'w1', customerId: 'c1', items: [], createdAt: 't', updatedAt: 't' });
    const f = TestBed.createComponent(WishlistComponent);
    f.detectChanges();
    TestBed.flushEffects();
    expect(f.componentInstance.products()).toEqual([]);
  });

  it('clearWishlist calls store.clear and empties products', () => {
    const f = TestBed.createComponent(WishlistComponent);
    f.detectChanges();
    f.componentInstance.clearWishlist();
    expect(store.clear).toHaveBeenCalled();
    expect(f.componentInstance.products()).toEqual([]);
  });

  it('onAddToCart redirects to login when unauthenticated', () => {
    authSignal.set(false);
    const f = TestBed.createComponent(WishlistComponent);
    f.detectChanges();
    const router = TestBed.inject(Router);
    const navigateSpy = vi.spyOn(router, 'navigate');
    f.componentInstance.onAddToCart('p1');
    expect(navigateSpy).toHaveBeenCalledWith(['/login'], {
      queryParams: { returnUrl: expect.any(String) },
    });
    expect(cartService.addItem).not.toHaveBeenCalled();
  });

  it('onAddToCart adds item and toasts success', () => {
    cartService.addItem.mockReturnValue(of({ items: [] }));
    const f = TestBed.createComponent(WishlistComponent);
    f.detectChanges();
    f.componentInstance.onAddToCart('p1');
    expect(cartService.addItem).toHaveBeenCalledWith('p1', 1);
    expect(cartStore.setCart).toHaveBeenCalled();
    expect(toast.success).toHaveBeenCalled();
  });

  it('onAddToCart toasts error on failure', () => {
    cartService.addItem.mockReturnValue(throwError(() => ({ message: 'nope' })));
    const f = TestBed.createComponent(WishlistComponent);
    f.detectChanges();
    f.componentInstance.onAddToCart('p1');
    expect(toast.error).toHaveBeenCalled();
  });
});
