import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '../../core/api/api-client';
import { Cart, CheckoutResponse } from '../../shared/models/cart.models';
import { CartService } from './cart.service';

const cart: Cart = {
  id: 'c1',
  customerId: 'cust-1',
  status: 'active',
  items: [],
  estimatedTotalCents: 500,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const checkoutResponse: CheckoutResponse = {
  orderId: 'o1',
  userId: 'u1',
  customerId: 'cust-1',
  status: 'placed',
  totalCents: 500,
  createdAt: '2026-01-01T00:00:00Z',
};

describe('CartService', () => {
  let service: CartService;
  const api = { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [CartService, { provide: ApiClient, useValue: api }],
    });
    service = TestBed.inject(CartService);
    api.get.mockReset();
    api.post.mockReset();
    api.patch.mockReset();
    api.delete.mockReset();
  });

  it('gets the cart for the authenticated user', () => {
    api.get.mockReturnValue(of(cart));

    let result: Cart | undefined;
    service.getCart().subscribe((value) => (result = value));

    expect(api.get).toHaveBeenCalledWith('/v1/cart');
    expect(result).toEqual(cart);
  });

  it('adds an item to the cart', () => {
    const updatedCart = {
      ...cart,
      items: [{ productId: 'p1', quantity: 2, name: 'Test', unitPriceCents: 250 }],
    };
    api.post.mockReturnValue(of(updatedCart));

    let result: Cart | undefined;
    service.addItem('p1', 2).subscribe((value) => (result = value));

    expect(api.post).toHaveBeenCalledWith('/v1/cart/items', { productId: 'p1', quantity: 2 });
    expect(result).toEqual(updatedCart);
  });

  it('updates an item quantity in the cart', () => {
    const updatedCart = {
      ...cart,
      items: [{ productId: 'p1', quantity: 5, name: 'Test', unitPriceCents: 250 }],
    };
    api.patch.mockReturnValue(of(updatedCart));

    let result: Cart | undefined;
    service.updateItem('p1', 5).subscribe((value) => (result = value));

    expect(api.patch).toHaveBeenCalledWith('/v1/cart/items/p1', { quantity: 5 });
    expect(result).toEqual(updatedCart);
  });

  it('removes an item from the cart', () => {
    const updatedCart = { ...cart, items: [] };
    api.delete.mockReturnValue(of(updatedCart));

    let result: Cart | undefined;
    service.removeItem('p1').subscribe((value) => (result = value));

    expect(api.delete).toHaveBeenCalledWith('/v1/cart/items/p1');
    expect(result).toEqual(updatedCart);
  });

  it('clears the cart', () => {
    api.delete.mockReturnValue(of(undefined));

    let result: void | undefined;
    service.clearCart().subscribe((value) => (result = value));

    expect(api.delete).toHaveBeenCalledWith('/v1/cart');
    expect(result).toBeUndefined();
  });

  it('checks out the cart with Idempotency-Key header', () => {
    api.post.mockReturnValue(of(checkoutResponse));

    let result: CheckoutResponse | undefined;
    service.checkout('idem-key-123', 'quote-1', 'addr-1').subscribe((value) => (result = value));

    expect(api.post).toHaveBeenCalledWith(
      '/v1/cart/checkout',
      { shippingQuoteId: 'quote-1', addressId: 'addr-1' },
      { headers: { 'Idempotency-Key': 'idem-key-123' } },
    );
    expect(result).toEqual(checkoutResponse);
  });
});
