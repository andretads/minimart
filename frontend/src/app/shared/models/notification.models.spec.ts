import { NOTIFICATION_TYPE_LABELS } from './notification.models';

describe('NOTIFICATION_TYPE_LABELS', () => {
  it('has all types', () => {
    expect(NOTIFICATION_TYPE_LABELS.order_confirmation).toBe('Confirmação do pedido');
    expect(NOTIFICATION_TYPE_LABELS.order_cancelled).toBe('Pedido cancelado');
    expect(NOTIFICATION_TYPE_LABELS.payment_succeeded).toBe('Pagamento aprovado');
    expect(NOTIFICATION_TYPE_LABELS.payment_failed).toBe('Pagamento recusado');
    expect(NOTIFICATION_TYPE_LABELS.shipment_created).toBe('Envio criado');
    expect(NOTIFICATION_TYPE_LABELS.shipment_shipped).toBe('Envio despachado');
    expect(NOTIFICATION_TYPE_LABELS.shipment_delivered).toBe('Envio entregue');
  });
});
