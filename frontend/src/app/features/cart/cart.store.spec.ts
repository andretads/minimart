import { TestBed } from '@angular/core/testing';
import { CartStore } from './cart.store';
import { AuthStore } from '@core/auth/auth.store';
import { Cart } from '@shared/models/cart.models';

describe('CartStore', () => {
  let store: CartStore;
  let auth: { isAuthenticated: ReturnType<typeof vi.fn> };

  beforeEach(() => {
    auth = { isAuthenticated: vi.fn(() => true) };
    TestBed.configureTestingModule({
      providers: [CartStore, { provide: AuthStore, useValue: auth }],
    });
    store = TestBed.inject(CartStore);
  });

  it('starts empty', () => {
    expect(store.cart()).toBeNull();
    expect(store.loading()).toBe(false);
    expect(store.error()).toBeNull();
    expect(store.itemCount()).toBe(0);
    expect(store.estimatedTotal()).toBe(0);
    expect(store.isEmpty()).toBe(true);
  });

  it('setCart updates cart and computeds', () => {
    const cart: Cart = {
      id: 'c1',
      customerId: 'cust',
      status: 'active',
      items: [
        {
          productId: 'p1',
          productNameSnapshot: 'W',
          unitPriceSnapshotCents: 500,
          quantity: 2,
          lineTotalCents: 1000,
        },
      ],
      estimatedTotalCents: 1000,
      createdAt: 't',
      updatedAt: 't',
    };
    store.setCart(cart);
    expect(store.cart()).toEqual(cart);
    expect(store.itemCount()).toBe(2);
    expect(store.estimatedTotal()).toBe(1000);
    expect(store.isEmpty()).toBe(false);
  });

  it('setLoading/setError', () => {
    store.setLoading(true);
    store.setError('e');
    expect(store.loading()).toBe(true);
    expect(store.error()).toBe('e');
  });

  it('clear nulls cart and error but not loading', () => {
    store.setLoading(true);
    store.setError('e');
    store.setCart({
      id: 'c',
      customerId: 'x',
      status: 'active',
      items: [],
      estimatedTotalCents: 0,
      createdAt: 't',
      updatedAt: 't',
    });
    store.clear();
    expect(store.cart()).toBeNull();
    expect(store.error()).toBeNull();
    expect(store.loading()).toBe(true);
  });

  it('auth effect nulls cart when logged out', () => {
    const cart: Cart = {
      id: 'c1',
      customerId: 'cust',
      status: 'active',
      items: [],
      estimatedTotalCents: 0,
      createdAt: 't',
      updatedAt: 't',
    };
    store.setCart(cart);
    auth.isAuthenticated.mockReturnValue(false);
    // trigger effect by re-evaluating: TestBed effect runs on change detection
    TestBed.flushEffects();
    expect(store.cart()).toBeNull();
  });
});
