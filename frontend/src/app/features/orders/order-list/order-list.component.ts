import { Component, inject, signal, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Subject, takeUntil, interval, timer, Subscription } from 'rxjs';
import { OrdersApiService } from '../orders-api.service';
import { Order, ORDER_STATUS_LABELS } from '@shared/models/order.models';
import { autoCancelAt, formatRemaining } from '@shared/utils/order-deadline';
import { MoneyPipe } from '@shared/pipes/money.pipe';
import { PageShellComponent } from '@shared/components/page-shell/page-shell.component';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { EmptyStateComponent } from '@shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '@shared/components/error-state/error-state.component';
import { PaginationComponent } from '@shared/components/pagination/pagination.component';
import { SkeletonComponent } from '@shared/components/skeleton/skeleton.component';
import { ProductImageComponent } from '@shared/components/product-image/product-image.component';

@Component({
  selector: 'app-order-list',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    MoneyPipe,
    PageShellComponent,
    StatusBadgeComponent,
    EmptyStateComponent,
    ErrorStateComponent,
    PaginationComponent,
    SkeletonComponent,
    ProductImageComponent,
  ],
  template: `
    <app-page-shell title="Meus pedidos">
      <div class="orders-page">
        @if (loading()) {
          <div class="orders-skeleton" aria-busy="true">
            @for (i of [1, 2, 3]; track i) {
              <app-skeleton variant="rectangular" height="8.5rem" />
            }
          </div>
        } @else if (error()) {
          <app-error-state
            title="Falha ao carregar pedidos"
            [message]="error()!"
            retryLabel="Tentar novamente"
            (retry)="load(page())"
          />
        } @else if (orders().length === 0) {
          <app-empty-state
            title="Nenhum pedido ainda"
            message="Quando você finalizar uma compra, seus pedidos aparecerão aqui."
            icon="package"
            actionLabel="Explorar produtos"
            (action)="goShopping()"
          />
        } @else {
          <div class="orders-toolbar">
            <p class="orders-summary">
              <strong>{{ total() }}</strong> pedido{{ total() === 1 ? '' : 's' }}
            </p>
          </div>

          <div class="orders-list">
            @for (order of orders(); track order.id) {
              <article class="order-card">
                <div class="order-media">
                  @if (order.items.length > 0) {
                    <app-product-image
                      [productId]="order.items[0].productId"
                      [name]="order.items[0].name"
                      class="order-thumb"
                    />
                  } @else {
                    <span class="order-thumb placeholder"><i class="icon icon-package"></i></span>
                  }
                </div>

                <div class="order-main">
                  <div class="order-head">
                    <div>
                      <span class="order-eyebrow">Pedido</span>
                      <h2>#{{ order.id.slice(0, 8).toUpperCase() }}</h2>
                    </div>
                    <app-status-badge
                      [status]="order.status"
                      [label]="ORDER_STATUS_LABELS[order.status]"
                    />
                  </div>

                  <p class="order-meta">
                    {{ formatDate(order.createdAt) }}
                    <span class="dot" aria-hidden="true">·</span>
                    {{ order.items.length }} item{{ order.items.length === 1 ? '' : 's' }}
                    <span class="dot" aria-hidden="true">·</span>
                    <span class="order-total">{{ order.totalCents | money }}</span>
                  </p>

                  @if (order.items.length > 0) {
                    <p class="order-items">{{ itemSummary(order) }}</p>
                  }
                  @if (order.status === 'placed') {
                    <p class="order-deadline">
                      Cancelamento automático em {{ remainingLabel(order) }}
                    </p>
                  }
                </div>

                <a class="order-action" [routerLink]="['/orders', order.id]">
                  Ver detalhes
                  <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span>
                </a>
              </article>
            }
          </div>

          @if (totalPages() > 1) {
            <app-pagination
              [currentPage]="page()"
              [totalPages]="totalPages()"
              (pageChange)="onPageChange($event)"
            />
          }
        }
      </div>
    </app-page-shell>
  `,
  styles: [
    `
      .orders-page {
        max-width: 62rem;
        margin: 0 auto;
      }
      .orders-skeleton {
        display: grid;
        gap: 0.9rem;
      }
      .orders-toolbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        margin-bottom: 1rem;
      }
      .orders-summary {
        margin: 0;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .orders-summary strong {
        color: var(--color-text);
        font-weight: 700;
      }

      .orders-list {
        display: grid;
        gap: 0.9rem;
      }
      .order-card {
        display: flex;
        align-items: center;
        gap: 1rem;
        padding: 1.1rem 1.25rem;
        border: 1px solid var(--color-border);
        border-radius: 1rem;
        background: var(--color-surface);
        transition:
          border-color 0.2s,
          box-shadow 0.2s,
          transform 0.2s;
      }
      .order-card:hover {
        border-color: color-mix(in oklab, var(--accent) 32%, transparent);
        box-shadow: var(--shadow-sm);
        transform: translateY(-1px);
      }
      .order-media {
        flex: 0 0 auto;
      }
      .order-thumb {
        display: grid;
        place-items: center;
        width: 4.25rem;
        height: 4.25rem;
        border-radius: 0.85rem;
        overflow: hidden;
        background: var(--color-surface-muted);
        color: var(--color-text-muted);
      }
      .order-thumb.placeholder {
        display: grid;
        place-items: center;
      }
      .order-main {
        flex: 1;
        min-width: 0;
      }
      .order-head {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 0.75rem;
      }
      .order-eyebrow {
        display: block;
        margin-bottom: 0.15rem;
        font-size: 0.625rem;
        font-weight: 700;
        letter-spacing: 0.12em;
        text-transform: uppercase;
        color: var(--color-text-muted);
      }
      .order-head h2 {
        margin: 0;
        font-size: 1rem;
        font-weight: 700;
        letter-spacing: -0.02em;
      }
      .order-meta {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.35rem;
        margin: 0.45rem 0 0.2rem;
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }
      .order-total {
        color: var(--color-text);
        font-weight: 650;
      }
      .dot {
        opacity: 0.55;
      }
      .order-items {
        margin: 0;
        font-size: 0.75rem;
        color: var(--color-text-muted);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
      .order-action {
        display: inline-flex;
        align-items: center;
        gap: 0.35rem;
        flex: 0 0 auto;
        padding: 0.55rem 0.85rem;
        border: 1px solid var(--color-border);
        border-radius: 0.7rem;
        color: var(--color-text-muted);
        font-size: 0.75rem;
        font-weight: 600;
        text-decoration: none;
        transition:
          border-color 0.2s,
          color 0.2s,
          background 0.2s;
      }
      .order-action:hover {
        border-color: color-mix(in oklab, var(--accent) 40%, transparent);
        color: var(--color-accent);
        background: color-mix(in oklab, var(--accent) 7%, transparent);
      }

      @media (max-width: 640px) {
        .order-card {
          align-items: flex-start;
          flex-wrap: wrap;
          padding: 1rem;
        }
        .order-thumb {
          width: 3.25rem;
          height: 3.25rem;
        }
        .order-main {
          flex-basis: calc(100% - 4.25rem);
        }
        .order-head {
          flex-direction: column;
        }
        .order-action {
          width: 100%;
          justify-content: center;
        }
      }
    `,
  ],
})
export class OrderListComponent implements OnInit, OnDestroy {
  private readonly ordersApi = inject(OrdersApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly destroy$ = new Subject<void>();
  private countdownSubscription?: Subscription;
  private reloadTimers: Subscription[] = [];

  readonly ORDER_STATUS_LABELS = ORDER_STATUS_LABELS;
  readonly pageSize = 5;
  orders = signal<Order[]>([]);
  now = signal(Date.now());
  loading = signal(true);
  error = signal<string | null>(null);
  page = signal(1);
  total = signal(0);
  totalPages = signal(0);

  ngOnInit(): void {
    this.route.queryParams.pipe(takeUntil(this.destroy$)).subscribe((params) => {
      const parsed = Number.parseInt(params['page'] ?? '1', 10);
      this.load(Number.isFinite(parsed) && parsed > 0 ? parsed : 1);
    });
    this.countdownSubscription = interval(1000)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => this.now.set(Date.now()));
  }

  ngOnDestroy(): void {
    this.clearReloadTimers();
    this.countdownSubscription?.unsubscribe();
    this.countdownSubscription = undefined;
    this.destroy$.next();
    this.destroy$.complete();
  }

  load(page: number): void {
    this.loading.set(true);
    this.error.set(null);
    this.ordersApi.list(page, this.pageSize).subscribe({
      next: (response) => {
        this.orders.set(response.data);
        this.page.set(response.meta.page);
        this.total.set(response.meta.total);
        this.totalPages.set(response.meta.totalPages);
        this.loading.set(false);
        this.scheduleAutoCancelReloads();
      },
      error: (err) => {
        this.error.set(err.message);
        this.loading.set(false);
      },
    });
  }

  onPageChange(page: number): void {
    this.router.navigate([], { queryParams: { page }, queryParamsHandling: 'merge' });
  }

  goShopping(): void {
    this.router.navigate(['/products']);
  }

  itemSummary(order: Order): string {
    const names = order.items.slice(0, 3).map((item) => item.name);
    const remaining = order.items.length - names.length;
    return remaining > 0 ? `${names.join(', ')} e mais ${remaining}` : names.join(', ');
  }

  remainingLabel(order: Order): string {
    if (order.status !== 'placed') {
      return '';
    }
    return formatRemaining(autoCancelAt(order).getTime() - this.now());
  }

  private clearReloadTimers(): void {
    this.reloadTimers.forEach((timer) => timer.unsubscribe());
    this.reloadTimers = [];
  }

  private scheduleAutoCancelReloads(): void {
    this.clearReloadTimers();
    for (const order of this.orders()) {
      if (order.status !== 'placed') {
        continue;
      }
      const delay = autoCancelAt(order).getTime() + 15000 - Date.now();
      if (delay <= 0) {
        continue;
      }
      const subscription = timer(delay)
        .pipe(takeUntil(this.destroy$))
        .subscribe(() => this.reloadSilently());
      this.reloadTimers.push(subscription);
    }
  }

  private reloadSilently(): void {
    this.ordersApi.list(this.page(), this.pageSize).subscribe({
      next: (response) => {
        this.orders.set(response.data);
        this.page.set(response.meta.page);
        this.total.set(response.meta.total);
        this.totalPages.set(response.meta.totalPages);
      },
      error: () => void 0,
    });
  }

  formatDate(iso: string): string {
    return new Date(iso).toLocaleDateString('pt-BR', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  }
}
