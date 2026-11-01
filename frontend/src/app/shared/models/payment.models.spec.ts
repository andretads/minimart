import { PAYMENT_STATUS_LABELS } from './payment.models';

describe('PAYMENT_STATUS_LABELS', () => {
  it('has all statuses', () => {
    expect(PAYMENT_STATUS_LABELS.pending).toBe('Pendente');
    expect(PAYMENT_STATUS_LABELS.succeeded).toBe('Aprovado');
    expect(PAYMENT_STATUS_LABELS.failed).toBe('Reprovado');
    expect(PAYMENT_STATUS_LABELS.unknown).toBe('Desconhecido');
    expect(PAYMENT_STATUS_LABELS.refunded).toBe('Reembolsado');
  });
});
