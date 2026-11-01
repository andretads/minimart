import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { Order, OrderListResponse } from '../../../shared/models/order.models';
import { OrdersApiService } from '../orders-api.service';
import { OrderListComponent } from './order-list.component';

const order: Order = {
  id: 'order-1234',
  userId: 'u1',
  items: [
    {
      productId: 'p1',
      name: 'Café',
      unitPriceCents: 500,
      quantity: 2,
      lineTotalCents: 1000,
    },
  ],
  totalCents: 1000,
  status: 'placed',
  createdAt: '2026-01-02T10:00:00Z',
};

describe('OrderListComponent', () => {
  let ordersApi: { list: ReturnType<typeof vi.fn> };
  let router: { navigate: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    ordersApi = {
      list: vi.fn((): ReturnType<OrdersApiService['list']> =>
        of({
          data: [order],
          meta: { page: 1, pageSize: 20, total: 21, totalPages: 2 },
        } as OrderListResponse),
      ),
    };
    router = { navigate: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [OrderListComponent],
      providers: [
        { provide: OrdersApiService, useValue: ordersApi },
        { provide: Router, useValue: router },
        { provide: ActivatedRoute, useValue: { queryParams: of({ page: '1' }) } },
      ],
    }).compileComponents();
  });

  it('loads the first page for the current user and renders orders', () => {
    const fixture = TestBed.createComponent(OrderListComponent);
    fixture.detectChanges();

    expect(ordersApi.list).toHaveBeenCalledWith(1, 5);
    expect(fixture.componentInstance.orders()).toEqual([order]);
    expect(fixture.componentInstance.total()).toBe(21);
    expect(fixture.nativeElement.textContent).toContain('Meus pedidos');
    expect(fixture.nativeElement.textContent).toContain('ORDER-12');
    expect(fixture.nativeElement.textContent).toContain('Café');
  });

  it('navigates with the selected page', () => {
    const fixture = TestBed.createComponent(OrderListComponent);
    fixture.detectChanges();

    fixture.componentInstance.onPageChange(2);

    expect(router.navigate).toHaveBeenCalledWith([], {
      queryParams: { page: 2 },
      queryParamsHandling: 'merge',
    });
  });

  it('summarizes extra items', () => {
    const fixture = TestBed.createComponent(OrderListComponent);
    fixture.detectChanges();
    const component = fixture.componentInstance;

    expect(component.itemSummary(order)).toBe('Café');
    expect(
      component.itemSummary({
        ...order,
        items: [
          ...order.items,
          { ...order.items[0], productId: 'p2', name: 'Pão' },
          { ...order.items[0], productId: 'p3', name: 'Leite' },
          { ...order.items[0], productId: 'p4', name: 'Ovos' },
        ],
      }),
    ).toBe('Café, Pão, Leite e mais 1');
  });

  it('clamps invalid page to 1', () => {
    TestBed.overrideProvider(ActivatedRoute, { useValue: { queryParams: of({ page: 'abc' }) } });
    const fixture = TestBed.createComponent(OrderListComponent);
    fixture.detectChanges();
    expect(ordersApi.list).toHaveBeenCalledWith(1, 5);
  });

  it('loads the page from query params', () => {
    TestBed.overrideProvider(ActivatedRoute, { useValue: { queryParams: of({ page: '3' }) } });
    const fixture = TestBed.createComponent(OrderListComponent);
    fixture.detectChanges();
    expect(ordersApi.list).toHaveBeenCalledWith(3, 5);
  });

  it('sets error on load failure', () => {
    ordersApi.list.mockReturnValue(throwError(() => ({ message: 'orders-fail' })));
    const fixture = TestBed.createComponent(OrderListComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.error()).toBe('orders-fail');
    expect(fixture.componentInstance.loading()).toBe(false);
  });

  it('goShopping navigates to products', () => {
    const fixture = TestBed.createComponent(OrderListComponent);
    fixture.detectChanges();
    fixture.componentInstance.goShopping();
    expect(router.navigate).toHaveBeenCalledWith(['/products']);
  });

  it('formatDate renders pt-BR date', () => {
    const fixture = TestBed.createComponent(OrderListComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.formatDate('2026-06-15T12:00:00Z')).toContain('2026');
  });

  it('itemSummary handles empty items', () => {
    const fixture = TestBed.createComponent(OrderListComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.itemSummary({ ...order, items: [] })).toBe('');
  });
});
