import { Component, inject, signal, computed, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Subject, takeUntil, interval, timer, Subscription } from 'rxjs';
import { OrdersApiService } from '../orders-api.service';
import { PaymentsApiService } from '@features/payments/payments-api.service';
import { ShippingApiService } from '@features/shipping/shipping-api.service';
import { CreatePaymentRequest } from '@shared/models/payment.models';
import {
  Order,
  OrderStatus,
  ORDER_STATUS_LABELS,
  CANCELLABLE_STATUSES,
} from '@shared/models/order.models';
import { Payment, PAYMENT_STATUS_LABELS } from '@shared/models/payment.models';
import {
  Shipment,
  TrackingEvent,
  DestinationSnapshot,
  SHIPMENT_STATUS_LABELS,
} from '@shared/models/shipping.models';
import { MoneyPipe } from '@shared/pipes/money.pipe';
import { autoCancelAt, formatRemaining } from '@shared/utils/order-deadline';
import { ButtonComponent } from '@shared/components/button/button.component';
import { StatusBadgeComponent } from '@shared/components/status-badge/status-badge.component';
import { ConfirmDialogComponent } from '@shared/components/confirm-dialog/confirm-dialog.component';
import { EmptyStateComponent } from '@shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '@shared/components/error-state/error-state.component';
import { PageShellComponent } from '@shared/components/page-shell/page-shell.component';
import { SkeletonComponent } from '@shared/components/skeleton/skeleton.component';
import { ProductImageComponent } from '@shared/components/product-image/product-image.component';
import { AuthStore } from '@core/auth/auth.store';
import { IdempotencyKeyService } from '@core/idempotency/idempotency-key.service';
import { ToastService } from '@core/toast/toast.service';

@Component({
  selector: 'app-order-details',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    MoneyPipe,
    ButtonComponent,
    StatusBadgeComponent,
    ConfirmDialogComponent,
    EmptyStateComponent,
    ErrorStateComponent,
    PageShellComponent,
    SkeletonComponent,
    ProductImageComponent,
  ],
  template: `
    <app-page-shell [title]="'Pedido ' + (order()?.id?.slice(0, 8) || 'Detalhes')">
      <div class="order-page">
        @if (loading()) {
          <div class="order-skeleton" aria-busy="true">
            <app-skeleton variant="text" width="40%" />
            <app-skeleton variant="text" width="60%" />
            @for (i of [1, 2, 3]; track i) {
              <app-skeleton variant="rectangular" height="3rem" />
            }
          </div>
        } @else if (error()) {
          <app-error-state
            title="Falha ao carregar pedido"
            [message]="error()!"
            retryLabel="Tentar novamente"
            (retry)="loadOrder()"
          />
        } @else if (!order()) {
          <app-empty-state
            title="Pedido não encontrado"
            message="O pedido que você procura não existe."
            icon="package"
          />
        } @else {
          <div class="order-layout">
            <section class="order-main">
              <header class="order-head surface-card">
                <div class="order-identity">
                  <span class="order-icon"><i class="icon icon-receipt icon-lg"></i></span>
                  <div>
                    <span class="eyebrow">Detalhes do pedido</span>
                    <h1>#{{ order()!.id.slice(0, 8).toUpperCase() }}</h1>
                    <p>Feito em {{ formatDate(order()!.createdAt) }}</p>
                  </div>
                </div>
                <div class="order-status">
                  <app-status-badge
                    [status]="order()!.status"
                    [label]="orderStatusLabel()"
                    size="lg"
                  />
                  @if (order()!.cancelledAt) {
                    <span class="cancelled-note"
                      >Cancelado em {{ formatDate(order()!.cancelledAt!) }}</span
                    >
                  }
                </div>
              </header>

              @if (order()!.status !== 'cancelled') {
                <div class="tracking-card surface-card">
                  <div class="tracking-steps">
                    <div class="tracking-step completed">
                      <span class="step-circle"><i class="icon icon-check"></i></span>
                      <span class="step-label">Pedido recebido</span>
                    </div>
                    <span class="tracking-line"></span>
                    <div
                      class="tracking-step"
                      [class.completed]="order()!.status !== 'placed'"
                      [class.active]="order()!.status === 'placed'"
                    >
                      <span class="step-circle"><i class="icon icon-credit-card"></i></span>
                      <span class="step-label">Pagamento</span>
                    </div>
                    <span class="tracking-line"></span>
                    <div
                      class="tracking-step"
                      [class.completed]="order()!.status === 'paid'"
                      [class.active]="order()!.status === 'paid'"
                    >
                      <span class="step-circle"><i class="icon icon-package"></i></span>
                      <span class="step-label">Processando</span>
                    </div>
                    <span class="tracking-line"></span>
                    <div class="tracking-step">
                      <span class="step-circle"><i class="icon icon-truck"></i></span>
                      <span class="step-label">Enviado</span>
                    </div>
                    <span class="tracking-line"></span>
                    <div class="tracking-step">
                      <span class="step-circle"><i class="icon icon-home"></i></span>
                      <span class="step-label">Entregue</span>
                    </div>
                  </div>
                </div>
              }

              <div class="panel-card surface-card">
                <div class="panel-head">
                  <h2>Itens</h2>
                  <span class="count-pill"
                    >{{ order()!.items.length }} item{{
                      order()!.items.length === 1 ? '' : 's'
                    }}</span
                  >
                </div>
                <div class="table-wrap">
                  <table class="order-table">
                    <thead>
                      <tr>
                        <th class="col-img"><span class="sr-only">Imagem</span></th>
                        <th>Produto</th>
                        <th class="num">Preço</th>
                        <th class="num">Qtd</th>
                        <th class="num">Total</th>
                      </tr>
                    </thead>
                    <tbody>
                      @for (item of order()!.items; track item.productId) {
                        <tr>
                          <td>
                            <app-product-image
                              [productId]="item.productId"
                              [name]="item.name"
                              class="thumb"
                            />
                          </td>
                          <td>
                            <div class="product-info">
                              <strong>{{ item.name }}</strong>
                              <span>SKU: PRD-{{ item.productId.slice(0, 8).toUpperCase() }}</span>
                            </div>
                          </td>
                          <td class="num">{{ item.unitPriceCents | money }}</td>
                          <td class="num">
                            <span class="qty-pill">{{ item.quantity }}</span>
                          </td>
                          <td class="num strong">{{ item.lineTotalCents | money }}</td>
                        </tr>
                      }
                    </tbody>
                  </table>
                </div>
              </div>

              <div class="totals-card surface-card">
                <div class="totals-row">
                  <span>Subtotal</span
                  ><span>{{
                    order()!.totalCents - (order()!.shippingCostCents || 0) | money
                  }}</span>
                </div>
                <div class="totals-row">
                  <span>Frete</span>
                  @if ((order()!.shippingCostCents ?? 0) > 0) {
                    <span>{{ order()!.shippingCostCents | money }}</span>
                  } @else {
                    <span class="free">Grátis</span>
                  }
                </div>
                @if (order()!.status === 'cancelled') {
                  <div class="totals-row refund">
                    <span>Reembolso</span><span>{{ order()!.totalCents | money }}</span>
                  </div>
                }
                <div class="totals-row total">
                  <span>{{ order()!.status === 'cancelled' ? 'Reembolsado' : 'Total' }}</span>
                  <span>{{ order()!.totalCents | money }}</span>
                </div>
              </div>

              @if (payment()) {
                <div class="panel-card surface-card">
                  <div class="panel-head">
                    <h2>Pagamento</h2>
                    <app-status-badge
                      [status]="payment()!.status"
                      [label]="paymentStatusLabel()"
                      size="lg"
                    />
                  </div>
                  <div class="payment-row">
                    <span class="order-icon sm"><i class="icon icon-credit-card"></i></span>
                    <div class="payment-copy">
                      <strong>{{ payment()!.method }}</strong>
                      <span
                        >{{ payment()!.provider }} · {{ payment()!.currency }} ·
                        {{ payment()!.id.slice(0, 8) }}…</span
                      >
                    </div>
                    <div class="payment-amount">
                      {{ payment()!.amountCents | money: payment()!.currency }}
                    </div>
                  </div>
                  @if (payment()!.failureReason) {
                    <p class="failure">
                      <i class="icon icon-alert"></i>{{ payment()!.failureReason }}
                    </p>
                  }
                  <div class="payment-actions">
                    <app-button
                      variant="outline"
                      size="sm"
                      label="Atualizar status"
                      (clicked)="refreshPayment()"
                    />
                    @if (autoRefreshActive()) {
                      <span class="auto-refresh"
                        ><i class="icon icon-refresh"></i>Atualizando automaticamente…</span
                      >
                    }
                  </div>
                </div>
              } @else if (canPay()) {
                <div class="panel-card surface-card pay-panel">
                  <span class="pay-icon"><i class="icon icon-clock icon-lg"></i></span>
                  <h2>Pagamento necessário</h2>
                  <p class="auto-cancel-warning">
                    Você tem 5 minutos para concluir o pagamento. Cancelamento automático em
                    {{ remainingLabel() }}.
                  </p>
                  <p>Este pedido está aguardando pagamento. Conclua sua compra para confirmá-lo.</p>
                  <app-button
                    label="Pagar agora"
                    variant="primary"
                    size="lg"
                    [fullWidth]="true"
                    [loading]="creatingPayment()"
                    (clicked)="createPayment()"
                  />
                </div>
              }

              @if (canCancel()) {
                <div class="cancel-banner">
                  <span class="cancel-icon"><i class="icon icon-alert"></i></span>
                  <div class="cancel-copy">
                    <h3>Cancelar pedido</h3>
                    <p>Isso libera o estoque reservado e não pode ser desfeito.</p>
                  </div>
                  <app-button
                    variant="danger"
                    size="sm"
                    label="Cancelar pedido"
                    [loading]="cancelling()"
                    (clicked)="confirmCancel.set(true)"
                  />
                </div>
              }

              @if (order()!.shippingQuoteId || shipment()) {
                <div class="panel-card surface-card">
                  <div class="panel-head">
                    <h2>Entrega</h2>
                    @if (shipment(); as ship) {
                      <app-status-badge
                        [status]="ship.status"
                        [label]="getShipmentStatusLabel(ship.status)"
                        size="sm"
                      />
                    }
                  </div>
                  <div class="shipping-info">
                    <div class="shipping-row">
                      <span class="shipping-label">Método</span>
                      <span class="shipping-value">{{
                        order()!.shippingMethodName || 'Não informado'
                      }}</span>
                    </div>
                    <div class="shipping-row">
                      <span class="shipping-label">Endereço de entrega</span>
                      <span class="shipping-value">{{
                        formatAddress(order()!.destinationSnapshot)
                      }}</span>
                    </div>
                    @if (shipment(); as ship) {
                      @if (ship.trackingCode) {
                        <div class="shipping-row">
                          <span class="shipping-label">Código de rastreio</span>
                          <span class="shipping-value tracking-code">{{ ship.trackingCode }}</span>
                        </div>
                      }
                      @if (ship.carrier) {
                        <div class="shipping-row">
                          <span class="shipping-label">Transportadora</span>
                          <span class="shipping-value">{{ ship.carrier }}</span>
                        </div>
                      }
                      @if (ship.shippedAt) {
                        <div class="shipping-row">
                          <span class="shipping-label">Enviado em</span>
                          <span class="shipping-value">{{ formatDate(ship.shippedAt) }}</span>
                        </div>
                      }
                      @if (ship.deliveredAt) {
                        <div class="shipping-row">
                          <span class="shipping-label">Entregue em</span>
                          <span class="shipping-value">{{ formatDate(ship.deliveredAt) }}</span>
                        </div>
                      }
                    }
                  </div>
                  @if (trackingEvents().length > 0) {
                    <div class="tracking-events">
                      <h3>Histórico de rastreamento</h3>
                      @for (event of trackingEvents(); track event.id) {
                        <div class="tracking-event">
                          <div
                            class="event-dot"
                            [class.completed]="isEventCompleted(event.status)"
                          ></div>
                          <div class="event-content">
                            <div class="event-header">
                              <span class="event-status">{{
                                getTrackingStatusLabel(event.status)
                              }}</span>
                              <span class="event-time">{{ formatDateTime(event.occurredAt) }}</span>
                            </div>
                            <div class="event-description">{{ event.description }}</div>
                            @if (event.location) {
                              <div class="event-location">{{ event.location }}</div>
                            }
                          </div>
                        </div>
                      }
                    </div>
                  }
                </div>
              }
            </section>

            <aside class="order-sidebar">
              <div class="summary-card surface-card">
                <h2>Resumo</h2>
                <div class="summary-row">
                  <span>Itens</span><span>{{ order()!.items.length }}</span>
                </div>
                <div class="summary-row">
                  <span>Subtotal</span
                  ><span>{{
                    order()!.totalCents - (order()!.shippingCostCents || 0) | money
                  }}</span>
                </div>
                <div class="summary-row">
                  <span>Frete</span>
                  @if ((order()!.shippingCostCents ?? 0) > 0) {
                    <span>{{ order()!.shippingCostCents | money }}</span>
                  } @else {
                    <span class="free">Grátis</span>
                  }
                </div>
                <div class="summary-row total">
                  <span>Total</span><span>{{ order()!.totalCents | money }}</span>
                </div>
                <div class="summary-divider"></div>
                <div class="summary-row">
                  <span>Status</span
                  ><app-status-badge [status]="order()!.status" [label]="orderStatusLabel()" />
                </div>
                <div class="summary-row">
                  <span>Feito em</span><span>{{ formatDate(order()!.createdAt) }}</span>
                </div>
                <div class="summary-row id-row">
                  <span>ID do pedido</span><code>{{ order()!.id }}</code>
                </div>
              </div>

              <div class="side-actions">
                <a routerLink="/products" class="side-link"
                  ><i class="icon icon-arrow-left"></i>Continuar comprando</a
                >
                <a routerLink="/orders" class="side-link"
                  ><i class="icon icon-receipt"></i>Todos os pedidos</a
                >
              </div>
            </aside>
          </div>
        }
      </div>

      <app-confirm-dialog
        [open]="confirmCancel()"
        title="Cancelar pedido"
        message="Tem certeza que deseja cancelar este pedido? Esta ação não pode ser desfeita."
        confirmLabel="Cancelar pedido"
        confirmVariant="danger"
        [confirmLoading]="cancelling()"
        (close)="confirmCancel.set(false)"
        (confirm)="cancelOrder()"
      />
    </app-page-shell>
  `,
  styles: [
    `
      .order-page {
        max-width: 76rem;
        margin: 0 auto;
      }
      .order-skeleton {
        display: grid;
        gap: 1rem;
      }
      .order-layout {
        display: grid;
        grid-template-columns: 1fr;
        gap: 1.5rem;
        align-items: start;
      }
      @media (min-width: 992px) {
        .order-layout {
          grid-template-columns: minmax(0, 1fr) 20rem;
        }
      }
      .order-main {
        display: grid;
        gap: 1.25rem;
        min-width: 0;
      }
      .order-sidebar {
        display: grid;
        gap: 1rem;
      }
      @media (min-width: 992px) {
        .order-sidebar {
          position: sticky;
          top: 6rem;
        }
      }

      .order-head {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        flex-wrap: wrap;
        gap: 1rem;
        padding: clamp(1.25rem, 3vw, 1.75rem);
      }
      .order-identity {
        display: flex;
        align-items: center;
        gap: 1rem;
      }
      .order-icon {
        display: grid;
        place-items: center;
        flex: 0 0 auto;
        width: 3.5rem;
        height: 3.5rem;
        border-radius: 1rem;
        background: color-mix(in oklab, var(--accent) 11%, transparent);
        color: var(--color-accent);
      }
      .order-icon.sm {
        width: 2.6rem;
        height: 2.6rem;
        border-radius: 0.8rem;
      }
      .order-identity h1 {
        margin: 0.15rem 0 0.2rem;
        font-size: clamp(1.4rem, 2.6vw, 1.9rem);
        font-weight: 750;
        letter-spacing: -0.03em;
      }
      .order-identity p {
        margin: 0;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .order-status {
        display: flex;
        flex-direction: column;
        align-items: flex-end;
        gap: 0.4rem;
      }
      .cancelled-note {
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }

      .tracking-card {
        padding: 1.4rem 1.25rem;
        overflow-x: auto;
      }
      .tracking-steps {
        display: flex;
        align-items: center;
        min-width: 28rem;
      }
      .tracking-step {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.5rem;
        color: var(--color-text-muted);
      }
      .tracking-step.completed,
      .tracking-step.active {
        color: var(--color-accent);
      }
      .step-circle {
        display: grid;
        place-items: center;
        width: 2.5rem;
        height: 2.5rem;
        border: 1.5px solid var(--color-border);
        border-radius: 999px;
        background: var(--color-surface);
        transition: all 0.25s;
      }
      .tracking-step.completed .step-circle {
        border-color: var(--color-accent);
        background: var(--color-accent);
        color: var(--accent-foreground);
      }
      .tracking-step.active .step-circle {
        box-shadow: 0 0 0 4px color-mix(in oklab, var(--accent) 14%, transparent);
      }
      .step-label {
        font-size: 0.6875rem;
        font-weight: 500;
        white-space: nowrap;
      }
      .tracking-line {
        flex: 1;
        height: 2px;
        margin: 0 0.35rem 1.2rem;
        background: var(--color-border);
      }
      .tracking-step.completed + .tracking-line {
        background: color-mix(in oklab, var(--accent) 55%, var(--color-border));
      }

      .panel-card {
        padding: clamp(1.25rem, 3vw, 1.6rem);
      }
      .panel-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        margin-bottom: 1rem;
      }
      .panel-head h2 {
        margin: 0;
        font-size: 1.05rem;
        font-weight: 700;
        letter-spacing: -0.02em;
      }
      .table-wrap {
        overflow-x: auto;
        margin: 0 -0.35rem;
      }
      .order-table {
        width: 100%;
        border-collapse: collapse;
      }
      .order-table th {
        padding: 0.6rem 0.35rem;
        border-bottom: 1px solid var(--color-border);
        font-size: 0.625rem;
        font-weight: 600;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--color-text-muted);
        text-align: left;
      }
      .order-table td {
        padding: 0.85rem 0.35rem;
        border-bottom: 1px solid var(--color-border);
        font-size: 0.8125rem;
        vertical-align: middle;
      }
      .order-table tbody tr:last-child td {
        border-bottom: 0;
      }
      .order-table .col-img {
        width: 3.75rem;
      }
      .order-table .num {
        text-align: right;
      }
      .order-table .strong {
        font-weight: 700;
        color: var(--color-accent);
      }
      .thumb {
        display: block;
        width: 2.75rem;
        height: 2.75rem;
        border-radius: 0.65rem;
        overflow: hidden;
      }
      .product-info {
        display: flex;
        flex-direction: column;
        gap: 0.15rem;
      }
      .product-info strong {
        font-size: 0.8125rem;
        font-weight: 600;
      }
      .product-info span {
        font-family: var(--font-mono);
        font-size: 0.625rem;
        color: var(--color-text-muted);
      }
      .qty-pill {
        display: inline-block;
        padding: 0.15rem 0.55rem;
        border-radius: 999px;
        background: var(--color-surface-muted);
        font-size: 0.75rem;
        font-weight: 600;
      }

      .totals-card {
        padding: 1.25rem 1.5rem;
      }
      .totals-row {
        display: flex;
        justify-content: space-between;
        padding: 0.4rem 0;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .totals-row .free {
        color: var(--color-accent);
        font-weight: 600;
      }
      .totals-row.refund {
        color: var(--color-danger);
      }
      .totals-row.total {
        margin-top: 0.45rem;
        padding-top: 0.85rem;
        border-top: 1px solid var(--color-border);
        font-size: 1.05rem;
        font-weight: 750;
        color: var(--color-text);
      }
      .totals-row.total span:last-child {
        color: var(--color-accent);
      }

      .payment-row {
        display: flex;
        align-items: center;
        gap: 0.85rem;
        padding: 0.9rem 0;
        border-top: 1px solid var(--color-border);
        border-bottom: 1px solid var(--color-border);
      }
      .payment-copy {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.1rem;
        min-width: 0;
      }
      .payment-copy strong {
        font-size: 0.875rem;
        text-transform: capitalize;
      }
      .payment-copy span {
        font-size: 0.75rem;
        color: var(--color-text-muted);
        overflow-wrap: anywhere;
      }
      .payment-amount {
        font-size: 0.9375rem;
        font-weight: 700;
        color: var(--color-accent);
      }
      .failure {
        display: flex;
        align-items: center;
        gap: 0.4rem;
        margin: 0.85rem 0;
        font-size: 0.75rem;
        color: var(--color-danger);
      }
      .panel-card app-button {
        margin-top: 0.9rem;
      }

      .pay-panel {
        display: flex;
        flex-direction: column;
        align-items: center;
        text-align: center;
        gap: 0.5rem;
      }
      .pay-icon {
        display: grid;
        place-items: center;
        width: 3.25rem;
        height: 3.25rem;
        border-radius: 1rem;
        background: var(--color-warning-bg);
        color: var(--color-warning);
      }
      .pay-panel h2 {
        margin: 0.35rem 0 0;
        font-size: 1.1rem;
        font-weight: 700;
      }
      .pay-panel p {
        max-width: 26rem;
        margin: 0 0 0.5rem;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }

      .cancel-banner {
        display: flex;
        align-items: flex-start;
        flex-wrap: wrap;
        gap: 0.85rem;
        padding: 1.15rem 1.25rem;
        border: 1px solid color-mix(in oklab, var(--color-warning) 35%, transparent);
        border-radius: 1rem;
        background: var(--color-warning-bg);
      }
      .cancel-icon {
        display: grid;
        place-items: center;
        flex: 0 0 auto;
        width: 2.25rem;
        height: 2.25rem;
        border-radius: 0.7rem;
        background: color-mix(in oklab, var(--color-warning) 14%, transparent);
        color: var(--color-warning);
      }
      .cancel-copy {
        flex: 1;
        min-width: 12rem;
      }
      .cancel-copy h3 {
        margin: 0 0 0.15rem;
        font-size: 0.875rem;
        font-weight: 700;
      }
      .cancel-copy p {
        margin: 0;
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }

      .summary-card {
        padding: 1.4rem;
      }
      .summary-card h2 {
        margin: 0 0 1rem;
        font-size: 1rem;
        font-weight: 700;
      }
      .summary-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        padding: 0.5rem 0;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .summary-row > span:last-child {
        color: var(--color-text);
        font-weight: 500;
      }
      .summary-row .free {
        color: var(--color-accent);
      }
      .summary-row.total {
        margin-top: 0.35rem;
        padding-top: 0.85rem;
        border-top: 1px solid var(--color-border);
        font-size: 1rem;
        font-weight: 700;
        color: var(--color-text);
      }
      .summary-row.total > span:last-child {
        color: var(--color-accent);
        font-weight: 750;
      }
      .summary-divider {
        height: 1px;
        margin: 0.6rem 0;
        background: var(--color-border);
      }
      .summary-row.id-row {
        align-items: flex-start;
      }
      .summary-row code {
        font-family: var(--font-mono);
        font-size: 0.625rem;
        text-align: right;
        overflow-wrap: anywhere;
      }
      .side-actions {
        display: grid;
        gap: 0.5rem;
      }
      .side-link {
        display: flex;
        align-items: center;
        gap: 0.55rem;
        padding: 0.7rem 0.9rem;
        border: 1px solid var(--color-border);
        border-radius: 0.8rem;
        background: var(--color-surface);
        color: var(--color-text-muted);
        font-size: 0.8125rem;
        font-weight: 500;
        text-decoration: none;
        transition:
          border-color 0.2s,
          color 0.2s;
      }
      .side-link:hover {
        border-color: color-mix(in oklab, var(--accent) 40%, transparent);
        color: var(--color-accent);
      }
      .side-link .icon:last-child {
        margin-left: auto;
      }
      .side-link .icon-receipt {
        margin-left: 0;
      }

      .payment-actions {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.85rem;
        margin-top: 0.9rem;
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
      @keyframes spin {
        to {
          transform: rotate(360deg);
        }
      }

      .shipping-info {
        margin-top: 1rem;
      }
      .shipping-row {
        display: flex;
        align-items: flex-start;
        gap: 1rem;
        padding: 0.5rem 0;
        border-bottom: 1px solid var(--color-border);
      }
      .shipping-row:last-child {
        border-bottom: none;
      }
      .shipping-label {
        flex: 0 0 120px;
        font-size: 1rem;
        font-weight: 700;
        color: var(--color-text);
      }
      .shipping-value {
        flex: 1;
        font-size: 0.8125rem;
        color: var(--color-text);
        word-break: break-word;
      }
      .tracking-code {
        font-family: var(--font-mono);
        background: var(--color-surface-muted);
        padding: 0.2rem 0.4rem;
        border-radius: 0.4rem;
      }

      .tracking-events {
        margin-top: 1.5rem;
        padding-top: 1rem;
        border-top: 1px solid var(--color-border);
      }
      .tracking-events h3 {
        margin: 0 0 1rem;
        font-size: 0.875rem;
        font-weight: 700;
      }
      .tracking-event {
        display: flex;
        gap: 0.75rem;
        padding: 0.75rem 0;
      }
      .event-dot {
        flex: 0 0 auto;
        width: 0.75rem;
        height: 0.75rem;
        border-radius: 50%;
        background: var(--color-border);
        margin-top: 0.25rem;
      }
      .event-dot.completed {
        background: var(--color-accent);
      }
      .event-content {
        flex: 1;
        min-width: 0;
      }
      .event-header {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        margin-bottom: 0.25rem;
      }
      .event-status {
        font-size: 0.6875rem;
        font-weight: 600;
        color: var(--color-accent);
        text-transform: uppercase;
        letter-spacing: 0.05em;
      }
      .event-time {
        font-size: 0.6875rem;
        color: var(--color-text-muted);
        white-space: nowrap;
      }
      .event-description {
        font-size: 0.75rem;
        color: var(--color-text);
        margin-bottom: 0.15rem;
      }
      .event-location {
        font-size: 0.6875rem;
        color: var(--color-text-muted);
      }

      @media (max-width: 640px) {
        .order-head {
          flex-direction: column;
        }
        .order-status {
          align-items: flex-start;
        }
        .order-table th:nth-child(3),
        .order-table td:nth-child(3) {
          display: none;
        }
      }
    `,
  ],
})
export class OrderDetailsComponent implements OnInit, OnDestroy {
  private readonly ordersApi = inject(OrdersApiService);
  private readonly paymentsApi = inject(PaymentsApiService);
  private readonly shippingApi = inject(ShippingApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly authStore = inject(AuthStore);
  private readonly idempotency = inject(IdempotencyKeyService);
  private readonly toast = inject(ToastService);
  private readonly destroy$ = new Subject<void>();

  order = signal<Order | null>(null);
  payment = signal<Payment | null>(null);
  shipment = signal<Shipment | null>(null);
  trackingEvents = signal<TrackingEvent[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  creatingPayment = signal(false);
  cancelling = signal(false);
  confirmCancel = signal(false);
  autoRefreshActive = signal(false);
  now = signal(Date.now());
  private countdownSubscription?: Subscription;
  private autoCancelNotified = false;
  private autoCancelPending = false;
  private pollSubscription?: Subscription;
  private shipmentRetrySubscription?: Subscription;
  private shipmentRetryAttempt = 0;
  private readonly maxSyncAttempts = 8;
  private readonly shipmentRetryAttempts = 5;
  private readonly shipmentRetryDelayMs = 2000;

  ngOnInit(): void {
    this.route.paramMap.pipe(takeUntil(this.destroy$)).subscribe((params) => {
      const id = params.get('id')!;
      this.stopOrderSync();
      this.loadOrder();
      this.loadPaymentForOrder(id);
    });
    this.countdownSubscription = interval(1000)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        this.now.set(Date.now());
        this.checkAutoCancelExpiry();
      });
  }

  ngOnDestroy(): void {
    this.stopOrderSync();
    this.stopShipmentRetry();
    this.countdownSubscription?.unsubscribe();
    this.countdownSubscription = undefined;
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadOrder(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) return;
    this.loading.set(true);
    this.error.set(null);
    this.shipment.set(null);
    this.trackingEvents.set([]);
    this.stopShipmentRetry();
    this.ordersApi.get(id).subscribe({
      next: (order) => {
        if (order.userId !== this.authStore.userId()) {
          this.error.set('Pedido não encontrado');
          this.order.set(null);
        } else {
          this.order.set(order);
          this.loadShipment(order.id);
          this.syncOrderUntilSettled();
        }
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.message);
        this.loading.set(false);
      },
    });
  }

  /**
   * Loads the shipment the shipping domain created for this order. The order
   * API never returns the shipment, so it is read from the shipping API and
   * kept in its own signal: the delivery section shows its tracking state, and
   * reloading the order never drops it. An order that is not shipped yet has
   * no shipment, and that is not an error.
   */
  private loadShipment(orderId: string): void {
    this.stopShipmentRetry();
    this.shipmentRetryAttempt = 0;
    this.fetchShipment(orderId);
  }

  private fetchShipment(orderId: string): void {
    this.shippingApi
      .getShipments(orderId)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (shipments) => {
          const shipment = shipments[0];
          if (!shipment) {
            this.shipment.set(null);
            this.trackingEvents.set([]);
            this.scheduleShipmentRetry(orderId);
            return;
          }
          this.shipment.set(shipment);
          this.loadTrackingEvents(shipment.id);
        },
        error: () => void 0,
      });
  }

  /**
   * The shipment is created when the order is paid, so it can show up shortly
   * after the order reaches that state. The read is retried a few times while
   * the order is paid, so the delivery section fills in without a manual page
   * reload. An order that is not paid yet has no shipment to wait for.
   */
  private scheduleShipmentRetry(orderId: string): void {
    if (this.order()?.status !== 'paid') return;
    if (this.shipmentRetryAttempt >= this.shipmentRetryAttempts) return;
    this.shipmentRetryAttempt++;
    this.shipmentRetrySubscription = timer(this.shipmentRetryDelayMs)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        this.fetchShipment(orderId);
      });
  }

  private stopShipmentRetry(): void {
    this.shipmentRetrySubscription?.unsubscribe();
    this.shipmentRetrySubscription = undefined;
  }

  private loadTrackingEvents(shipmentId: string): void {
    this.shippingApi
      .getTrackingEvents(shipmentId)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (events) => this.trackingEvents.set(events),
        error: () => void 0,
      });
  }

  loadPaymentForOrder(orderId: string): void {
    const storedPaymentId = sessionStorage.getItem(`mm_payment_${orderId}`);
    if (storedPaymentId) {
      this.paymentsApi.get(storedPaymentId).subscribe({
        next: (payment) => {
          this.payment.set(payment);
          this.syncOrderUntilSettled();
        },
        error: () => void 0,
      });
    }
  }

  canPay = computed(() => {
    const o = this.order();
    return o && ['placed'].includes(o.status) && !this.payment();
  });

  deadline = computed(() => (this.order() ? autoCancelAt(this.order()!) : null));
  remainingMs = computed(() => {
    const d = this.deadline();
    return d ? d.getTime() - this.now() : 0;
  });
  remainingLabel = computed(() => formatRemaining(this.remainingMs()));

  canCancel = computed(() => {
    const o = this.order();
    return o && CANCELLABLE_STATUSES.includes(o.status);
  });

  orderStatusLabel = computed(() =>
    this.order() ? ORDER_STATUS_LABELS[this.order()!.status] : '',
  );

  paymentStatusLabel = computed(() =>
    this.payment() ? PAYMENT_STATUS_LABELS[this.payment()!.status] : '',
  );

  createPayment(): void {
    const o = this.order();
    if (!o) return;
    this.creatingPayment.set(true);
    const key = this.idempotency.getPaymentKey(o.id);
    const request: CreatePaymentRequest = {
      orderId: o.id,
      userId: this.authStore.userId()!,
      amountCents: o.totalCents,
      currency: 'BRL',
      method: 'card',
    };
    this.paymentsApi.create(request, key).subscribe({
      next: (payment) => {
        sessionStorage.setItem(`mm_payment_${o.id}`, payment.id);
        this.payment.set(payment);
        this.creatingPayment.set(false);
        this.toast.success('Pagamento criado. Verifique o status para atualizações.');
        this.syncOrderUntilSettled();
      },
      error: (err) => {
        this.creatingPayment.set(false);
        this.toast.error(err.message);
      },
    });
  }

  refreshPayment(): void {
    const p = this.payment();
    if (!p) return;
    this.paymentsApi.get(p.id).subscribe({
      next: (payment) => {
        this.payment.set(payment);
        this.reloadOrderSilently();
        this.syncOrderUntilSettled();
      },
      error: (err) => this.toast.error(err.message),
    });
  }

  cancelOrder(): void {
    const o = this.order();
    if (!o) return;
    this.cancelling.set(true);
    this.ordersApi.cancel(o.id).subscribe({
      next: (updatedOrder) => {
        this.order.set(updatedOrder);
        this.cancelling.set(false);
        this.confirmCancel.set(false);
        this.toast.success('Pedido cancelado');
      },
      error: (err) => {
        this.cancelling.set(false);
        this.confirmCancel.set(false);
        this.toast.error(err.message);
      },
    });
  }

  private checkAutoCancelExpiry(): void {
    const o = this.order();
    if (!o || o.status !== 'placed' || this.autoCancelNotified || this.autoCancelPending) {
      return;
    }
    if (this.remainingMs() > 0) {
      return;
    }
    this.autoCancelPending = true;
    this.reloadOrderSilently(() => {
      this.autoCancelPending = false;
      const status = this.order()?.status;
      if (status === 'cancelled' && !this.autoCancelNotified) {
        this.autoCancelNotified = true;
        this.toast.error('Pedido cancelado automaticamente por falta de pagamento.');
      }
    });
  }

  private reloadOrderSilently(onDone?: () => void): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      onDone?.();
      return;
    }
    this.ordersApi.get(id).subscribe({
      next: (order) => {
        this.order.set(order);
        if (order.status === 'paid' && !this.shipment()) {
          this.loadShipment(id);
        }
        onDone?.();
      },
      error: () => onDone?.(),
    });
  }

  private syncOrderUntilSettled(): void {
    if (this.order()?.status !== 'placed' || !this.payment()) {
      return;
    }
    this.stopOrderSync();
    this.autoRefreshActive.set(true);
    let attempts = 0;
    this.pollSubscription = interval(3000)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        attempts++;
        this.reloadOrderSilently(() => {
          const status = this.order()?.status;
          if (status && status !== 'placed') {
            this.stopOrderSync();
            this.notifyPaymentOutcome(status);
          } else if (attempts >= this.maxSyncAttempts) {
            this.stopOrderSync();
          }
        });
      });
  }

  private stopOrderSync(): void {
    this.autoRefreshActive.set(false);
    this.pollSubscription?.unsubscribe();
    this.pollSubscription = undefined;
  }

  private notifyPaymentOutcome(status: OrderStatus): void {
    if (status === 'paid') {
      this.toast.success('Pagamento confirmado.');
    } else if (status === 'payment_failed') {
      this.toast.error('Pagamento não aprovado.');
    } else if (status === 'payment_unknown') {
      this.toast.success('Pagamento aguardando confirmação.');
    }
  }

  formatDate(iso: string): string {
    return new Date(iso).toLocaleDateString('pt-BR', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  }

  formatDateTime(iso: string): string {
    return new Date(iso).toLocaleString('pt-BR', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  formatAddress(dest?: DestinationSnapshot): string {
    if (!dest) return 'Não informado';
    return `${dest.fullName}, ${dest.street}, ${dest.city} - ${dest.state}, ${dest.zip}, ${dest.country}`;
  }

  getShipmentStatusLabel(status: string): string {
    return SHIPMENT_STATUS_LABELS[status as keyof typeof SHIPMENT_STATUS_LABELS] || status;
  }

  getTrackingStatusLabel(status: string): string {
    const labels: Record<string, string> = {
      pending: 'Aguardando',
      processing: 'Processando',
      shipped: 'Enviado',
      delivered: 'Entregue',
      failed: 'Falhou',
    };
    return labels[status] || status;
  }

  isEventCompleted(status: string): boolean {
    return ['shipped', 'delivered'].includes(status);
  }
}
