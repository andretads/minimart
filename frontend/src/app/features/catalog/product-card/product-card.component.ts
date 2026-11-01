import { Component, input, output, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { Product } from '@shared/models/catalog.models';
import { ProductImageComponent } from '@shared/components/product-image/product-image.component';
import { MoneyPipe } from '@shared/pipes/money.pipe';
import { WishlistStore } from '@features/wishlist/wishlist.store';
import { AuthStore } from '@core/auth/auth.store';

@Component({
  selector: 'app-product-card',
  standalone: true,
  imports: [CommonModule, RouterLink, ProductImageComponent, MoneyPipe],
  template: `
    @if (loading()) {
      <div class="card-skeleton" aria-hidden="true">
        <div class="skeleton-media"></div>
        <div class="skeleton-line w-70"></div>
        <div class="skeleton-line w-45"></div>
        <div class="skeleton-line w-30"></div>
      </div>
    } @else {
      <article class="product-card" [class.is-adding]="adding()">
        <div class="product-media">
          <a
            [routerLink]="['/products', product().id]"
            class="media-link"
            [attr.aria-label]="'Ver ' + product().name"
          >
            <app-product-image
              [productId]="product().id"
              [name]="product().name"
              [imageUrl]="product().imageUrl ?? null"
              class="product-image"
            />
          </a>
          @if (isOutOfStock()) {
            <span class="stock-badge">Esgotado</span>
          }
          <button
            class="wishlist-btn"
            type="button"
            (click)="toggleWishlist($event)"
            [disabled]="!isAuthenticated()"
            [class.active]="isAuthenticated() && inWishlist()"
            [attr.aria-pressed]="inWishlist()"
            [attr.aria-label]="
              inWishlist() ? 'Remover da lista de desejos' : 'Adicionar à lista de desejos'
            "
          >
            <span
              [class]="inWishlist() ? 'icon icon-sm icon-heart-filled' : 'icon icon-sm icon-heart'"
              aria-hidden="true"
            ></span>
          </button>
        </div>

        <div class="product-info">
          <!-- <span class="product-eyebrow">Escolha MiniMart</span> -->
          <a [routerLink]="['/products', product().id]" class="product-name">{{
            product().name
          }}</a>
          @if (hasRating()) {
            <div class="rating" [attr.aria-label]="'Avaliado ' + ratingLabel() + ' de 5'">
              <span class="stars" aria-hidden="true">
                @for (filled of stars(); track $index) {
                  <span class="icon icon-sm icon-star" [class.dim]="!filled"></span>
                }
              </span>
              <span class="rating-value">{{ ratingLabel() }} ({{ product().totalReviews }})</span>
            </div>
          } @else {
            <div class="rating rating-empty">
              <span class="rating-value">Sem avaliações</span>
            </div>
          }
          <div class="product-footer">
            <span class="product-price">{{ product().priceCents | money }}</span>
            <button
              class="add-button"
              type="button"
              (click)="addToCart.emit(product().id)"
              [disabled]="adding() || isOutOfStock()"
              [attr.aria-label]="'Adicionar ' + product().name + ' ao carrinho'"
            >
              <span class="icon icon-sm icon-cart" aria-hidden="true"></span>
              <span>{{
                isOutOfStock() ? 'Esgotado' : adding() ? 'Adicionando…' : 'Adicionar'
              }}</span>
            </button>
          </div>
        </div>
      </article>
    }
  `,
  styles: [
    `
      :host {
        display: block;
        height: 100%;
      }
      .product-card {
        position: relative;
        display: flex;
        flex-direction: column;
        height: 100%;
        overflow: hidden;
        border: 1px solid var(--border);
        border-radius: var(--radius-xl);
        background: var(--card);
        box-shadow: var(--shadow-xs);
        transition:
          transform var(--transition),
          box-shadow var(--transition),
          border-color var(--transition);
      }
      .product-card:hover {
        transform: translateY(-5px);
        border-color: color-mix(in oklab, var(--accent) 30%, var(--border));
        box-shadow: var(--shadow-lg);
      }
      .product-media {
        position: relative;
        margin: 0.45rem;
        overflow: hidden;
        border-radius: calc(var(--radius-xl) - 0.35rem);
        background: var(--muted);
      }
      .media-link {
        display: block;
        aspect-ratio: 1 / 1;
      }
      .product-image {
        width: 100%;
        height: 100%;
        transition: transform var(--transition-slow);
      }
      .product-card:hover .product-image {
        transform: scale(1.045);
      }
      .stock-badge {
        position: absolute;
        top: 0.6rem;
        left: 0.6rem;
        padding: 0.28rem 0.55rem;
        border-radius: var(--radius-full);
        background: color-mix(in oklab, var(--background) 86%, transparent);
        color: var(--destructive);
        border: 1px solid color-mix(in oklab, var(--destructive) 26%, transparent);
        -webkit-backdrop-filter: blur(8px);
        backdrop-filter: blur(8px);
        font-size: 0.66rem;
        font-weight: 750;
      }
      .wishlist-btn {
        position: absolute;
        top: 0.55rem;
        right: 0.55rem;
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
      .wishlist-btn:hover {
        color: var(--destructive);
        transform: scale(1.06);
      }
      .wishlist-btn.active {
        color: var(--destructive);
        background: var(--color-danger-bg);
        border-color: color-mix(in oklab, var(--destructive) 30%, var(--border));
      }
      .product-info {
        display: flex;
        flex: 1;
        flex-direction: column;
        gap: 0.32rem;
        padding: 0.55rem 0.9rem 0.9rem;
      }
      .product-eyebrow {
        color: var(--muted-foreground);
        font-size: 0.63rem;
        font-weight: 700;
        letter-spacing: 0.09em;
        text-transform: uppercase;
      }
      .product-name {
        display: -webkit-box;
        overflow: hidden;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        color: var(--card-foreground);
        font-size: 0.9rem;
        font-weight: 650;
        line-height: 1.35;
        letter-spacing: -0.015em;
      }
      .product-name:hover {
        color: var(--accent-strong);
      }
      .rating {
        display: flex;
        align-items: center;
        gap: 0.35rem;
      }
      .stars {
        display: inline-flex;
        gap: 0.06rem;
        color: #f59e0b;
      }
      .stars .icon {
        width: 0.78rem;
        height: 0.78rem;
      }
      .stars .icon.dim {
        opacity: 0.25;
      }
      .rating-value {
        color: var(--muted-foreground);
        font-size: 0.7rem;
      }
      .product-footer {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.6rem;
        margin-top: auto;
        padding-top: 0.65rem;
      }
      .product-price {
        color: var(--foreground);
        font-size: 1.02rem;
        font-weight: 800;
        letter-spacing: -0.03em;
      }
      .add-button {
        display: inline-flex;
        align-items: center;
        gap: 0.35rem;
        min-height: 2.1rem;
        padding: 0.35rem 0.75rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-full);
        background: var(--muted);
        color: var(--foreground);
        font-size: 0.74rem;
        font-weight: 650;
        cursor: pointer;
        transition:
          background-color var(--transition-fast),
          color var(--transition-fast),
          border-color var(--transition-fast);
      }
      .add-button:hover:not(:disabled) {
        background: var(--accent);
        border-color: var(--accent);
        color: var(--accent-foreground);
      }
      .add-button:disabled {
        opacity: 0.6;
        cursor: wait;
      }
      .product-card.is-adding .add-button {
        background: var(--accent);
        color: var(--accent-foreground);
      }
      .card-skeleton {
        height: 100%;
        padding: 0.45rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-xl);
        background: var(--card);
      }
      .skeleton-media,
      .skeleton-line {
        background: linear-gradient(
          90deg,
          var(--color-skeleton-base) 25%,
          var(--color-skeleton-highlight) 50%,
          var(--color-skeleton-base) 75%
        );
        background-size: 200% 100%;
        animation: shimmer 1.5s infinite;
      }
      .skeleton-media {
        aspect-ratio: 1 / 1;
        border-radius: calc(var(--radius-xl) - 0.35rem);
      }
      .skeleton-line {
        height: 0.7rem;
        margin: 0.85rem 0.5rem 0;
        border-radius: var(--radius-sm);
      }
      .skeleton-line.w-70 {
        width: 70%;
      }
      .skeleton-line.w-45 {
        width: 45%;
      }
      .skeleton-line.w-30 {
        width: 30%;
        margin-bottom: 0.6rem;
      }
    `,
  ],
})
export class ProductCardComponent {
  product = input.required<Product>();
  loading = input(false);
  adding = input(false);
  addToCart = output<string>();

  private readonly wishlistStore = inject(WishlistStore);
  private readonly authStore = inject(AuthStore);
  readonly isAuthenticated = this.authStore.isAuthenticated;

  readonly inWishlist = computed(() => this.wishlistStore.isInWishlist(this.product().id));
  readonly isOutOfStock = computed(() => {
    const stock = this.product().stock;
    return stock !== undefined && stock <= 0;
  });

  private readonly ratingAverage = computed(() => {
    const total = this.product().totalReviews ?? 0;
    if (total <= 0) {
      return 0;
    }
    return (this.product().totalRating ?? 0) / total;
  });

  hasRating = computed(() => (this.product().totalReviews ?? 0) > 0);

  ratingLabel = computed(() => this.ratingAverage().toFixed(1).replace('.', ','));

  stars = computed(() => {
    const filled = Math.round(this.ratingAverage());
    return Array.from({ length: 5 }, (_, index) => index < filled);
  });

  toggleWishlist(event: Event): void {
    event.preventDefault();
    event.stopPropagation();
    if (!this.isAuthenticated()) {
      return;
    }
    this.wishlistStore.toggle(this.product().id);
  }

  quickView(event: Event): void {
    event.preventDefault();
    event.stopPropagation();
  }
}
