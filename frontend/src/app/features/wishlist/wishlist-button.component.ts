import { Component, input, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { WishlistStore } from './wishlist.store';
import { AuthStore } from '@core/auth/auth.store';

@Component({
  selector: 'app-wishlist-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    @if (isAuthenticated()) {
      <button
        type="button"
        class="wishlist-btn"
        (click)="toggle()"
        [attr.aria-pressed]="inWishlist()"
        [attr.aria-label]="
          inWishlist() ? 'Remover da lista de desejos' : 'Adicionar à lista de desejos'
        "
        [disabled]="loading()"
      >
        <span
          [class]="inWishlist() ? 'icon icon-sm icon-heart-filled' : 'icon icon-sm icon-heart'"
          aria-hidden="true"
        ></span>
      </button>
    } @else {
      <button
        type="button"
        class="wishlist-btn"
        (click)="promptLogin()"
        [attr.aria-label]="'Entre para adicionar à lista de desejos'"
      >
        <span class="icon icon-sm icon-heart" aria-hidden="true"></span>
      </button>
    }
  `,
  styles: [
    `
      .wishlist-btn {
        display: grid;
        place-items: center;
        width: 2.1rem;
        height: 2.1rem;
        border: 1px solid var(--border);
        border-radius: 50%;
        background: color-mix(in oklab, var(--background) 82%, transparent);
        color: var(--muted-foreground);
        cursor: pointer;
        -webkit-backdrop-filter: blur(8px);
        backdrop-filter: blur(8px);
        transition:
          color var(--transition-fast),
          background-color var(--transition-fast),
          transform var(--transition-fast);
      }
      .wishlist-btn:hover:not(:disabled) {
        color: var(--destructive);
        transform: scale(1.06);
      }
      .wishlist-btn:disabled {
        opacity: 0.6;
        cursor: wait;
      }
    `,
  ],
})
export class WishlistButtonComponent {
  productId = input.required<string>();

  private readonly store = inject(WishlistStore);
  private readonly authStore = inject(AuthStore);

  readonly isAuthenticated = this.authStore.isAuthenticated;
  readonly inWishlist = computed(() => this.store.isInWishlist(this.productId()));
  readonly loading = this.store.loading;

  toggle(): void {
    if (this.isAuthenticated()) {
      this.store.toggle(this.productId());
    }
  }

  promptLogin(): void {
    // Could emit an event or navigate to login
    // For now, just do nothing - the button is still clickable but doesn't do anything
  }
}
