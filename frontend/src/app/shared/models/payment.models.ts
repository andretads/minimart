export interface Payment {
  id: string;
  orderId: string;
  userId: string;
  amountCents: number;
  currency: string;
  method: string;
  status: 'pending' | 'succeeded' | 'failed' | 'unknown' | 'refunded';
  provider: string;
  providerRef?: string;
  failureReason?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreatePaymentRequest {
  orderId: string;
  userId: string;
  amountCents: number;
  currency: string;
  method: string;
}

export type PaymentStatus = Payment['status'];

export const PAYMENT_STATUS_LABELS: Record<PaymentStatus, string> = {
  pending: 'Pendente',
  succeeded: 'Aprovado',
  failed: 'Reprovado',
  unknown: 'Desconhecido',
  refunded: 'Reembolsado',
};
