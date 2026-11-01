import { Injectable, signal, computed, inject, effect } from '@angular/core';
import { WishlistService } from './wishlist.service';
import { Wishlist } from '@shared/models/wishlist.models';
import { AuthStore } from '@core/auth/auth.store';

@Injectable({ providedIn: 'root' })
export class WishlistStore {
  private readonly service = inject(WishlistService);
  private readonly authStore = inject(AuthStore);

  readonly wishlist = signal<Wishlist | null>(null);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);

  private readonly pending = signal<Set<string>>(new Set());
  private loadedForUserId: string | null = null;

  readonly productIds = computed(
    () => new Set(this.wishlist()?.items.map((i) => i.productId) ?? []),
  );

  constructor() {
    effect(() => {
      const userId = this.authStore.userId();
      if (!userId) {
        if (this.wishlist() !== null || this.error() !== null) {
          this.wishlist.set(null);
          this.error.set(null);
        }
        this.loading.set(false);
        this.loadedForUserId = null;
        return;
      }
      if (this.loadedForUserId !== userId) {
        this.load(userId);
      }
    });
  }

  isInWishlist(productId: string): boolean {
    return this.productIds().has(productId);
  }

  load(userId?: string): void {
    const resolvedUserId = userId ?? this.authStore.userId();
    if (!resolvedUserId) {
      return;
    }
    this.loading.set(true);
    this.error.set(null);
    this.service.getWishlist().subscribe({
      next: (wishlist: Wishlist) => {
        this.wishlist.set(wishlist);
        this.loadedForUserId = resolvedUserId;
        this.loading.set(false);
      },
      error: (err: Error) => {
        this.error.set(err.message || 'Failed to load wishlist');
        this.loading.set(false);
      },
    });
  }

  add(productId: string): void {
    if (this.pending().has(productId)) {
      return;
    }
    this.markPending(productId, true);
    this.service.addItem(productId).subscribe({
      next: (wishlist: Wishlist) => {
        this.wishlist.set(wishlist);
        this.markPending(productId, false);
      },
      error: (err: Error) => {
        this.error.set(err.message || 'Failed to add to wishlist');
        this.markPending(productId, false);
        // Resync with server in case of conflict (already present).
        if (this.isApiConflict(err)) {
          this.load();
        }
      },
    });
  }

  remove(productId: string): void {
    if (this.pending().has(productId)) {
      return;
    }
    this.markPending(productId, true);
    this.service.removeItem(productId).subscribe({
      next: (wishlist: Wishlist) => {
        this.wishlist.set(wishlist);
        this.markPending(productId, false);
      },
      error: (err: Error) => {
        this.error.set(err.message || 'Failed to remove from wishlist');
        this.markPending(productId, false);
        // Resync with server in case the item is already gone.
        if (this.isApiNotFound(err)) {
          this.load();
        }
      },
    });
  }

  clear(): void {
    this.service.clearWishlist().subscribe({
      next: () => this.wishlist.update((w) => (w ? { ...w, items: [] } : null)),
      error: (err: Error) => this.error.set(err.message || 'Failed to clear wishlist'),
    });
  }

  toggle(productId: string): void {
    if (this.wishlist() === null) {
      this.load();
      // Defer decision until load resolves: re-run toggle after load by checking again shortly.
      // Simpler: optimistically decide based on empty state; load will correct via effect.
    }
    if (this.productIds().has(productId)) {
      this.remove(productId);
    } else {
      this.add(productId);
    }
  }

  private markPending(productId: string, add: boolean): void {
    this.pending.update((set) => {
      const next = new Set(set);
      if (add) {
        next.add(productId);
      } else {
        next.delete(productId);
      }
      return next;
    });
  }

  private isApiConflict(err: Error): boolean {
    return (err as { status?: number }).status === 409;
  }

  private isApiNotFound(err: Error): boolean {
    return (err as { status?: number }).status === 404;
  }
}
