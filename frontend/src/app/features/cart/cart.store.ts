import { Injectable, inject, signal, computed, effect } from '@angular/core';
import { Cart } from '@shared/models/cart.models';
import { AuthStore } from '@core/auth/auth.store';

@Injectable({ providedIn: 'root' })
export class CartStore {
  private readonly authStore = inject(AuthStore);

  cart = signal<Cart | null>(null);
  loading = signal(false);
  error = signal<string | null>(null);

  itemCount = computed(() => this.cart()?.items.reduce((sum, item) => sum + item.quantity, 0) ?? 0);
  estimatedTotal = computed(() => this.cart()?.estimatedTotalCents ?? 0);
  isEmpty = computed(() => this.itemCount() === 0);

  constructor() {
    effect(() => {
      if (!this.authStore.isAuthenticated()) {
        this.cart.set(null);
      }
    });
  }

  setCart(cart: Cart | null): void {
    this.cart.set(cart);
  }

  setLoading(loading: boolean): void {
    this.loading.set(loading);
  }

  setError(error: string | null): void {
    this.error.set(error);
  }

  clear(): void {
    this.cart.set(null);
    this.error.set(null);
  }
}
