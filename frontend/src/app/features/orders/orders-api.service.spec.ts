import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '../../core/api/api-client';
import { AuthStore } from '../../core/auth/auth.store';
import { Order, OrderListResponse } from '../../shared/models/order.models';
import { OrdersApiService } from './orders-api.service';

const order: Order = {
  id: 'o1',
  userId: 'u1',
  status: 'placed',
  items: [],
  totalCents: 500,
  createdAt: '2026-01-01T00:00:00Z',
};

describe('OrdersApiService', () => {
  let service: OrdersApiService;
  const api = { get: vi.fn(), post: vi.fn() };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        OrdersApiService,
        { provide: ApiClient, useValue: api },
        { provide: AuthStore, useValue: { userId: () => 'u1' } },
      ],
    });
    service = TestBed.inject(OrdersApiService);
    api.get.mockReset();
    api.post.mockReset();
  });

  it('lists orders for the authenticated user with pagination', () => {
    const response: OrderListResponse = {
      data: [order],
      meta: { page: 2, pageSize: 10, total: 11, totalPages: 2 },
    };
    api.get.mockReturnValue(of(response));

    let result: OrderListResponse | undefined;
    service.list(2, 10).subscribe((value) => (result = value));

    expect(api.get).toHaveBeenCalledWith('/v1/orders?user_id=u1&page=2&page_size=10');
    expect(result).toEqual(response);
  });

  it('list throws synchronously when unauthenticated', () => {
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [
        OrdersApiService,
        { provide: ApiClient, useValue: api },
        { provide: AuthStore, useValue: { userId: () => null } },
      ],
    });
    const unauth = TestBed.inject(OrdersApiService);
    expect(() => unauth.list()).toThrow('User not authenticated');
  });

  it('gets an order by id', () => {
    api.get.mockReturnValue(of(order));

    let result: Order | undefined;
    service.get('o1').subscribe((value) => (result = value));

    expect(api.get).toHaveBeenCalledWith('/v1/orders/o1');
    expect(result).toEqual(order);
  });

  it('cancels an order', () => {
    const cancelled = { ...order, status: 'cancelled' as const };
    api.post.mockReturnValue(of(cancelled));

    let result: Order | undefined;
    service.cancel('o1').subscribe((value) => (result = value));

    expect(api.post).toHaveBeenCalledWith('/v1/orders/o1/cancel', {});
    expect(result).toEqual(cancelled);
  });
});
