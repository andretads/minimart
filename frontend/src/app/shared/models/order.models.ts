import { DestinationSnapshot } from './shipping.models';

export interface OrderItem {
  productId: string;
  name: string;
  unitPriceCents: number;
  quantity: number;
  lineTotalCents: number;
}

export interface Order {
  id: string;
  userId: string;
  items: OrderItem[];
  totalCents: number;
  status: 'placed' | 'paid' | 'payment_failed' | 'payment_unknown' | 'cancelled';
  cancelledAt?: string;
  createdAt: string;
  autoCancelAt?: string;
  // Shipping information
  shippingQuoteId?: string;
  shippingMethodId?: string;
  shippingCostCents?: number;
  shippingMethodName?: string;
  destinationSnapshot?: DestinationSnapshot;
}

export type OrderStatus = Order['status'];

export interface OrderListResponse {
  data: Order[];
  meta: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}

export const ORDER_STATUS_LABELS: Record<OrderStatus, string> = {
  placed: 'Aguardando pagamento',
  paid: 'Pagamento confirmado',
  payment_failed: 'Pagamento não aprovado',
  payment_unknown: 'Aguardando confirmação',
  cancelled: 'Cancelado',
};

export const CANCELLABLE_STATUSES: OrderStatus[] = ['placed', 'payment_failed', 'payment_unknown'];
