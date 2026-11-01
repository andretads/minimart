import { TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { AuthStore } from '@core/auth/auth.store';
import { IdempotencyKeyService } from '@core/idempotency/idempotency-key.service';
import { ToastService } from '@core/toast/toast.service';
import { PaymentsApiService } from '@features/payments/payments-api.service';
import { ShippingApiService } from '@features/shipping/shipping-api.service';
import { Order } from '@shared/models/order.models';
import { Payment } from '@shared/models/payment.models';
import { Shipment, TrackingEvent } from '@shared/models/shipping.models';
import { OrdersApiService } from '../orders-api.service';
import { OrderDetailsComponent } from './order-details.component';

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
  totalCents: 1250,
  status: 'paid',
  createdAt: '2026-01-02T10:00:00Z',
  shippingQuoteId: 'quote-1',
  shippingMethodId: 'method-1',
  shippingMethodName: 'Sedex',
  shippingCostCents: 250,
  destinationSnapshot: {
    label: 'Casa',
    fullName: 'Ana Silva',
    street: 'Rua das Flores, 10',
    city: 'São Paulo',
    state: 'SP',
    zip: '01001000',
    country: 'BR',
  },
};

const shipment: Shipment = {
  id: 'ship-1',
  orderId: 'order-1234',
  userId: 'u1',
  methodId: 'method-1',
  trackingCode: 'BR123',
  carrier: 'Correios',
  shippingCostCents: 250,
  destinationSnapshot: order.destinationSnapshot!,
  status: 'shipped',
  shippedAt: '2026-01-03T10:00:00Z',
  createdAt: '2026-01-02T12:00:00Z',
  updatedAt: '2026-01-03T10:00:00Z',
};

const trackingEvent: TrackingEvent = {
  id: 'trk-1',
  shipmentId: 'ship-1',
  status: 'shipped',
  description: 'Objeto enviado',
  location: 'São Paulo',
  occurredAt: '2026-01-03T10:00:00Z',
  createdAt: '2026-01-03T10:00:00Z',
};

const payment: Payment = {
  id: 'pay-1',
  orderId: 'order-1234',
  userId: 'u1',
  amountCents: 1250,
  currency: 'BRL',
  method: 'card',
  status: 'succeeded',
  provider: 'stripe',
  createdAt: '2026-01-02T10:00:00Z',
  updatedAt: '2026-01-02T10:05:00Z',
};

describe('OrderDetailsComponent', () => {
  let ordersApi: { get: ReturnType<typeof vi.fn>; cancel: ReturnType<typeof vi.fn> };
  let shippingApi: {
    getShipments: ReturnType<typeof vi.fn>;
    getTrackingEvents: ReturnType<typeof vi.fn>;
  };
  let paymentsApi: { get: ReturnType<typeof vi.fn>; create: ReturnType<typeof vi.fn> };
  let toast: { error: ReturnType<typeof vi.fn>; success: ReturnType<typeof vi.fn> };
  let router: { navigate: ReturnType<typeof vi.fn> };
  let authStore: { userId: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    sessionStorage.clear();
    ordersApi = {
      get: vi.fn(() => of(order)),
      cancel: vi.fn(() => of({ ...order, status: 'cancelled' })),
    };
    shippingApi = {
      getShipments: vi.fn(() => of([shipment])),
      getTrackingEvents: vi.fn(() => of([trackingEvent])),
    };
    paymentsApi = { get: vi.fn(() => of(payment)), create: vi.fn(() => of(payment)) };
    toast = { error: vi.fn(), success: vi.fn() };
    router = { navigate: vi.fn() };
    authStore = { userId: vi.fn(() => 'u1') };

    await TestBed.configureTestingModule({
      imports: [OrderDetailsComponent],
      providers: [
        { provide: OrdersApiService, useValue: ordersApi },
        { provide: PaymentsApiService, useValue: paymentsApi },
        { provide: ShippingApiService, useValue: shippingApi },
        { provide: AuthStore, useValue: authStore },
        { provide: IdempotencyKeyService, useValue: { getPaymentKey: vi.fn(() => 'idem-key') } },
        { provide: ToastService, useValue: toast },
        { provide: Router, useValue: router },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { paramMap: { get: vi.fn(() => 'order-1234') } },
            paramMap: of({ get: vi.fn(() => 'order-1234') }),
          },
        },
      ],
    }).compileComponents();
  });

  it('renders the shipping snapshot carried by the order', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Entrega');
    expect(text).toContain('Sedex');
    expect(text).toContain('Ana Silva');
    expect(text).toContain('Rua das Flores, 10');
  });

  it('loads the shipment for the order and shows its tracking history', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();

    expect(shippingApi.getShipments).toHaveBeenCalledWith('order-1234');
    expect(shippingApi.getTrackingEvents).toHaveBeenCalledWith('ship-1');

    const loaded = fixture.componentInstance.shipment()!;
    expect(loaded.id).toBe('ship-1');
    expect(loaded.status).toBe('shipped');
    expect(loaded.trackingCode).toBe('BR123');
    expect(loaded.carrier).toBe('Correios');

    const events = fixture.componentInstance.trackingEvents();
    expect(events.length).toBe(1);
    expect(events[0].description).toBe('Objeto enviado');

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('BR123');
    expect(text).toContain('Objeto enviado');
  });

  it('shows the shipping snapshot of an order that has no shipment yet', () => {
    shippingApi.getShipments.mockReturnValue(of([]));

    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.shipment()).toBeNull();
    expect(fixture.componentInstance.trackingEvents()).toEqual([]);
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Sedex');
    expect(text).not.toContain('Código de rastreio');
  });

  it('clears the shipment when the loaded order has none', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.shipment()).not.toBeNull();

    shippingApi.getShipments.mockReturnValue(of([]));
    fixture.componentInstance.loadOrder();
    fixture.detectChanges();

    expect(fixture.componentInstance.shipment()).toBeNull();
    expect(fixture.componentInstance.trackingEvents()).toEqual([]);
    const text = fixture.nativeElement.textContent as string;
    expect(text).not.toContain('Código de rastreio');
  });

  it('keeps the shipment when the order is reloaded', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();

    fixture.componentInstance.payment.set(payment);
    fixture.componentInstance.refreshPayment();
    fixture.detectChanges();

    expect(fixture.componentInstance.shipment()!.trackingCode).toBe('BR123');
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('BR123');
  });

  it('fetches the shipment when a reload finds the order paid', () => {
    shippingApi.getShipments.mockReturnValue(of([]));

    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.shipment()).toBeNull();

    shippingApi.getShipments.mockClear();
    ordersApi.get.mockReturnValue(of({ ...order, status: 'paid' }));
    fixture.componentInstance['reloadOrderSilently']();

    expect(shippingApi.getShipments).toHaveBeenCalledWith('order-1234');
  });

  it('rejects an order belonging to another user', () => {
    authStore.userId.mockReturnValue('other');
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.error()).toBe('Pedido não encontrado');
    expect(fixture.componentInstance.order()).toBeNull();
  });

  it('sets error when order load fails', () => {
    ordersApi.get.mockReturnValue(throwError(() => ({ message: 'gone' })));
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.error()).toBe('gone');
    expect(fixture.componentInstance.loading()).toBe(false);
  });

  it('canPay true for placed order without payment', () => {
    ordersApi.get.mockReturnValue(of({ ...order, status: 'placed' }));
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.canPay()).toBe(true);
  });

  it('canPay false when a payment exists', () => {
    ordersApi.get.mockReturnValue(of({ ...order, status: 'placed' }));
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    fixture.componentInstance.payment.set(payment);
    expect(fixture.componentInstance.canPay()).toBe(false);
  });

  it('canCancel reflects cancellable status', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.canCancel()).toBe(false);
  });

  it('orderStatusLabel and paymentStatusLabel', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    fixture.componentInstance.payment.set(payment);
    expect(fixture.componentInstance.orderStatusLabel()).toBeTruthy();
    expect(fixture.componentInstance.paymentStatusLabel()).toBeTruthy();
  });

  it('createPayment creates, stores id and toasts success', () => {
    ordersApi.get.mockReturnValue(of({ ...order, status: 'placed' }));
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    fixture.componentInstance.createPayment();
    expect(paymentsApi.create).toHaveBeenCalled();
    expect(sessionStorage.getItem('mm_payment_order-1234')).toBe('pay-1');
    expect(toast.success).toHaveBeenCalled();
    expect(fixture.componentInstance.creatingPayment()).toBe(false);
  });

  it('createPayment toasts error on failure', () => {
    ordersApi.get.mockReturnValue(of({ ...order, status: 'placed' }));
    paymentsApi.create.mockReturnValue(throwError(() => ({ message: 'pay-fail' })));
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    fixture.componentInstance.createPayment();
    expect(toast.error).toHaveBeenCalledWith('pay-fail');
  });

  it('refreshPayment no-ops without payment', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    paymentsApi.get.mockClear();
    fixture.componentInstance.refreshPayment();
    expect(paymentsApi.get).not.toHaveBeenCalled();
  });

  it('refreshPayment toasts error on failure', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    fixture.componentInstance.payment.set(payment);
    paymentsApi.get.mockReturnValue(throwError(() => ({ message: 'ref-fail' })));
    fixture.componentInstance.refreshPayment();
    expect(toast.error).toHaveBeenCalledWith('ref-fail');
  });

  it('cancelOrder cancels and toasts success', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    fixture.componentInstance.cancelOrder();
    expect(ordersApi.cancel).toHaveBeenCalledWith('order-1234');
    expect(toast.success).toHaveBeenCalledWith('Pedido cancelado');
    expect(fixture.componentInstance.confirmCancel()).toBe(false);
  });

  it('cancelOrder toasts error on failure', () => {
    ordersApi.cancel.mockReturnValue(throwError(() => ({ message: 'cancel-fail' })));
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    fixture.componentInstance.cancelOrder();
    expect(toast.error).toHaveBeenCalledWith('cancel-fail');
  });

  it('loadPaymentForOrder restores payment from sessionStorage', () => {
    sessionStorage.setItem('mm_payment_order-1234', 'pay-1');
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    expect(paymentsApi.get).toHaveBeenCalledWith('pay-1');
    expect(fixture.componentInstance.payment()?.id).toBe('pay-1');
    sessionStorage.removeItem('mm_payment_order-1234');
  });

  it('format helpers', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.formatDate('2026-06-15T12:00:00Z')).toContain('2026');
    expect(fixture.componentInstance.formatDateTime('2026-06-15T12:00:00Z')).toContain('2026');
    expect(fixture.componentInstance.formatAddress(undefined)).toBe('Não informado');
    expect(fixture.componentInstance.formatAddress(order.destinationSnapshot)).toContain(
      'Ana Silva',
    );
  });

  it('status label helpers', () => {
    const fixture = TestBed.createComponent(OrderDetailsComponent);
    fixture.detectChanges();
    expect(fixture.componentInstance.getShipmentStatusLabel('shipped')).toBeTruthy();
    expect(fixture.componentInstance.getShipmentStatusLabel('weird')).toBe('weird');
    expect(fixture.componentInstance.getTrackingStatusLabel('shipped')).toBe('Enviado');
    expect(fixture.componentInstance.getTrackingStatusLabel('weird')).toBe('weird');
    expect(fixture.componentInstance.isEventCompleted('delivered')).toBe(true);
    expect(fixture.componentInstance.isEventCompleted('pending')).toBe(false);
  });
});
