import { TestBed } from '@angular/core/testing';
import { Observable, of, throwError } from 'rxjs';
import { vi } from 'vitest';
import { ActivatedRoute, ParamMap, Router, convertToParamMap } from '@angular/router';
import { PaymentStatusComponent } from './payment-status.component';
import { PaymentsApiService } from '@features/payments/payments-api.service';
import { AuthStore } from '@core/auth/auth.store';
import { ToastService } from '@core/toast/toast.service';
import { Payment } from '@shared/models/payment.models';

const payment = (status: Payment['status']): Payment => ({
  id: 'pay-1',
  orderId: 'o1',
  userId: 'u1',
  amountCents: 500,
  currency: 'BRL',
  method: 'card',
  status,
  provider: 'stripe',
  providerRef: 'pi',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
});

describe('PaymentStatusComponent', () => {
  let api: { get: ReturnType<typeof vi.fn> };
  let route: { paramMap: Observable<ParamMap>; snapshot: { paramMap: ParamMap } };
  let router: { navigate: ReturnType<typeof vi.fn> };
  let auth: { userId: ReturnType<typeof vi.fn> };
  let toast: { success: ReturnType<typeof vi.fn>; error: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    api = { get: vi.fn(() => of(payment('succeeded'))) };
    route = {
      paramMap: of(convertToParamMap({ id: 'pay-1' })),
      snapshot: { paramMap: convertToParamMap({ id: 'pay-1' }) },
    };
    router = { navigate: vi.fn(() => Promise.resolve(true)) };
    auth = { userId: vi.fn(() => 'u1') };
    toast = { success: vi.fn(), error: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [PaymentStatusComponent],
      providers: [
        { provide: PaymentsApiService, useValue: api },
        { provide: ActivatedRoute, useValue: route },
        { provide: Router, useValue: router },
        { provide: AuthStore, useValue: auth },
        { provide: ToastService, useValue: toast },
      ],
    }).compileComponents();
  });

  it('loads a succeeded payment', () => {
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    expect(api.get).toHaveBeenCalledWith('pay-1');
    expect(f.componentInstance.payment()?.status).toBe('succeeded');
    expect(f.componentInstance.autoRefreshActive()).toBe(false);
    expect(f.componentInstance.loading()).toBe(false);
  });

  it('rejects payment belonging to another user', () => {
    auth.userId.mockReturnValue('other');
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    expect(f.componentInstance.error()).toBe('Pagamento não encontrado');
    expect(f.componentInstance.payment()).toBeNull();
  });

  it('sets error on load failure', () => {
    api.get.mockReturnValue(throwError(() => ({ message: 'not-found' })));
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    expect(f.componentInstance.error()).toBe('not-found');
  });

  it('activates auto-refresh for pending payments', () => {
    api.get.mockReturnValue(of(payment('pending')));
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    expect(f.componentInstance.autoRefreshActive()).toBe(true);
  });

  it('status computeds', () => {
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    expect(f.componentInstance.statusLabel()).toBeTruthy();
    expect(f.componentInstance.paymentStatusClass()).toBe('succeeded');
    expect(f.componentInstance.paymentIcon()).toBe('icon icon-lg icon-pay-succeeded');
    expect(f.componentInstance.statusDescription()).toContain('sucesso');
  });

  it('computeds default when no payment', () => {
    api.get.mockReturnValue(throwError(() => new Error('x')));
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    expect(f.componentInstance.statusLabel()).toBe('');
    expect(f.componentInstance.paymentStatusClass()).toBe('');
    expect(f.componentInstance.paymentIcon()).toBe('icon');
    expect(f.componentInstance.statusDescription()).toBe('');
  });

  it('statusDescription covers all statuses', () => {
    const cases: [Payment['status'], string][] = [
      ['pending', 'processado'],
      ['succeeded', 'sucesso'],
      ['failed', 'recusado'],
      ['unknown', 'incerto'],
      ['refunded', 'reembolsado'],
    ];
    for (const [status, fragment] of cases) {
      api.get.mockReturnValue(of(payment(status)));
      const f = TestBed.createComponent(PaymentStatusComponent);
      f.detectChanges();
      expect(f.componentInstance.statusDescription()).toContain(fragment);
      f.destroy();
    }
  });

  it('refresh re-fetches payment', () => {
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    api.get.mockClear();
    f.componentInstance.refresh();
    expect(api.get).toHaveBeenCalledWith('pay-1');
  });

  it('refresh no-ops without payment', () => {
    api.get.mockReturnValue(throwError(() => new Error('x')));
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    api.get.mockClear();
    f.componentInstance.refresh();
    expect(api.get).not.toHaveBeenCalled();
  });

  it('refresh toasts error on failure', () => {
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    api.get.mockReturnValue(throwError(() => ({ message: 'nope' })));
    f.componentInstance.refresh();
    expect(toast.error).toHaveBeenCalledWith('nope');
  });

  it('formatDate renders pt-BR', () => {
    const f = TestBed.createComponent(PaymentStatusComponent);
    f.detectChanges();
    expect(f.componentInstance.formatDate('2026-01-01T12:00:00Z')).toContain('2026');
  });
});
