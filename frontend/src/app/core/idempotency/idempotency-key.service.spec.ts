import { TestBed } from '@angular/core/testing';
import { IdempotencyKeyService } from './idempotency-key.service';

describe('IdempotencyKeyService', () => {
  let service: IdempotencyKeyService;

  beforeEach(() => {
    sessionStorage.clear();
    vi.spyOn(crypto, 'randomUUID').mockReturnValue(
      'fixed-uuid' as unknown as ReturnType<typeof crypto.randomUUID>,
    );
    TestBed.configureTestingModule({});
    service = TestBed.inject(IdempotencyKeyService);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    sessionStorage.clear();
  });

  it('generates and persists a checkout key', () => {
    expect(service.getCheckoutKey('u1', 'c1')).toBe('fixed-uuid');
    expect(sessionStorage.getItem('mm_checkout_u1_c1')).toBe('fixed-uuid');
  });

  it('reuses the persisted checkout key without generating a new one', () => {
    sessionStorage.setItem('mm_checkout_u1_c1', 'existing-key');

    expect(service.getCheckoutKey('u1', 'c1')).toBe('existing-key');
    expect(crypto.randomUUID).not.toHaveBeenCalled();
  });

  it('clearCheckoutKey removes the entry', () => {
    service.getCheckoutKey('u1', 'c1');
    service.clearCheckoutKey('u1', 'c1');
    expect(sessionStorage.getItem('mm_checkout_u1_c1')).toBeNull();
  });

  it('generates and persists a payment key', () => {
    expect(service.getPaymentKey('o1')).toBe('fixed-uuid');
    expect(sessionStorage.getItem('mm_payment_o1')).toBe('fixed-uuid');
  });

  it('reuses the persisted payment key', () => {
    sessionStorage.setItem('mm_payment_o1', 'existing-pay');
    expect(service.getPaymentKey('o1')).toBe('existing-pay');
  });

  it('clearPaymentKey removes the entry', () => {
    service.getPaymentKey('o1');
    service.clearPaymentKey('o1');
    expect(sessionStorage.getItem('mm_payment_o1')).toBeNull();
  });

  it('newQuoteCycleKey returns a fresh uuid without touching storage', () => {
    const before = sessionStorage.length;
    const key = service.newQuoteCycleKey();
    expect(key).toBe('fixed-uuid');
    expect(sessionStorage.length).toBe(before);
  });
});
