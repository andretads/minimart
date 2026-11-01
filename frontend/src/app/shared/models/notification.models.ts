export interface Notification {
  id: string;
  type:
    | 'order_confirmation'
    | 'order_cancelled'
    | 'payment_succeeded'
    | 'payment_failed'
    | 'shipment_created'
    | 'shipment_shipped'
    | 'shipment_delivered';
  userId: string;
  orderId: string;
  subject: string;
  body: string;
  sentAt: string;
  createdAt: string;
  readAt?: string;
  status?: string;
}

export type NotificationType = Notification['type'];

export interface NotificationPreferences {
  emailEnabled: boolean;
  smsEnabled: boolean;
  pushEnabled: boolean;
  marketingEnabled: boolean;
  updatedAt?: string;
}

export const NOTIFICATION_TYPE_LABELS: Record<NotificationType, string> = {
  order_confirmation: 'Confirmação do pedido',
  order_cancelled: 'Pedido cancelado',
  payment_succeeded: 'Pagamento aprovado',
  payment_failed: 'Pagamento recusado',
  shipment_created: 'Envio criado',
  shipment_shipped: 'Envio despachado',
  shipment_delivered: 'Envio entregue',
};
