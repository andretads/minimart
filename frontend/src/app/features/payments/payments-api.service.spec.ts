import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { ApiClient } from '../../core/api/api-client';
import { AuthStore } from '../../core/auth/auth.store';
import { Payment, CreatePaymentRequest } from '../../shared/models/payment.models';
import { PaymentsApiService } from './payments-api.service';

const payment: Payment = {
  id: 'pay-1',
  orderId: 'o1',
  userId: 'u1',
  amountCents: 5000,
  currency: 'BRL',
  method: 'card',
  status: 'pending',
  provider: 'stripe',
  providerRef: 'pi_123',
  failureReason: undefined,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const createRequest: CreatePaymentRequest = {
  orderId: 'o1',
  userId: 'u1',
  amountCents: 5000,
  currency: 'BRL',
  method: 'card',
};

describe('PaymentsApiService', () => {
  let service: PaymentsApiService;
  const api = { get: vi.fn(), post: vi.fn() };

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        PaymentsApiService,
        { provide: ApiClient, useValue: api },
        { provide: AuthStore, useValue: { userId: () => 'u1' } },
      ],
    });
    service = TestBed.inject(PaymentsApiService);
    api.get.mockReset();
    api.post.mockReset();
  });

  it('creates a payment with Idempotency-Key header', () => {
    api.post.mockReturnValue(of(payment));

    let result: Payment | undefined;
    service.create(createRequest, 'idem-pay-456').subscribe((value) => (result = value));

    expect(api.post).toHaveBeenCalledWith('/v1/payments', createRequest, {
      headers: { 'Idempotency-Key': 'idem-pay-456' },
    });
    expect(result).toEqual(payment);
  });

  it('gets a payment by id', () => {
    api.get.mockReturnValue(of(payment));

    let result: Payment | undefined;
    service.get('pay-1').subscribe((value) => (result = value));

    expect(api.get).toHaveBeenCalledWith('/v1/payments/pay-1');
    expect(result).toEqual(payment);
  });
});
