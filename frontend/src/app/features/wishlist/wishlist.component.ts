import { Component, inject, OnInit, signal, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { WishlistStore } from './wishlist.store';
import { WishlistItem } from '@shared/models/wishlist.models';
import { ProductCardComponent } from '../catalog/product-card/product-card.component';
import { PageShellComponent } from '@shared/components/page-shell/page-shell.component';
import { EmptyStateComponent } from '@shared/components/empty-state/empty-state.component';
import { ShowcaseApiService } from '../catalog/showcase-api.service';
import { Product } from '@shared/models/catalog.models';
import { AuthStore } from '@core/auth/auth.store';
import { CartService } from '@features/cart/cart.service';
import { CartStore } from '@features/cart/cart.store';
import { ToastService } from '@core/toast/toast.service';
import { forkJoin, of, catchError } from 'rxjs';

@Component({
  selector: 'app-wishlist',
  standalone: true,
  imports: [CommonModule, ProductCardComponent, PageShellComponent, EmptyStateComponent],
  template: `
    <app-page-shell [title]="'Minha Lista de Desejos'">
      <div class="wishlist-page">
        @if (loading()) {
          <div class="wishlist-skeleton" aria-busy="true">
            @for (i of [1, 2, 3, 4]; track i) {
              <div class="product-card-skeleton">
                <div class="skeleton-media"></div>
                <div class="skeleton-line w-70"></div>
                <div class="skeleton-line w-45"></div>
                <div class="skeleton-line w-30"></div>
              </div>
            }
          </div>
        } @else if (products().length === 0) {
          <app-empty-state
            title="Lista de desejos vazia"
            message="Adicione produtos que você gostaria de comprar futuramente."
            icon="heart"
          />
        } @else {
          <div class="wishlist-header">
            <h2>{{ products().length }} item{{ products().length !== 1 ? 's' : '' }}</h2>
            <button class="clear-btn" (click)="clearWishlist()" [disabled]="loading()">
              Limpar lista
            </button>
          </div>
          <div class="wishlist-grid">
            @for (product of products(); track product.id) {
              <app-product-card
                [product]="product"
                [adding]="addingIds().has(product.id)"
                (addToCart)="onAddToCart($event)"
              />
            }
          </div>
        }
      </div>
    </app-page-shell>
  `,
  styles: [
    `
      .wishlist-page {
        max-width: 76rem;
        margin: 0 auto;
      }
      .wishlist-skeleton {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr));
        gap: 1.5rem;
      }
      .wishlist-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 1.5rem;
      }
      .wishlist-header h2 {
        margin: 0;
        font-size: 1.25rem;
        font-weight: 600;
        color: var(--foreground);
      }
      .clear-btn {
        padding: 0.5rem 1rem;
        border: 1px solid var(--border);
        border-radius: var(--radius-full);
        background: transparent;
        color: var(--muted-foreground);
        font-size: 0.875rem;
        font-weight: 500;
        cursor: pointer;
        transition:
          color var(--transition-fast),
          border-color var(--transition-fast);
      }
      .clear-btn:hover:not(:disabled) {
        color: var(--destructive);
        border-color: var(--destructive);
      }
      .clear-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
      .wishlist-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr));
        gap: 1.5rem;
      }
      @media (max-width: 640px) {
        .wishlist-grid {
          grid-template-columns: repeat(2, 1fr);
        }
      }
      .product-card-skeleton {
        border: 1px solid var(--border);
        border-radius: var(--radius-xl);
        background: var(--card);
        padding: 0.45rem;
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
export class WishlistComponent implements OnInit {
  private readonly store = inject(WishlistStore);
  private readonly showcaseApi = inject(ShowcaseApiService);
  private readonly router = inject(Router);
  private readonly authStore = inject(AuthStore);
  private readonly cartService = inject(CartService);
  private readonly cartStore = inject(CartStore);
  private readonly toast = inject(ToastService);

  readonly loading = this.store.loading;
  readonly products = signal<Product[]>([]);
  readonly addingIds = signal<Set<string>>(new Set());

  // effect runs in injection context (field initializer)
  private readonly _wishlistEffect = effect(() => {
    const wishlist = this.store.wishlist();
    if (wishlist && wishlist.items.length > 0) {
      this.fetchProducts(wishlist.items);
    } else {
      this.products.set([]);
    }
  });

  ngOnInit(): void {
    this.store.load();
  }

  private fetchProducts(items: WishlistItem[]): void {
    const productIds = items.map((i) => i.productId);
    const requests = productIds.map((id) =>
      this.showcaseApi.get(id).pipe(catchError(() => of(null))),
    );

    forkJoin(requests).subscribe((results) => {
      const validProducts = results.filter((p): p is Product => p !== null);
      this.products.set(validProducts);
    });
  }

  onAddToCart(productId: string): void {
    if (!this.authStore.isAuthenticated()) {
      this.router.navigate(['/login'], { queryParams: { returnUrl: this.router.url } });
      return;
    }
    this.addingIds.update((ids) => {
      const next = new Set(ids);
      next.add(productId);
      return next;
    });
    this.cartService.addItem(productId, 1).subscribe({
      next: (cart) => {
        this.cartStore.setCart(cart);
        this.toast.success('Produto adicionado ao carrinho');
        this.addingIds.update((ids) => {
          const next = new Set(ids);
          next.delete(productId);
          return next;
        });
      },
      error: (err) => {
        this.addingIds.update((ids) => {
          const next = new Set(ids);
          next.delete(productId);
          return next;
        });
        this.toast.error(err?.message || 'Não foi possível adicionar ao carrinho');
      },
    });
  }

  clearWishlist(): void {
    this.store.clear();
    this.products.set([]);
  }
}
