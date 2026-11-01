export interface ShippingMethod {
  id: string;
  name: string;
  description: string;
  active: boolean;
  estimatedMinDays: number;
  estimatedMaxDays: number;
  createdAt: string;
  updatedAt: string;
}

export interface ShippingZone {
  id: string;
  name: string;
  country: string;
  state: string;
  city: string;
  postalCodeStart: string;
  postalCodeEnd: string;
  createdAt: string;
  updatedAt: string;
}

export interface ShippingRate {
  id: string;
  zoneId: string;
  methodId: string;
  baseCostCents: number;
  freeThresholdCents: number;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface DestinationSnapshot {
  label: string;
  fullName: string;
  street: string;
  city: string;
  state: string;
  zip: string;
  country: string;
  phone?: string;
}

export interface ShippingQuote {
  id: string;
  customerId: string;
  methodId: string;
  destinationSnapshot: DestinationSnapshot;
  shippingCostCents: number;
  subtotalCents: number;
  freeThresholdCents: number;
  estimatedMinDays: number;
  estimatedMaxDays: number;
  status: 'active' | 'expired' | 'consumed';
  expiresAt: string;
  createdAt: string;
}

export interface Shipment {
  id: string;
  orderId: string;
  userId: string;
  methodId: string;
  trackingCode: string;
  carrier: string;
  shippingCostCents: number;
  destinationSnapshot: DestinationSnapshot;
  status: 'pending' | 'processing' | 'shipped' | 'delivered' | 'failed';
  shippedAt?: string;
  deliveredAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface TrackingEvent {
  id: string;
  shipmentId: string;
  status: string;
  description: string;
  location: string;
  occurredAt: string;
  createdAt: string;
}

export const SHIPMENT_STATUS_LABELS: Record<Shipment['status'], string> = {
  pending: 'Aguardando processamento',
  processing: 'Em processamento',
  shipped: 'Enviado',
  delivered: 'Entregue',
  failed: 'Falhou',
};
