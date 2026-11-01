import { describe, it, expect } from 'vitest';
import { Order } from '../models/order.models';
import { AUTO_CANCEL_WINDOW_MS, autoCancelAt, formatRemaining } from './order-deadline';

const order: Order = {
  id: 'o1',
  userId: 'u1',
  items: [],
  totalCents: 1000,
  status: 'placed',
  createdAt: '2026-01-02T10:00:00Z',
};

describe('order-deadline', () => {
  it('AUTO_CANCEL_WINDOW_MS is five minutes', () => {
    expect(AUTO_CANCEL_WINDOW_MS).toBe(5 * 60 * 1000);
  });

  it('autoCancelAt falls back to createdAt plus window', () => {
    const deadline = autoCancelAt(order);
    expect(deadline.getTime()).toBe(new Date(order.createdAt).getTime() + AUTO_CANCEL_WINDOW_MS);
  });

  it('autoCancelAt uses provided deadline', () => {
    const deadlineIso = '2026-01-02T10:10:00Z';
    const deadline = autoCancelAt({ ...order, autoCancelAt: deadlineIso });
    expect(deadline.toISOString()).toBe(new Date(deadlineIso).toISOString());
  });

  it('formatRemaining pads minutes and seconds', () => {
    expect(formatRemaining(299000)).toBe('04:59');
    expect(formatRemaining(0)).toBe('00:00');
    expect(formatRemaining(-1000)).toBe('00:00');
  });
});
