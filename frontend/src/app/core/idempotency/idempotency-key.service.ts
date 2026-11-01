import { Injectable } from '@angular/core';

const CHECKOUT_KEY_PREFIX = 'mm_checkout_';
const PAYMENT_KEY_PREFIX = 'mm_payment_';

@Injectable({ providedIn: 'root' })
export class IdempotencyKeyService {
  private generateKey(): string {
    return crypto.randomUUID();
  }

  getCheckoutKey(userId: string, cartId: string): string {
    const key = `${CHECKOUT_KEY_PREFIX}${userId}_${cartId}`;
    const stored = sessionStorage.getItem(key);
    if (stored) return stored;
    const newKey = this.generateKey();
    sessionStorage.setItem(key, newKey);
    return newKey;
  }

  clearCheckoutKey(userId: string, cartId: string): void {
    const key = `${CHECKOUT_KEY_PREFIX}${userId}_${cartId}`;
    sessionStorage.removeItem(key);
  }

  getPaymentKey(orderId: string): string {
    const key = `${PAYMENT_KEY_PREFIX}${orderId}`;
    const stored = sessionStorage.getItem(key);
    if (stored) return stored;
    const newKey = this.generateKey();
    sessionStorage.setItem(key, newKey);
    return newKey;
  }

  clearPaymentKey(orderId: string): void {
    const key = `${PAYMENT_KEY_PREFIX}${orderId}`;
    sessionStorage.removeItem(key);
  }

  // newQuoteCycleKey returns the key prefix for one round of shipping quotes.
  // Every quote of the round derives its own key from it, so a retry of the
  // round replays the same quotes instead of pricing them again.
  newQuoteCycleKey(): string {
    return this.generateKey();
  }
}
