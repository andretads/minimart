import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { vi } from 'vitest';
import { WishlistButtonComponent } from './wishlist-button.component';
import { WishlistStore } from './wishlist.store';
import { AuthStore } from '@core/auth/auth.store';

describe('WishlistButtonComponent', () => {
  let store: {
    isInWishlist: ReturnType<typeof vi.fn>;
    toggle: ReturnType<typeof vi.fn>;
    loading: ReturnType<typeof signal>;
  };
  let auth: { isAuthenticated: ReturnType<typeof signal> };

  beforeEach(async () => {
    store = { isInWishlist: vi.fn(() => false), toggle: vi.fn(), loading: signal(false) };
    auth = { isAuthenticated: signal(true) };
    await TestBed.configureTestingModule({
      imports: [WishlistButtonComponent],
      providers: [
        { provide: WishlistStore, useValue: store },
        { provide: AuthStore, useValue: auth },
      ],
    }).compileComponents();
  });

  it('inWishlist reflects store', () => {
    store.isInWishlist.mockReturnValue(true);
    const f = TestBed.createComponent(WishlistButtonComponent);
    f.componentRef.setInput('productId', 'p1');
    expect(f.componentInstance.inWishlist()).toBe(true);
  });

  it('toggle calls store when authenticated', () => {
    const f = TestBed.createComponent(WishlistButtonComponent);
    f.componentRef.setInput('productId', 'p1');
    f.componentInstance.toggle();
    expect(store.toggle).toHaveBeenCalledWith('p1');
  });

  it('toggle no-ops when unauthenticated', () => {
    auth.isAuthenticated.set(false);
    const f = TestBed.createComponent(WishlistButtonComponent);
    f.componentRef.setInput('productId', 'p1');
    f.componentInstance.toggle();
    expect(store.toggle).not.toHaveBeenCalled();
  });

  it('promptLogin does nothing', () => {
    const f = TestBed.createComponent(WishlistButtonComponent);
    f.componentRef.setInput('productId', 'p1');
    expect(() => f.componentInstance.promptLogin()).not.toThrow();
  });
});
