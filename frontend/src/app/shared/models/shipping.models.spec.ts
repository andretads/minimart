import { SHIPMENT_STATUS_LABELS } from './shipping.models';

describe('SHIPMENT_STATUS_LABELS', () => {
  it('has all statuses', () => {
    expect(SHIPMENT_STATUS_LABELS.pending).toBe('Aguardando processamento');
    expect(SHIPMENT_STATUS_LABELS.processing).toBe('Em processamento');
    expect(SHIPMENT_STATUS_LABELS.shipped).toBe('Enviado');
    expect(SHIPMENT_STATUS_LABELS.delivered).toBe('Entregue');
    expect(SHIPMENT_STATUS_LABELS.failed).toBe('Falhou');
  });
});
