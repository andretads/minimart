import { Component, inject, signal, computed, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { catchError, forkJoin, map, of, Subject, switchMap, takeUntil } from 'rxjs';
import { ShowcaseApiService } from '../showcase-api.service';
import { Product, ProductReview } from '@shared/models/catalog.models';
import { ToastService } from '@core/toast/toast.service';
import { ProductImageComponent } from '@shared/components/product-image/product-image.component';
import { MoneyPipe } from '@shared/pipes/money.pipe';
import { QuantityStepperComponent } from '@shared/components/quantity-stepper/quantity-stepper.component';
import { ButtonComponent } from '@shared/components/button/button.component';
import { EmptyStateComponent } from '@shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '@shared/components/error-state/error-state.component';
import { PageShellComponent } from '@shared/components/page-shell/page-shell.component';
import { PaginationComponent } from '@shared/components/pagination/pagination.component';
import { SkeletonComponent } from '@shared/components/skeleton/skeleton.component';
import { ReviewFormComponent } from '@shared/components/review-form/review-form.component';
import { ReviewListComponent } from '@shared/components/review-list/review-list.component';
import { AuthStore } from '@core/auth/auth.store';
import { CartService } from '@features/cart/cart.service';
import { CartStore } from '@features/cart/cart.store';
import { WishlistStore } from '@features/wishlist/wishlist.store';

@Component({
  selector: 'app-product-detail',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    ProductImageComponent,
    MoneyPipe,
    QuantityStepperComponent,
    ButtonComponent,
    EmptyStateComponent,
    ErrorStateComponent,
    PageShellComponent,
    SkeletonComponent,
    PaginationComponent,
    ReviewFormComponent,
    ReviewListComponent,
  ],
  template: `
    <app-page-shell [title]="product()?.name || 'Produto'">
      <div class="detail-page">
        @if (loading()) {
          <div class="detail-skeleton" aria-busy="true">
            <app-skeleton variant="rectangular" width="100%" height="420px" radius="1.25rem" />
            <div class="skeleton-info">
              <app-skeleton variant="text" width="40%" height="2rem" />
              <app-skeleton variant="text" width="25%" height="1.5rem" />
              <app-skeleton variant="text" width="80%" />
              <app-skeleton variant="rectangular" width="100%" height="3rem" />
            </div>
          </div>
        } @else if (error()) {
          <app-error-state
            title="Falha ao carregar produto"
            [message]="error()!"
            retryLabel="Tentar novamente"
            (retry)="loadProduct()"
          />
        } @else if (!product()) {
          <app-empty-state
            title="Produto não encontrado"
            message="O produto que você procura não existe."
            icon="search"
          />
        } @else {
          <nav class="breadcrumb" aria-label="Navegação">
            <a routerLink="/">Início</a>
            <span class="icon icon-sm icon-chevron-right" aria-hidden="true"></span>
            <a routerLink="/products">Produtos</a>
            <span class="icon icon-sm icon-chevron-right" aria-hidden="true"></span>
            <span aria-current="page">{{ product()!.name }}</span>
          </nav>

          <div class="detail-grid">
            <div class="gallery">
              <div class="main-image-wrapper">
                <app-product-image
                  [productId]="product()!.id"
                  [name]="product()!.name"
                  [imageUrl]="activeImageUrl()"
                  class="main-image"
                />
              </div>
              @if (gallery().length > 1) {
                <div class="thumbnail-nav" aria-label="Galeria do produto">
                  @for (img of gallery(); track img.id; let i = $index) {
                    <button
                      class="thumb-btn"
                      [class.active]="activeThumb() === i"
                      (click)="setMainImage(i)"
                      [attr.aria-label]="'Ver imagem ' + (i + 1)"
                    >
                      <app-product-image
                        [productId]="product()!.id"
                        [name]="product()!.name"
                        [imageUrl]="img.url"
                        class="thumb-img"
                      />
                    </button>
                  }
                </div>
              }
            </div>

            <div class="detail-info">
              <div class="detail-topline">
                <span class="stock-chip" [class.out-of-stock]="outOfStock()"
                  ><span class="stock-dot" aria-hidden="true"></span> {{ stockLabel() }}</span
                >
                @if (product()!.sku) {
                  <span class="product-sku">SKU {{ product()!.sku }}</span>
                }
              </div>
              <h1 class="detail-name">{{ product()!.name }}</h1>
              @if (hasRating()) {
                <div class="detail-rating">
                  <span class="stars" aria-hidden="true">
                    @for (filled of stars(); track $index) {
                      <span class="icon icon-sm icon-star" [class.muted-star]="!filled"></span>
                    }
                  </span>
                  <span>{{ ratingLabel() }}</span>
                  <a href="#reviews">{{ product()!.totalReviews }} avaliações</a>
                </div>
              } @else {
                <div class="detail-rating"><span>Sem avaliações</span></div>
              }
              <div class="detail-price">{{ product()!.priceCents | money }}</div>
              <p class="detail-description">
                {{ product()!.description || 'Sem descrição disponível.' }}
              </p>

              <div class="detail-actions">
                <div class="quantity-row">
                  <span class="quantity-label">Quantidade</span>
                  <app-quantity-stepper
                    [value]="quantity()"
                    [min]="1"
                    [max]="maxQuantity()"
                    [disabled]="addingToCart()"
                    (valueChange)="quantity.set($event)"
                  />
                </div>
                <div class="action-buttons">
                  <app-button
                    [label]="outOfStock() ? 'Esgotado' : 'Adicionar ao carrinho'"
                    variant="primary"
                    size="lg"
                    [fullWidth]="true"
                    [disabled]="!isAuthenticated() || addingToCart() || outOfStock()"
                    [loading]="addingToCart()"
                    (clicked)="addToCart()"
                  />
                  <button
                    class="wishlist-action"
                    (click)="toggleWishlist()"
                    [disabled]="!isAuthenticated()"
                    type="button"
                    [attr.aria-pressed]="inWishlist()"
                  >
                    <span
                      [class]="inWishlist() ? 'icon icon-heart-filled' : 'icon icon-heart'"
                      aria-hidden="true"
                    ></span>
                    <span>{{ inWishlist() ? 'Salvo' : 'Salvar' }}</span>
                  </button>
                </div>
                @if (!isAuthenticated()) {
                  <p class="auth-prompt">
                    Por favor
                    <a [routerLink]="'/login'" [queryParams]="{ returnUrl: router.url }">entre</a>
                    para adicionar itens ao carrinho.
                  </p>
                }
              </div>

              <div class="product-meta">
                <div class="meta-item">
                  <span class="icon icon-truck" aria-hidden="true"></span
                  ><span><strong>Frete grátis</strong>Em pedidos acima de R$ 50</span>
                </div>
                <div class="meta-item">
                  <span class="icon icon-rotate-ccw" aria-hidden="true"></span
                  ><span><strong>Devolução em 30 dias</strong>Sem perguntas</span>
                </div>
                <div class="meta-item">
                  <span class="icon icon-shield" aria-hidden="true"></span
                  ><span><strong>Pagamento seguro</strong>Finalização criptografada</span>
                </div>
                <div class="meta-item">
                  <span class="icon icon-headset" aria-hidden="true"></span
                  ><span><strong>Aqui para ajudar</strong>7 dias por semana</span>
                </div>
              </div>
            </div>
          </div>

          <div class="tabs" id="reviews">
            <input type="radio" name="product-tab" id="tab-description" class="tab-radio" checked />
            <input
              type="radio"
              name="product-tab"
              id="tab-specs"
              class="tab-radio"
              (change)="onTabChange()"
            />
            <input type="radio" name="product-tab" id="tab-reviews" class="tab-radio" />
            <div class="tab-list" role="tablist" aria-label="Detalhes do produto">
              <label for="tab-description" class="tab-label">Descrição</label>
              <label for="tab-specs" class="tab-label">Especificações</label>
              <label for="tab-reviews" class="tab-label">Avaliações ({{ reviews().length }})</label>
            </div>
            <div class="tab-panels">
              <section class="tab-panel panel-description">
                <h3>Descrição do produto</h3>
                <p>{{ product()!.description || 'Sem descrição disponível.' }}</p>
              </section>
              <section class="tab-panel panel-specs">
                <h3>Especificações</h3>
                @if (details().length) {
                  <table class="spec-table">
                    <tbody>
                      @for (detail of paginatedDetails(); track detail.name) {
                        <tr>
                          <th>{{ detail.name }}</th>
                          <td>{{ detail.value }}</td>
                        </tr>
                      }
                    </tbody>
                  </table>
                  @if (specsTotalPages() > 1) {
                    <nav class="specs-pagination" aria-label="Paginação das especificações">
                      <app-pagination
                        [currentPage]="specsPage()"
                        [totalPages]="specsTotalPages()"
                        (pageChange)="specsPage.set($event)"
                      />
                    </nav>
                  }
                } @else {
                  <p>Nenhuma especificação disponível.</p>
                }
              </section>
              <section class="tab-panel panel-reviews">
                <h3>Avaliações dos clientes</h3>

                @if (isAuthenticated() && !showReviewForm()) {
                  <div class="review-cta">
                    <app-button
                      label="Escrever avaliação"
                      variant="outline"
                      (clicked)="showReviewForm.set(true)"
                    />
                  </div>
                }

                @if (isAuthenticated() && showReviewForm()) {
                  <app-review-form
                    [productId]="product()!.id"
                    [submitting]="submittingReview()"
                    (submitted)="onReviewSubmitted($event)"
                    (cancelled)="showReviewForm.set(false)"
                  />
                }

                <app-review-list
                  class="review-list"
                  [reviews]="reviews()"
                  [totalReviews]="product()?.totalReviews ?? 0"
                />
              </section>
            </div>
          </div>
        }
      </div>
    </app-page-shell>
  `,
  styles: [
    `
      .detail-page {
        max-width: 76rem;
        margin: 0 auto;
      }
      .detail-skeleton {
        display: grid;
        gap: 2.5rem;
        grid-template-columns: 1fr;
      }
      .skeleton-info {
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }
      @media (min-width: 992px) {
        .detail-skeleton {
          grid-template-columns: 1.05fr 0.95fr;
        }
      }

      .breadcrumb {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 0.375rem;
        margin-bottom: 1.5rem;
        font-size: 0.8125rem;
      }
      .breadcrumb a {
        color: var(--color-text-muted);
        text-decoration: none;
        transition: color 0.2s;
      }
      .breadcrumb a:hover {
        color: var(--color-accent);
      }
      .breadcrumb .icon {
        color: var(--color-text-subtle);
      }
      .breadcrumb span:last-child {
        color: var(--color-text);
        font-weight: 500;
      }

      .detail-grid {
        display: grid;
        grid-template-columns: 1fr;
        gap: 2.5rem;
        margin-bottom: 3rem;
        align-items: start;
      }
      @media (min-width: 992px) {
        .detail-grid {
          grid-template-columns: 1.05fr 0.95fr;
        }
      }

      .gallery {
        position: sticky;
        top: 6rem;
      }
      .main-image-wrapper {
        position: relative;
        border: 1px solid var(--color-border);
        border-radius: 1.5rem;
        overflow: hidden;
        background: var(--color-surface-muted);
        box-shadow: var(--shadow-md);
      }
      .main-image {
        display: block;
        width: 100%;
        aspect-ratio: 1;
      }
      .thumbnail-nav {
        display: flex;
        gap: 0.625rem;
        margin-top: 0.875rem;
        overflow-x: auto;
        padding-bottom: 0.25rem;
      }
      .thumb-btn {
        flex: 0 0 auto;
        width: 4.5rem;
        height: 4.5rem;
        padding: 0;
        border: 2px solid transparent;
        border-radius: 0.875rem;
        overflow: hidden;
        background: var(--color-surface-muted);
        cursor: pointer;
        opacity: 0.7;
        transition:
          opacity 0.2s,
          border-color 0.2s;
      }
      .thumb-btn:hover,
      .thumb-btn.active {
        opacity: 1;
        border-color: var(--color-accent);
      }
      .thumb-img {
        display: block;
        width: 100%;
        height: 100%;
      }

      .detail-info {
        display: flex;
        flex-direction: column;
      }
      .detail-topline {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        margin-bottom: 0.875rem;
      }
      .stock-chip {
        display: inline-flex;
        align-items: center;
        gap: 0.375rem;
        padding: 0.3rem 0.65rem;
        border-radius: 999px;
        font-size: 0.75rem;
        font-weight: 600;
        color: var(--color-accent);
        background: color-mix(in oklab, var(--color-accent) 12%, transparent);
      }
      .stock-chip.out-of-stock {
        color: var(--destructive);
        background: color-mix(in oklab, var(--destructive) 12%, transparent);
      }
      .stock-chip.out-of-stock .stock-dot {
        background: var(--destructive);
        box-shadow: 0 0 0 3px color-mix(in oklab, var(--destructive) 20%, transparent);
      }
      .stock-dot {
        width: 0.375rem;
        height: 0.375rem;
        border-radius: 999px;
        background: var(--color-accent);
        box-shadow: 0 0 0 3px color-mix(in oklab, var(--color-accent) 20%, transparent);
      }
      .product-sku {
        font-family: var(--font-mono);
        font-size: 0.6875rem;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--color-text-subtle);
      }
      .detail-name {
        margin: 0.25rem 0 0;
        font-size: clamp(1.9rem, 4vw, 2.75rem);
        font-weight: 750;
        line-height: 1.05;
        letter-spacing: -0.025em;
        color: var(--color-text);
      }
      .detail-rating {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        margin-top: 0.875rem;
        font-size: 0.875rem;
        color: var(--color-text-muted);
      }
      .detail-rating .stars {
        display: inline-flex;
        gap: 0.1rem;
        color: #f59e0b;
      }
      .detail-rating a {
        margin-left: 0.25rem;
        color: var(--color-text-muted);
        text-decoration: none;
        border-bottom: 1px solid var(--color-border);
      }
      .detail-rating a:hover {
        color: var(--color-accent);
      }
      .muted-star {
        opacity: 0.2;
      }
      .detail-price {
        margin: 1.25rem 0 0.75rem;
        font-size: clamp(2rem, 3vw, 2.5rem);
        font-weight: 750;
        letter-spacing: -0.02em;
        color: var(--color-text);
      }
      .detail-description {
        max-width: 34rem;
        margin: 0 0 1.75rem;
        font-size: 0.9375rem;
        line-height: 1.7;
        color: var(--color-text-muted);
      }

      .detail-actions {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        padding: 1.25rem;
        border: 1px solid var(--color-border);
        border-radius: 1.25rem;
        background: var(--color-surface);
        box-shadow: var(--shadow-sm);
      }
      .quantity-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
      }
      .quantity-label {
        font-size: 0.875rem;
        font-weight: 600;
        color: var(--color-text);
      }
      .action-buttons {
        display: grid;
        grid-template-columns: 1fr auto;
        gap: 0.625rem;
        align-items: stretch;
      }
      .wishlist-action {
        display: inline-flex;
        align-items: center;
        gap: 0.4rem;
        padding: 0.75rem 1rem;
        border: 1px solid var(--color-border);
        border-radius: 999px;
        background: transparent;
        color: var(--color-text-muted);
        font-size: 0.8125rem;
        font-weight: 600;
        cursor: pointer;
        transition:
          color 0.2s,
          border-color 0.2s,
          background 0.2s;
      }
      .wishlist-action:hover:not(:disabled) {
        color: var(--color-accent);
        border-color: color-mix(in oklab, var(--color-accent) 45%, transparent);
        background: color-mix(in oklab, var(--color-accent) 7%, transparent);
      }
      .wishlist-action:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
      .auth-prompt {
        margin: 0;
        font-size: 0.8125rem;
        text-align: center;
        color: var(--color-text-muted);
      }
      .auth-prompt a {
        color: var(--color-accent);
        font-weight: 600;
      }

      .product-meta {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 0.875rem;
        margin-top: 1.5rem;
        padding-top: 1.5rem;
        border-top: 1px solid var(--color-border);
      }
      .meta-item {
        display: flex;
        gap: 0.625rem;
        align-items: flex-start;
        font-size: 0.75rem;
        color: var(--color-text-muted);
        line-height: 1.45;
      }
      .meta-item .icon {
        flex: 0 0 auto;
        margin-top: 0.1rem;
        color: var(--color-accent);
      }
      .meta-item strong {
        display: block;
        margin-bottom: 0.1rem;
        font-size: 0.8125rem;
        font-weight: 600;
        color: var(--color-text);
      }

      .tabs {
        position: relative;
        overflow: hidden;
        border: 1px solid var(--color-border);
        border-radius: 1.25rem;
        background: var(--color-surface);
      }
      .tab-radio {
        position: absolute;
        width: 1px;
        height: 1px;
        opacity: 0;
        pointer-events: none;
      }
      .tab-list {
        display: flex;
        gap: 0.25rem;
        padding: 0 0.75rem;
        overflow-x: auto;
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface-muted);
      }
      .tab-label {
        padding: 1rem 1.1rem;
        border-bottom: 2px solid transparent;
        margin-bottom: -1px;
        font-size: 0.875rem;
        font-weight: 600;
        color: var(--color-text-muted);
        white-space: nowrap;
        cursor: pointer;
        transition:
          color 0.2s,
          border-color 0.2s;
      }
      .tab-label:hover {
        color: var(--color-text);
      }
      .tab-panel {
        display: none;
        max-width: 60rem;
        padding: 1.75rem;
        animation: fade-up 0.3s ease both;
      }
      .tab-panel h3 {
        margin: 0 0 0.75rem;
        font-size: 1.125rem;
        font-weight: 700;
        letter-spacing: -0.015em;
      }
      .tab-panel h4 {
        margin: 1.5rem 0 0.5rem;
        font-size: 0.9375rem;
        font-weight: 700;
      }
      .tab-panel p {
        margin: 0;
        color: var(--color-text-muted);
        line-height: 1.7;
      }
      #tab-description:checked ~ .tab-list label[for='tab-description'],
      #tab-specs:checked ~ .tab-list label[for='tab-specs'],
      #tab-reviews:checked ~ .tab-list label[for='tab-reviews'] {
        color: var(--color-accent);
        border-color: var(--color-accent);
      }
      #tab-description:checked ~ .tab-panels .panel-description,
      #tab-specs:checked ~ .tab-panels .panel-specs,
      #tab-reviews:checked ~ .tab-panels .panel-reviews {
        display: block;
      }
      .feature-list {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 0.5rem 1rem;
        margin: 1rem 0 0;
        padding: 0;
        list-style: none;
      }
      .feature-list li {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        font-size: 0.875rem;
        color: var(--color-text);
      }
      .feature-list .icon {
        color: var(--color-accent);
      }

      .spec-table {
        width: 100%;
        border-collapse: collapse;
      }
      .spec-table th,
      .spec-table td {
        padding: 0.8rem 0.25rem;
        border-bottom: 1px solid var(--color-border);
        text-align: left;
        font-size: 0.875rem;
      }
      .spec-table th {
        width: 35%;
        font-weight: 500;
        color: var(--color-text-muted);
      }
      .spec-table tr:last-child th,
      .spec-table tr:last-child td {
        border-bottom: 0;
      }

      .specs-pagination {
        margin-top: 1rem;
        padding-top: 1rem;
        border-top: 1px solid var(--color-border);
      }

      .review-summary {
        display: flex;
        flex-wrap: wrap;
        gap: 2rem;
        padding: 1.25rem;
        margin-bottom: 1.5rem;
        border-radius: 1rem;
        background: var(--color-surface-muted);
      }
      .overall-rating {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.35rem;
        min-width: 9rem;
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }
      .big-rating {
        font-size: 2.75rem;
        font-weight: 750;
        line-height: 1;
        color: var(--color-text);
      }
      .overall-rating .stars {
        display: inline-flex;
        gap: 0.1rem;
        color: #f59e0b;
      }
      .review-item {
        padding: 1.1rem 0;
        border-bottom: 1px solid var(--color-border);
      }
      .review-item:last-child {
        border-bottom: 0;
      }
      .review-head {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 1rem;
        margin-bottom: 0.35rem;
      }
      .review-head > div:first-child {
        display: flex;
        flex-direction: column;
      }
      .review-head strong {
        font-size: 0.9rem;
      }
      .review-head span {
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }
      .review-head .stars {
        display: inline-flex;
        gap: 0.1rem;
        color: #f59e0b;
      }
      .review-item p {
        margin: 0;
        font-size: 0.875rem;
        line-height: 1.6;
        color: var(--color-text-muted);
      }
      .reviews-empty {
        padding: 1rem 0;
      }
      .review-list {
        display: block;
        margin-top: 1rem;
      }

      @media (max-width: 991px) {
        .gallery {
          position: static;
        }
      }
      @media (max-width: 640px) {
        .detail-actions {
          padding: 1rem;
        }
        .action-buttons {
          grid-template-columns: 1fr;
        }
        .product-meta {
          grid-template-columns: 1fr;
        }
        .feature-list {
          grid-template-columns: 1fr;
        }
        .tab-panel {
          padding: 1.25rem;
        }
        .tab-label {
          padding: 0.9rem 0.8rem;
          font-size: 0.8125rem;
        }
        .review-summary {
          padding: 1rem;
          gap: 1rem;
        }
        .overall-rating {
          flex-direction: row;
          min-width: auto;
        }
      }
    `,
  ],
})
export class ProductDetailComponent implements OnInit, OnDestroy {
  private readonly api = inject(ShowcaseApiService);
  private readonly route = inject(ActivatedRoute);
  protected readonly router = inject(Router);
  private readonly authStore = inject(AuthStore);
  private readonly cartService = inject(CartService);
  private readonly cartStore = inject(CartStore);
  private readonly toast = inject(ToastService);
  private readonly destroy$ = new Subject<void>();
  private readonly wishlistStore = inject(WishlistStore);

  product = signal<Product | null>(null);
  reviews = signal<ProductReview[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  quantity = signal(1);
  addingToCart = signal(false);
  readonly inWishlist = computed(() => this.wishlistStore.isInWishlist(this.product()?.id ?? ''));
  activeThumb = signal(0);
  showReviewForm = signal(false);
  submittingReview = signal(false);

  // Specs pagination
  readonly SPECS_PAGE_SIZE = 5;
  specsPage = signal(1);

  specsTotalPages = computed(() => Math.ceil(this.details().length / this.SPECS_PAGE_SIZE));

  paginatedDetails = computed(() => {
    const all = this.details();
    const page = this.specsPage();
    const start = (page - 1) * this.SPECS_PAGE_SIZE;
    return all.slice(start, start + this.SPECS_PAGE_SIZE);
  });

  isAuthenticated = this.authStore.isAuthenticated;

  gallery = computed(() => this.product()?.images ?? []);

  details = computed(() => this.product()?.details ?? []);

  activeImageUrl = computed(() => {
    const images = this.gallery();
    return images[this.activeThumb()]?.url ?? this.product()?.imageUrl ?? null;
  });

  outOfStock = computed(() => {
    const stock = this.product()?.stock;
    return stock !== undefined && stock <= 0;
  });

  maxQuantity = computed(() => {
    const stock = this.product()?.stock ?? 99;
    return Math.max(1, Math.min(99, stock));
  });

  stockLabel = computed(() => {
    const stock = this.product()?.stock;
    if (stock === undefined) {
      return 'Em estoque';
    }
    return stock > 0 ? `${stock} em estoque` : 'Esgotado';
  });

  hasRating = computed(() => (this.product()?.totalReviews ?? 0) > 0);

  private readonly ratingAverage = computed(() => {
    const total = this.product()?.totalReviews ?? 0;
    if (total <= 0) {
      return 0;
    }
    return (this.product()?.totalRating ?? 0) / total;
  });

  ratingLabel = computed(() => this.ratingAverage().toFixed(1).replace('.', ','));

  stars = computed(() => {
    const filled = Math.round(this.ratingAverage());
    return Array.from({ length: 5 }, (_, index) => index < filled);
  });

  ngOnInit(): void {
    this.route.paramMap
      .pipe(
        map((params) => params.get('id')!),
        switchMap((id) => this.load(id)),
        takeUntil(this.destroy$),
      )
      .subscribe((result) => {
        this.applyResult(result);
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  setMainImage(index: number): void {
    this.activeThumb.set(index);
  }

  toggleWishlist(): void {
    const product = this.product();
    if (product) {
      this.wishlistStore.toggle(product.id);
    }
  }

  onTabChange(): void {
    this.specsPage.set(1);
  }

  addToCart(): void {
    if (!this.isAuthenticated() || this.addingToCart() || this.outOfStock()) return;
    this.addingToCart.set(true);
    this.cartService.addItem(this.product()!.id, this.quantity()).subscribe({
      next: (cart) => {
        this.addingToCart.set(false);
        this.cartStore.setCart(cart);
        this.toast.success('Produto adicionado ao carrinho');
      },
      error: (err) => {
        this.addingToCart.set(false);
        this.toast.error(err.message || 'Não foi possível adicionar ao carrinho');
      },
    });
  }

  onReviewSubmitted(data: { description: string; rating: number }): void {
    const product = this.product();
    if (!product) return;
    this.submittingReview.set(true);
    this.api.createReview(product.id, data).subscribe({
      next: () => {
        this.toast.success('Avaliação enviada para moderação');
        this.showReviewForm.set(false);
        this.loadProduct();
      },
      error: (err) => {
        this.toast.error(err.message || 'Erro ao enviar avaliação');
        this.submittingReview.set(false);
      },
      complete: () => this.submittingReview.set(false),
    });
  }

  loadProduct(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) return;
    this.loading.set(true);
    this.error.set(null);
    this.load(id).subscribe((result) => {
      this.applyResult(result);
    });
  }

  private load(id: string) {
    return forkJoin({
      product: this.api.get(id),
      reviews: this.api.getReviews(id).pipe(
        map((r) => r.data),
        catchError(() => of([] as ProductReview[])),
      ),
    }).pipe(
      catchError((err) => {
        this.error.set(err.message);
        return of(null);
      }),
    );
  }

  private applyResult(result: { product: Product; reviews: ProductReview[] } | null): void {
    if (result) {
      this.product.set(result.product);
      this.reviews.set(result.reviews);
      this.activeThumb.set(0);
      this.specsPage.set(1);
    }
    this.loading.set(false);
  }
}
