import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ProductCardComponent } from './product-card.component';
import { WishlistStore } from '@features/wishlist/wishlist.store';
import { AuthStore } from '@core/auth/auth.store';
import { Product } from '@shared/models/catalog.models';

const product: Product = {
  id: 'p1',
  name: 'Widget',
  priceCents: 1299,
  stock: 5,
  totalReviews: 10,
  totalRating: 45,
  imageUrl: undefined,
};

describe('ProductCardComponent', () => {
  let wishlist: { isInWishlist: ReturnType<typeof vi.fn>; toggle: ReturnType<typeof vi.fn> };
  let auth: { isAuthenticated: ReturnType<typeof signal> };

  beforeEach(async () => {
    wishlist = { isInWishlist: vi.fn(() => false), toggle: vi.fn() };
    auth = { isAuthenticated: signal(true) };
    await TestBed.configureTestingModule({
      imports: [ProductCardComponent],
      providers: [
        provideRouter([]),
        { provide: WishlistStore, useValue: wishlist },
        { provide: AuthStore, useValue: auth },
      ],
    }).compileComponents();
  });

  it('hasRating true when totalReviews > 0', () => {
    const f = TestBed.createComponent(ProductCardComponent);
    f.componentRef.setInput('product', product);
    expect(f.componentInstance.hasRating()).toBe(true);
    expect(f.componentInstance.ratingLabel()).toBe('4,5');
    expect(f.componentInstance.stars()).toEqual([true, true, true, true, true]);
  });

  it('hasRating false when no reviews', () => {
    const f = TestBed.createComponent(ProductCardComponent);
    f.componentRef.setInput('product', { ...product, totalReviews: 0, totalRating: 0 });
    expect(f.componentInstance.hasRating()).toBe(false);
    expect(f.componentInstance.ratingLabel()).toBe('0,0');
    expect(f.componentInstance.stars()).toEqual([false, false, false, false, false]);
  });

  it('isOutOfStock true when stock <= 0', () => {
    const f = TestBed.createComponent(ProductCardComponent);
    f.componentRef.setInput('product', { ...product, stock: 0 });
    expect(f.componentInstance.isOutOfStock()).toBe(true);
  });

  it('isOutOfStock false when stock undefined', () => {
    const f = TestBed.createComponent(ProductCardComponent);
    f.componentRef.setInput('product', { ...product, stock: undefined });
    expect(f.componentInstance.isOutOfStock()).toBe(false);
  });

  it('inWishlist reflects store', () => {
    wishlist.isInWishlist.mockReturnValue(true);
    const f = TestBed.createComponent(ProductCardComponent);
    f.componentRef.setInput('product', product);
    expect(f.componentInstance.inWishlist()).toBe(true);
  });

  it('toggleWishlist prevents default, stops propagation and toggles', () => {
    const f = TestBed.createComponent(ProductCardComponent);
    f.componentRef.setInput('product', product);
    const event = { preventDefault: vi.fn(), stopPropagation: vi.fn() } as unknown as Event;
    f.componentInstance.toggleWishlist(event);
    expect(event.preventDefault).toHaveBeenCalled();
    expect(event.stopPropagation).toHaveBeenCalled();
    expect(wishlist.toggle).toHaveBeenCalledWith('p1');
  });

  it('toggleWishlist does nothing when unauthenticated', () => {
    auth.isAuthenticated.set(false);
    const f = TestBed.createComponent(ProductCardComponent);
    f.componentRef.setInput('product', product);
    const event = { preventDefault: vi.fn(), stopPropagation: vi.fn() } as unknown as Event;
    f.componentInstance.toggleWishlist(event);
    expect(wishlist.toggle).not.toHaveBeenCalled();
  });

  it('quickView prevents default and stops propagation', () => {
    const f = TestBed.createComponent(ProductCardComponent);
    f.componentRef.setInput('product', product);
    const event = { preventDefault: vi.fn(), stopPropagation: vi.fn() } as unknown as Event;
    f.componentInstance.quickView(event);
    expect(event.preventDefault).toHaveBeenCalled();
    expect(event.stopPropagation).toHaveBeenCalled();
  });

  it('emits addToCart from template button', () => {
    const f = TestBed.createComponent(ProductCardComponent);
    f.componentRef.setInput('product', product);
    let emitted: string | undefined;
    f.componentInstance.addToCart.subscribe((id) => (emitted = id));
    f.detectChanges();
    const btn = f.nativeElement.querySelector('.add-button') as HTMLButtonElement;
    btn.click();
    expect(emitted).toBe('p1');
  });
});
