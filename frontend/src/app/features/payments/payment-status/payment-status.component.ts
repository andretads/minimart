import { Component, inject, signal, computed, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Subject, takeUntil, interval, switchMap, Subscription } from 'rxjs';
import { PaymentsApiService } from '@features/payments/payments-api.service';
import { Payment, PaymentStatus, PAYMENT_STATUS_LABELS } from '@shared/models/payment.models';
import { MoneyPipe } from '@shared/pipes/money.pipe';
import { ButtonComponent } from '@shared/components/button/button.component';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { EmptyStateComponent } from '@shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '@shared/components/error-state/error-state.component';
import { PageShellComponent } from '@shared/components/page-shell/page-shell.component';
import { SkeletonComponent } from '@shared/components/skeleton/skeleton.component';
import { ProductImageComponent } from '@shared/components/product-image/product-image.component';
import { AuthStore } from '@core/auth/auth.store';
import { ToastService } from '@core/toast/toast.service';

@Component({
  selector: 'app-payment-status',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    MoneyPipe,
    ButtonComponent,
    StatusBadgeComponent,
    EmptyStateComponent,
    ErrorStateComponent,
    PageShellComponent,
    SkeletonComponent,
    ProductImageComponent,
  ],
  template: `
    <app-page-shell [title]="'Pagamento ' + (payment()?.id?.slice(0, 8) || 'Estado')">
      <div class="payment-page">
        @if (loading()) {
          <div class="payment-skeleton" aria-busy="true">
            <app-skeleton variant="text" width="40%" />
            <app-skeleton variant="text" width="60%" />
            <app-skeleton variant="rectangular" height="3rem" />
            <app-skeleton variant="rectangular" height="3rem" />
          </div>
        } @else if (error()) {
          <app-error-state
            title="Falha ao carregar pagamento"
            [message]="error()!"
            retryLabel="Tentar novamente"
            (retry)="loadPayment()"
          />
        } @else if (!payment()) {
          <app-empty-state
            title="Pagamento não encontrado"
            message="O pagamento que você procura não existe."
            icon="credit-card"
          />
        } @else {
          <div class="payment-layout">
            <header class="payment-head surface-card" [class]="payment()!.status">
              <span class="payment-icon"><i [class]="paymentIcon()"></i></span>
              <div class="payment-identity">
                <span class="eyebrow">Status do pagamento</span>
                <h1>#{{ payment()!.id.slice(0, 8).toUpperCase() }}</h1>
                <p>Criado em {{ formatDate(payment()!.createdAt) }}</p>
              </div>
              <app-status-badge [status]="payment()!.status" [label]="statusLabel()" size="lg" />
            </header>

            @if (payment()!.status === 'pending') {
              <div class="progress-card surface-card">
                <div class="progress-steps">
                  <div class="progress-step active">
                    <span class="step-circle"><i class="icon icon-clock"></i></span>
                    <span>Iniciado</span>
                  </div>
                  <span class="progress-line"></span>
                  <div class="progress-step">
                    <span class="step-circle"><i class="icon icon-credit-card"></i></span>
                    <span>Processando</span>
                  </div>
                  <span class="progress-line"></span>
                  <div class="progress-step">
                    <span class="step-circle"><i class="icon icon-check"></i></span>
                    <span>Concluído</span>
                  </div>
                </div>
              </div>
            }

            <div class="details-card surface-card">
              <section class="detail-section">
                <div class="section-head">
                  <h2>Estado</h2>
                  <app-status-badge
                    [status]="payment()!.status"
                    [label]="statusLabel()"
                    size="sm"
                  />
                </div>
                <p class="status-description">{{ statusDescription() }}</p>
              </section>

              <section class="detail-section">
                <h2>Detalhes</h2>
                <dl class="detail-list">
                  <div class="detail-row">
                    <dt>Valor</dt>
                    <dd class="amount">
                      {{ payment()!.amountCents | money: payment()!.currency }}
                    </dd>
                  </div>
                  <div class="detail-row">
                    <dt>Moeda</dt>
                    <dd>{{ payment()!.currency }}</dd>
                  </div>
                  <div class="detail-row">
                    <dt>Método</dt>
                    <dd class="capitalize">{{ payment()!.method }}</dd>
                  </div>
                  <div class="detail-row">
                    <dt>Provedor</dt>
                    <dd>{{ payment()!.provider }}</dd>
                  </div>
                  @if (payment()!.providerRef) {
                    <div class="detail-row">
                      <dt>Referência do provedor</dt>
                      <dd class="mono">{{ payment()!.providerRef }}</dd>
                    </div>
                  }
                  @if (payment()!.failureReason) {
                    <div class="detail-row failure">
                      <dt>Motivo da falha</dt>
                      <dd>{{ payment()!.failureReason }}</dd>
                    </div>
                  }
                </dl>
              </section>

              <section class="detail-section">
                <h2>Pedido</h2>
                <div class="order-info">
                  <app-product-image
                    [productId]="payment()!.orderId"
                    [name]="'Pedido ' + payment()!.orderId.slice(0, 8)"
                    class="order-thumb"
                  />
                  <div class="order-copy">
                    <strong>Pedido #{{ payment()!.orderId.slice(0, 8).toUpperCase() }}</strong>
                    <span>Total {{ payment()!.amountCents | money: payment()!.currency }}</span>
                  </div>
                  <a [routerLink]="['/orders', payment()!.orderId]" class="order-link">
                    Ver pedido<i class="icon icon-arrow-right"></i>
                  </a>
                </div>
              </section>

              @if (payment()!.status === 'pending') {
                <div class="pending-actions">
                  <app-button
                    variant="outline"
                    size="sm"
                    label="Atualizar status"
                    (clicked)="refresh()"
                  />
                  @if (autoRefreshActive()) {
                    <span class="auto-refresh"
                      ><i class="icon icon-refresh"></i>Atualização automática a cada 5s</span
                    >
                  }
                </div>
              }
            </div>

            <div class="bottom-actions">
              <a [routerLink]="['/orders', payment()!.orderId]" class="side-link"
                ><i class="icon icon-arrow-left"></i>Voltar ao pedido</a
              >
              <a routerLink="/products" class="side-link"
                ><i class="icon icon-bag"></i>Continuar comprando</a
              >
            </div>
          </div>
        }
      </div>
    </app-page-shell>
  `,
  styles: [
    `
      .payment-page {
        max-width: 48rem;
        margin: 0 auto;
      }
      .payment-skeleton {
        display: grid;
        gap: 1rem;
      }
      .payment-layout {
        display: grid;
        gap: 1.25rem;
      }

      .payment-head {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 1rem;
        padding: clamp(1.25rem, 3vw, 1.75rem);
      }
      .payment-icon {
        display: grid;
        place-items: center;
        flex: 0 0 auto;
        width: 3.75rem;
        height: 3.75rem;
        border-radius: 1.1rem;
        background: var(--color-warning-bg);
        color: var(--color-warning);
      }
      .payment-head.succeeded .payment-icon {
        background: var(--color-success-bg);
        color: var(--color-success);
      }
      .payment-head.failed .payment-icon {
        background: var(--color-danger-bg);
        color: var(--color-danger);
      }
      .payment-head.unknown .payment-icon {
        background: var(--color-warning-bg);
        color: var(--color-warning);
      }
      .payment-identity {
        flex: 1;
        min-width: 12rem;
      }
      .payment-identity h1 {
        margin: 0.15rem 0 0.2rem;
        font-size: clamp(1.35rem, 2.6vw, 1.75rem);
        font-weight: 750;
        letter-spacing: -0.03em;
      }
      .payment-identity p {
        margin: 0;
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }

      .progress-card {
        padding: 1.25rem;
        overflow-x: auto;
      }
      .progress-steps {
        display: flex;
        align-items: center;
        min-width: 22rem;
      }
      .progress-step {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.5rem;
        font-size: 0.6875rem;
        color: var(--color-text-muted);
      }
      .progress-step.active {
        color: var(--color-accent);
        font-weight: 600;
      }
      .step-circle {
        display: grid;
        place-items: center;
        width: 2.25rem;
        height: 2.25rem;
        border: 1.5px solid var(--color-border);
        border-radius: 999px;
        background: var(--color-surface);
      }
      .progress-step.active .step-circle {
        border-color: var(--color-accent);
        color: var(--color-accent);
        box-shadow: 0 0 0 4px color-mix(in oklab, var(--accent) 14%, transparent);
      }
      .progress-line {
        flex: 1;
        height: 2px;
        margin: 0 0.5rem 1.15rem;
        background: var(--color-border);
      }

      .details-card {
        padding: clamp(1.25rem, 3vw, 1.75rem);
      }
      .detail-section {
        margin-bottom: 1.4rem;
        padding-bottom: 1.4rem;
        border-bottom: 1px solid var(--color-border);
      }
      .detail-section:last-of-type {
        margin-bottom: 0;
        padding-bottom: 0;
        border-bottom: 0;
      }
      .detail-section h2,
      .section-head h2 {
        margin: 0 0 0.75rem;
        font-size: 0.9375rem;
        font-weight: 700;
        letter-spacing: -0.01em;
      }
      .section-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        margin-bottom: 0.5rem;
      }
      .section-head h2 {
        margin: 0;
      }
      .status-description {
        margin: 0;
        font-size: 0.875rem;
        line-height: 1.65;
        color: var(--color-text-muted);
      }

      .detail-list {
        display: grid;
        margin: 0;
      }
      .detail-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        padding: 0.6rem 0;
        border-bottom: 1px solid var(--color-border);
        font-size: 0.8125rem;
      }
      .detail-row:last-child {
        border-bottom: 0;
      }
      .detail-row dt {
        color: var(--color-text-muted);
      }
      .detail-row dd {
        margin: 0;
        font-weight: 500;
        text-align: right;
      }
      .detail-row .amount {
        font-size: 1.15rem;
        font-weight: 750;
        color: var(--color-accent);
      }
      .detail-row .mono {
        font-family: var(--font-mono);
        font-size: 0.6875rem;
        overflow-wrap: anywhere;
      }
      .capitalize {
        text-transform: capitalize;
      }
      .detail-row.failure dd {
        color: var(--color-danger);
      }

      .order-info {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.85rem;
        padding: 1rem;
        border: 1px solid var(--color-border);
        border-radius: 0.9rem;
        background: var(--color-surface-muted);
      }
      .order-thumb {
        flex: 0 0 auto;
        width: 2.75rem;
        height: 2.75rem;
        border-radius: 0.7rem;
        overflow: hidden;
      }
      .order-copy {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.1rem;
        min-width: 10rem;
      }
      .order-copy strong {
        font-size: 0.8125rem;
        font-weight: 600;
      }
      .order-copy span {
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }
      .order-link {
        display: inline-flex;
        align-items: center;
        gap: 0.35rem;
        font-size: 0.75rem;
        font-weight: 600;
        color: var(--color-accent);
        text-decoration: none;
      }
      .order-link:hover {
        text-decoration: underline;
      }

      .pending-actions {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.85rem;
        margin-top: 1.25rem;
        padding-top: 1.25rem;
        border-top: 1px solid var(--color-border);
      }
      .auto-refresh {
        display: inline-flex;
        align-items: center;
        gap: 0.4rem;
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }
      .auto-refresh .icon {
        animation: spin 1.4s linear infinite;
        color: var(--color-accent);
      }

      .bottom-actions {
        display: grid;
        gap: 0.5rem;
      }
      @media (min-width: 640px) {
        .bottom-actions {
          grid-template-columns: 1fr 1fr;
        }
      }
      .side-link {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.5rem;
        padding: 0.75rem 1rem;
        border: 1px solid var(--color-border);
        border-radius: 0.8rem;
        background: var(--color-surface);
        font-size: 0.8125rem;
        font-weight: 500;
        color: var(--color-text-muted);
        text-decoration: none;
        transition:
          border-color 0.2s,
          color 0.2s;
      }
      .side-link:hover {
        border-color: color-mix(in oklab, var(--accent) 40%, transparent);
        color: var(--color-accent);
      }
      @keyframes spin {
        to {
          transform: rotate(360deg);
        }
      }

      @media (max-width: 560px) {
        .payment-head {
          align-items: flex-start;
        }
        .payment-head app-status-badge {
          width: 100%;
        }
      }
    `,
  ],
})
export class PaymentStatusComponent implements OnInit, OnDestroy {
  private readonly paymentsApi = inject(PaymentsApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly authStore = inject(AuthStore);
  private readonly toast = inject(ToastService);
  private readonly destroy$ = new Subject<void>();

  payment = signal<Payment | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);
  autoRefreshActive = signal(false);
  private pollSubscription: Subscription | undefined;

  ngOnInit(): void {
    this.route.paramMap.pipe(takeUntil(this.destroy$)).subscribe(() => {
      this.loadPayment();
    });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    if (this.pollSubscription) this.pollSubscription.unsubscribe();
  }

  loadPayment(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) return;
    this.loading.set(true);
    this.error.set(null);
    this.paymentsApi.get(id).subscribe({
      next: (payment) => {
        if (payment.userId !== this.authStore.userId()) {
          this.error.set('Pagamento não encontrado');
          this.payment.set(null);
        } else {
          this.payment.set(payment);
          this.startAutoRefresh(payment.status);
        }
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.message);
        this.loading.set(false);
      },
    });
  }

  startAutoRefresh(status: PaymentStatus): void {
    if (this.pollSubscription) this.pollSubscription.unsubscribe();
    if (status !== 'pending') {
      this.autoRefreshActive.set(false);
      return;
    }
    this.autoRefreshActive.set(true);
    this.pollSubscription = interval(5000)
      .pipe(
        switchMap(() => this.paymentsApi.get(this.payment()!.id)),
        takeUntil(this.destroy$),
      )
      .subscribe({
        next: (payment) => {
          this.payment.set(payment);
          if (payment.status !== 'pending') {
            this.autoRefreshActive.set(false);
            if (this.pollSubscription) this.pollSubscription.unsubscribe();
            this.toast.success(
              `Pagamento ${payment.status === 'succeeded' ? 'aprovado' : payment.status === 'failed' ? 'reprovado' : payment.status === 'refunded' ? 'reembolsado' : (payment.status as string) === 'pending' ? 'pendente' : 'atualizado'}`,
            );
          }
        },
        error: () => void 0,
      });
  }

  refresh(): void {
    const p = this.payment();
    if (!p) return;
    this.paymentsApi.get(p.id).subscribe({
      next: (payment) => {
        this.payment.set(payment);
        this.startAutoRefresh(payment.status);
      },
      error: (err) => this.toast.error(err.message),
    });
  }

  statusLabel = computed(() =>
    this.payment() ? PAYMENT_STATUS_LABELS[this.payment()!.status] : '',
  );

  paymentStatusClass = computed(() => {
    const p = this.payment();
    if (!p) return '';
    return p.status;
  });

  paymentIcon = computed(() => {
    const p = this.payment();
    if (!p) return 'icon';
    return `icon icon-lg icon-pay-${p.status}`;
  });

  statusDescription = computed(() => {
    const p = this.payment();
    if (!p) return '';
    switch (p.status) {
      case 'pending':
        return 'Seu pagamento está sendo processado. Isso geralmente leva alguns segundos.';
      case 'succeeded':
        return 'Pagamento concluído com sucesso. Seu pedido será atualizado em breve.';
      case 'failed':
        return 'O pagamento foi recusado ou falhou. Você pode tentar novamente pela página do pedido.';
      case 'unknown':
        return 'O status do pagamento é incerto. A cobrança pode ter sido processada. Fale com o suporte se necessário.';
      case 'refunded':
        return 'O pagamento foi reembolsado porque o pedido foi cancelado. O valor retorna ao método de pagamento original.';
      default:
        return '';
    }
  });

  formatDate(iso: string): string {
    return new Date(iso).toLocaleDateString('pt-BR', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }
}
