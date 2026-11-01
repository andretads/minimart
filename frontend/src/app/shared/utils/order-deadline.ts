import { Order } from '../models/order.models';

export const AUTO_CANCEL_WINDOW_MS = 5 * 60 * 1000;

export function autoCancelAt(order: Order): Date {
  if (order.autoCancelAt) {
    return new Date(order.autoCancelAt);
  }
  return new Date(new Date(order.createdAt).getTime() + AUTO_CANCEL_WINDOW_MS);
}

export function formatRemaining(ms: number): string {
  const safe = Math.max(0, Math.floor(ms));
  const minutes = Math.floor(safe / 60000);
  const seconds = Math.floor((safe % 60000) / 1000);
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}
