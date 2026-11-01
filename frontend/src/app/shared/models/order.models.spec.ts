import { ORDER_STATUS_LABELS, CANCELLABLE_STATUSES } from './order.models';

describe('order models constants', () => {
  it('ORDER_STATUS_LABELS', () => {
    expect(ORDER_STATUS_LABELS.placed).toBe('Aguardando pagamento');
    expect(ORDER_STATUS_LABELS.paid).toBe('Pagamento confirmado');
    expect(ORDER_STATUS_LABELS.payment_failed).toBe('Pagamento não aprovado');
    expect(ORDER_STATUS_LABELS.payment_unknown).toBe('Aguardando confirmação');
    expect(ORDER_STATUS_LABELS.cancelled).toBe('Cancelado');
  });

  it('CANCELLABLE_STATUSES', () => {
    expect(CANCELLABLE_STATUSES).toEqual(['placed', 'payment_failed', 'payment_unknown']);
  });
});
