import { Component, inject, computed, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Observable, forkJoin, of } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import { Router } from '@angular/router';
import { Address } from '@shared/models/address.models';
import { CartService } from '@features/cart/cart.service';
import { CartStore } from '@features/cart/cart.store';
import { AuthStore } from '@core/auth/auth.store';
import { CustomerApiService } from '@features/customer/customer-api.service';
import { ShippingApiService } from '@features/shipping/shipping-api.service';
import { ShippingMethod, ShippingQuote } from '@shared/models/shipping.models';
import { MoneyPipe } from '@shared/pipes/money.pipe';
import { QuantityStepperComponent } from '@shared/components/quantity-stepper/quantity-stepper.component';
import { ButtonComponent } from '@shared/components/button/button.component';
import { EmptyStateComponent } from '@shared/components/empty-state/empty-state.component';
import { ErrorStateComponent } from '@shared/components/error-state/error-state.component';
import { PageShellComponent } from '@shared/components/page-shell/page-shell.component';
import { ProductImageComponent } from '@shared/components/product-image/product-image.component';
import { SkeletonComponent } from '@shared/components/skeleton/skeleton.component';
import { IdempotencyKeyService } from '@core/idempotency/idempotency-key.service';
import { ToastService } from '@core/toast/toast.service';
import { isApiError } from '@core/api/api-error';

interface QuoteResult {
  methodId: string;
  quote: ShippingQuote | null;
  error: string | null;
}

@Component({
  selector: 'app-cart',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MoneyPipe,
    QuantityStepperComponent,
    ButtonComponent,
    EmptyStateComponent,
    ErrorStateComponent,
    PageShellComponent,
    ProductImageComponent,
    SkeletonComponent,
  ],
  template: `
    <app-page-shell title="Carrinho de compras">
      <div class="cart-page">
        @if (store.loading()) {
          <div class="cart-skeleton" aria-busy="true">
            @for (i of [1, 2, 3]; track i) {
              <div class="cart-item-skeleton">
                <app-skeleton variant="rectangular" width="6rem" height="6rem" radius="1rem" />
                <div class="skeleton-lines">
                  <app-skeleton variant="text" width="60%" />
                  <app-skeleton variant="text" width="40%" />
                  <app-skeleton variant="text" width="30%" />
                </div>
              </div>
            }
          </div>
        } @else if (store.error()) {
          <app-error-state
            title="Falha ao carregar carrinho"
            [message]="store.error()!"
            retryLabel="Tentar novamente"
            (retry)="loadCart()"
          />
        } @else if (store.isEmpty()) {
          <app-empty-state
            title="Seu carrinho está vazio"
            message="Parece que você ainda não adicionou nada. Comece a comprar para preencher seu carrinho!"
            icon="cart"
            actionLabel="Continuar comprando"
            (action)="continueShopping()"
          />
        } @else {
          <div class="cart-head">
            <div>
              <span class="eyebrow">Sua sacola</span>
              <h2>{{ store.itemCount() }} {{ store.itemCount() === 1 ? 'item' : 'itens' }}</h2>
            </div>
            <button class="clear-link" type="button" (click)="clearCart()">
              <span class="icon icon-sm icon-trash" aria-hidden="true"></span>
              Limpar carrinho
            </button>
          </div>

          @if (freeThresholdCents() > 0) {
            <div class="shipping-bar">
              <span class="shipping-icon"
                ><span class="icon icon-truck" aria-hidden="true"></span
              ></span>
              <div class="shipping-body">
                @if (store.estimatedTotal() >= freeThresholdCents()) {
                  <span
                    ><strong>Frete grátis liberado.</strong> Seu pedido vai por nossa conta.</span
                  >
                } @else {
                  <span
                    >Adicione mais
                    <strong>{{ freeThresholdCents() - store.estimatedTotal() | money }}</strong>
                    para frete grátis.</span
                  >
                }
                <div class="shipping-track">
                  <span [style.width.%]="freeShippingProgress()"></span>
                </div>
              </div>
            </div>
          }

          <div class="cart-layout">
            <section class="cart-items" aria-label="Itens do carrinho">
              @for (item of store.cart()!.items; track item.productId) {
                <article class="cart-item">
                  <app-product-image
                    [productId]="item.productId"
                    [name]="item.productNameSnapshot"
                    class="item-image"
                  />
                  <div class="item-body">
                    <div class="item-top">
                      <h3>{{ item.productNameSnapshot }}</h3>
                      <button
                        class="remove-btn"
                        type="button"
                        (click)="removeItem(item.productId)"
                        [disabled]="updatingItem() === item.productId"
                        aria-label="Remove {{ item.productNameSnapshot }}"
                      >
                        <span class="icon icon-sm icon-x" aria-hidden="true"></span>
                      </button>
                    </div>
                    <div class="item-meta">
                      <span class="item-stock"
                        ><span class="stock-dot" aria-hidden="true"></span>Em estoque</span
                      >
                      <span class="item-unit">{{ item.unitPriceSnapshotCents | money }} cada</span>
                    </div>
                    <div class="item-actions">
                      <app-quantity-stepper
                        [value]="item.quantity"
                        [min]="1"
                        [max]="99"
                        [disabled]="updatingItem() === item.productId"
                        (valueChange)="updateQuantity(item.productId, $event)"
                      />
                      <span class="item-line-total">{{ item.lineTotalCents | money }}</span>
                    </div>
                  </div>
                </article>
              }

              <button class="continue-link" type="button" (click)="continueShopping()">
                <span class="icon icon-sm icon-arrow-right" aria-hidden="true"></span>
                Continuar comprando
              </button>
            </section>

            <aside class="cart-checkout">
              <div class="checkout-card">
                <h3>Resumo do pedido</h3>

                <div class="promo">
                  <label for="promo-code">Código promocional</label>
                  <div class="promo-row">
                    <input
                      id="promo-code"
                      type="text"
                      placeholder="Ex.: PROMO10"
                      [ngModel]="promoCode()"
                      (ngModelChange)="promoCode.set($event)"
                      (keyup.enter)="applyPromo()"
                    />
                    <app-button
                      label="Aplicar"
                      variant="outline"
                      size="sm"
                      [loading]="applyingPromo()"
                      [disabled]="!promoCode().trim()"
                      (clicked)="applyPromo()"
                    />
                  </div>
                  @if (promoApplied()) {
                    <p class="promo-success">
                      <span class="icon icon-sm icon-check-circle" aria-hidden="true"></span> Cupom
                      aplicado — {{ promoDiscount() | money }} de desconto
                    </p>
                  }
                </div>

                <div class="delivery">
                  <div class="delivery-field">
                    <label for="delivery-address">Endereço de entrega</label>
                    @if (loadingAddresses()) {
                      <p class="delivery-hint">Carregando endereços…</p>
                    } @else if (addresses().length === 0) {
                      <p class="delivery-hint">
                        Nenhum endereço cadastrado.
                        <button type="button" class="delivery-link" (click)="goToProfile()">
                          Cadastrar endereço
                        </button>
                      </p>
                    } @else {
                      <select
                        id="delivery-address"
                        [ngModel]="selectedAddressId()"
                        (ngModelChange)="onAddressChange($event)"
                      >
                        @for (address of addresses(); track address.id) {
                          <option [value]="address.id">
                            {{ address.label }} — {{ address.fullName }} · {{ address.city }}/{{
                              address.state
                            }}
                          </option>
                        }
                      </select>
                    }
                  </div>

                  <div class="delivery-field">
                    <label for="delivery-method">Método de envio</label>
                    @if (loadingQuotes()) {
                      <p class="delivery-hint">Calculando frete…</p>
                    } @else if (quoteError()) {
                      <p class="delivery-hint error">{{ quoteError() }}</p>
                    } @else if (shippingMethods().length === 0) {
                      <p class="delivery-hint">Nenhum método de envio disponível.</p>
                    } @else {
                      <select
                        id="delivery-method"
                        [ngModel]="selectedMethodId()"
                        (ngModelChange)="onMethodSelect($event)"
                      >
                        @for (method of shippingMethods(); track method.id) {
                          @if (shippingQuotes().get(method.id); as quote) {
                            <option [value]="method.id">
                              {{ method.name }} —
                              {{
                                quote.shippingCostCents === 0
                                  ? 'Grátis'
                                  : (quote.shippingCostCents | money)
                              }}
                              · {{ quote.estimatedMinDays }}–{{ quote.estimatedMaxDays }} dias
                            </option>
                          }
                        }
                      </select>
                    }
                  </div>
                </div>

                <div class="summary-rows">
                  <div class="summary-row">
                    <span>Subtotal</span><span>{{ store.estimatedTotal() | money }}</span>
                  </div>
                  @if (promoApplied()) {
                    <div class="summary-row discount">
                      <span>Desconto</span><span>−{{ promoDiscount() | money }}</span>
                    </div>
                  }
                  <div class="summary-row">
                    <span>Frete</span
                    ><span>{{ shippingCost() === 0 ? 'Grátis' : (shippingCost() | money) }}</span>
                  </div>
                  <div class="summary-row total">
                    <span>Total</span><span>{{ totalWithShipping() | money }}</span>
                  </div>
                </div>

                <app-button
                  label="Finalizar compra"
                  variant="primary"
                  size="lg"
                  [fullWidth]="true"
                  [loading]="checkoutLoading()"
                  (clicked)="checkout()"
                />
                <p class="secure-note">
                  <span class="icon icon-sm icon-lock" aria-hidden="true"></span> Compra segura. Não
                  armazenamos dados do cartão.
                </p>
                <div class="payment-methods">
                  <span>Aceitamos</span>
                  <div>
                    <span class="pay-badge">Visa</span>
                    <span class="pay-badge">MC</span>
                    <span class="pay-badge">Amex</span>
                    <span class="pay-badge">PayPal</span>
                  </div>
                </div>
              </div>
            </aside>
          </div>
        }
      </div>
    </app-page-shell>
  `,
  styles: [
    `
      .cart-page {
        max-width: 74rem;
        margin: 0 auto;
      }
      .cart-skeleton {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
      }
      .cart-item-skeleton {
        display: flex;
        gap: 1rem;
        padding: 1rem;
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        border-radius: 1rem;
      }
      .skeleton-lines {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
      }

      .cart-head {
        display: flex;
        align-items: flex-end;
        justify-content: space-between;
        gap: 1rem;
        margin-bottom: 1.25rem;
      }
      .cart-head h2 {
        margin: 0.15rem 0 0;
        font-size: clamp(1.5rem, 2.5vw, 2rem);
        font-weight: 750;
        letter-spacing: -0.025em;
      }
      .clear-link {
        display: inline-flex;
        align-items: center;
        gap: 0.4rem;
        padding: 0.4rem 0;
        border: 0;
        background: none;
        color: var(--color-text-muted);
        font: inherit;
        font-size: 0.8125rem;
        font-weight: 500;
        cursor: pointer;
        transition: color 0.2s;
      }
      .clear-link:hover {
        color: var(--color-danger);
      }

      .shipping-bar {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        margin-bottom: 1.5rem;
        padding: 0.9rem 1.1rem;
        border: 1px solid color-mix(in oklab, var(--accent) 22%, transparent);
        border-radius: 1rem;
        background: color-mix(in oklab, var(--accent) 6%, transparent);
      }
      .shipping-icon {
        display: grid;
        place-items: center;
        flex: 0 0 auto;
        width: 2.25rem;
        height: 2.25rem;
        border-radius: 0.75rem;
        background: color-mix(in oklab, var(--accent) 14%, transparent);
        color: var(--color-accent);
      }
      .shipping-body {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 0.45rem;
        font-size: 0.8125rem;
        color: var(--color-text-muted);
      }
      .shipping-body strong {
        color: var(--color-text);
        font-weight: 600;
      }
      .shipping-track {
        height: 0.35rem;
        overflow: hidden;
        border-radius: 999px;
        background: var(--color-border);
      }
      .shipping-track span {
        display: block;
        height: 100%;
        border-radius: inherit;
        background: var(--color-accent);
        transition: width 0.4s ease;
      }

      .cart-layout {
        display: grid;
        grid-template-columns: 1fr;
        gap: 2rem;
        align-items: start;
      }
      @media (min-width: 992px) {
        .cart-layout {
          grid-template-columns: minmax(0, 1fr) 23rem;
        }
      }

      .cart-items {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
      }
      .cart-item {
        display: grid;
        grid-template-columns: 6rem minmax(0, 1fr);
        gap: 1.1rem;
        padding: 1rem;
        border: 1px solid var(--color-border);
        border-radius: 1.1rem;
        background: var(--color-surface);
        transition:
          border-color 0.2s,
          box-shadow 0.2s;
      }
      .cart-item:hover {
        border-color: color-mix(in oklab, var(--accent) 30%, var(--color-border));
        box-shadow: var(--shadow-sm);
      }
      .item-image {
        width: 6rem;
        aspect-ratio: 1;
        border-radius: 0.9rem;
        overflow: hidden;
      }
      .item-body {
        min-width: 0;
      }
      .item-top {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 0.75rem;
      }
      .item-top h3 {
        margin: 0;
        font-size: 0.95rem;
        font-weight: 600;
        line-height: 1.4;
        letter-spacing: -0.01em;
      }
      .remove-btn {
        display: grid;
        place-items: center;
        flex: 0 0 auto;
        width: 1.75rem;
        height: 1.75rem;
        border: 0;
        border-radius: 999px;
        background: transparent;
        color: var(--color-text-subtle);
        cursor: pointer;
        transition:
          color 0.2s,
          background 0.2s;
      }
      .remove-btn:hover:not(:disabled) {
        color: var(--color-danger);
        background: var(--color-danger-bg);
      }
      .remove-btn:disabled {
        opacity: 0.45;
        cursor: not-allowed;
      }
      .item-meta {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 0.6rem;
        margin-top: 0.35rem;
      }
      .item-stock {
        display: inline-flex;
        align-items: center;
        gap: 0.3rem;
        font-size: 0.6875rem;
        font-weight: 600;
        color: var(--color-accent);
      }
      .stock-dot {
        width: 0.375rem;
        height: 0.375rem;
        border-radius: 999px;
        background: var(--color-accent);
      }
      .item-unit {
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }
      .item-actions {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 0.75rem;
        margin-top: 0.9rem;
      }
      .item-line-total {
        font-size: 1.05rem;
        font-weight: 700;
        letter-spacing: -0.02em;
      }
      .continue-link {
        display: inline-flex;
        align-items: center;
        align-self: flex-start;
        gap: 0.4rem;
        margin-top: 0.35rem;
        padding: 0.3rem 0;
        border: 0;
        background: none;
        color: var(--color-accent);
        font: inherit;
        font-size: 0.8125rem;
        font-weight: 600;
        cursor: pointer;
      }
      .continue-link .icon {
        transform: rotate(180deg);
      }

      .checkout-card {
        position: sticky;
        top: 6rem;
        padding: 1.5rem;
        border: 1px solid var(--color-border);
        border-radius: 1.25rem;
        background: var(--color-surface);
        box-shadow: var(--shadow-md);
      }
      .checkout-card h3 {
        margin: 0 0 1.25rem;
        font-size: 1.15rem;
        font-weight: 700;
        letter-spacing: -0.015em;
      }
      .promo label {
        display: block;
        margin-bottom: 0.4rem;
        font-size: 0.6875rem;
        font-weight: 600;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--color-text-muted);
      }
      .promo-row {
        display: flex;
        gap: 0.5rem;
      }
      .promo-row input {
        flex: 1;
        min-width: 0;
        padding: 0.6rem 0.8rem;
        border: 1px solid var(--color-border);
        border-radius: 0.75rem;
        background: var(--color-bg);
        color: var(--color-text);
        font: inherit;
        font-size: 0.8125rem;
      }
      .promo-row input:focus {
        outline: 2px solid color-mix(in oklab, var(--accent) 45%, transparent);
        outline-offset: 1px;
        border-color: var(--color-accent);
      }
      .promo-success {
        display: flex;
        align-items: center;
        gap: 0.35rem;
        margin: 0.5rem 0 0;
        font-size: 0.75rem;
        color: var(--color-accent);
      }
      .delivery {
        display: grid;
        gap: 0.8rem;
        margin-top: 1rem;
      }
      .delivery-field label {
        display: block;
        margin-bottom: 0.4rem;
        font-size: 0.6875rem;
        font-weight: 600;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--color-text-muted);
      }
      .delivery-field select {
        width: 100%;
        padding: 0.6rem 0.8rem;
        border: 1px solid var(--color-border);
        border-radius: 0.75rem;
        background: var(--color-bg);
        color: var(--color-text);
        font: inherit;
        font-size: 0.8125rem;
      }
      .delivery-field select:focus {
        outline: 2px solid color-mix(in oklab, var(--accent) 45%, transparent);
        outline-offset: 1px;
        border-color: var(--color-accent);
      }
      .delivery-hint {
        margin: 0;
        font-size: 0.75rem;
        color: var(--color-text-muted);
      }
      .delivery-hint.error {
        color: var(--color-danger);
      }
      .delivery-link {
        padding: 0;
        border: 0;
        background: none;
        color: var(--color-accent);
        font: inherit;
        font-size: inherit;
        font-weight: 600;
        cursor: pointer;
      }
      .summary-rows {
        margin: 1.25rem 0;
        padding: 1rem 0;
        border-top: 1px solid var(--color-border);
        border-bottom: 1px solid var(--color-border);
      }
      .summary-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0.4rem 0;
        font-size: 0.875rem;
        color: var(--color-text-muted);
      }
      .summary-row span:last-child {
        color: var(--color-text);
        font-weight: 500;
      }
      .summary-row.discount span:last-child {
        color: var(--color-accent);
      }
      .summary-row.total {
        margin-top: 0.5rem;
        padding-top: 0.9rem;
        border-top: 1px solid var(--color-border);
        font-size: 1rem;
        font-weight: 700;
        color: var(--color-text);
      }
      .summary-row.total span:last-child {
        font-size: 1.3rem;
        font-weight: 750;
        letter-spacing: -0.02em;
      }
      .secure-note {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.35rem;
        margin: 0.85rem 0 0;
        font-size: 0.6875rem;
        color: var(--color-text-muted);
      }
      .payment-methods {
        margin-top: 1.25rem;
        padding-top: 1.1rem;
        border-top: 1px solid var(--color-border);
        text-align: center;
      }
      .payment-methods > span {
        display: block;
        margin-bottom: 0.5rem;
        font-size: 0.6875rem;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--color-text-muted);
      }
      .payment-methods > div {
        display: flex;
        flex-wrap: wrap;
        justify-content: center;
        gap: 0.35rem;
      }

      @media (max-width: 991px) {
        .checkout-card {
          position: static;
        }
      }
      @media (max-width: 560px) {
        .cart-item {
          grid-template-columns: 4.75rem minmax(0, 1fr);
          padding: 0.85rem;
        }
        .item-image {
          width: 4.75rem;
        }
        .item-actions {
          flex-wrap: wrap;
        }
        .checkout-card {
          padding: 1.15rem;
        }
        .cart-head {
          align-items: center;
        }
      }
    `,
  ],
})
export class CartComponent implements OnInit {
  private readonly cartService = inject(CartService);
  private readonly customerApi = inject(CustomerApiService);
  private readonly shippingApi = inject(ShippingApiService);
  protected readonly store = inject(CartStore);
  private readonly authStore = inject(AuthStore);
  private readonly idempotency = inject(IdempotencyKeyService);
  private readonly toast = inject(ToastService);

  updatingItem = signal<string | null>(null);
  checkoutLoading = signal(false);
  promoCode = signal('');
  promoApplied = signal(false);
  promoDiscount = signal(0);
  applyingPromo = signal(false);

  // Shipping integration
  addresses = signal<Address[]>([]);
  selectedAddressId = signal<string | null>(null);
  shippingMethods = signal<ShippingMethod[]>([]);
  shippingQuotes = signal<Map<string, ShippingQuote>>(new Map());
  selectedMethodId = signal<string | null>(null);
  selectedQuote = signal<ShippingQuote | null>(null);
  loadingAddresses = signal(false);
  loadingQuotes = signal(false);
  quoteError = signal<string | null>(null);

  freeThresholdCents = computed(() => {
    const selected = this.selectedQuote()?.freeThresholdCents ?? 0;
    if (selected > 0) {
      return selected;
    }
    const thresholds = Array.from(this.shippingQuotes().values())
      .map((quote) => quote.freeThresholdCents)
      .filter((threshold) => threshold > 0);
    return thresholds.length > 0 ? Math.min(...thresholds) : 0;
  });

  freeShippingProgress = computed(() => {
    const threshold = this.freeThresholdCents();
    if (threshold <= 0) {
      return 0;
    }
    return Math.min((this.store.estimatedTotal() / threshold) * 100, 100);
  });

  private quoteRequestId = 0;
  private quotedKey = '';
  private quoteCycleKey = '';
  private quoteCycleFor = '';
  private quoteInFlightFor = '';

  ngOnInit(): void {
    this.loadCart();
    this.loadAddresses();
    this.loadShippingMethods();
  }

  loadCart(): void {
    this.store.setLoading(true);
    this.store.setError(null);
    this.cartService.getCart().subscribe({
      next: (cart) => {
        this.store.setCart(cart);
        this.store.setLoading(false);
        this.store.setError(null);
        this.refreshQuotes();
      },
      error: (err) => {
        this.store.setError(err.message);
        this.store.setLoading(false);
      },
    });
  }

  loadAddresses(): void {
    this.loadingAddresses.set(true);
    this.customerApi.getAddresses().subscribe({
      next: (addresses) => {
        this.addresses.set(addresses);
        // Auto-select default address
        const defaultAddr = addresses.find((a) => a.isDefault);
        if (defaultAddr) {
          this.selectedAddressId.set(defaultAddr.id);
        }
        this.loadingAddresses.set(false);
        this.refreshQuotes();
      },
      error: () => {
        this.loadingAddresses.set(false);
        this.toast.error('Falha ao carregar endereços');
      },
    });
  }

  loadShippingMethods(): void {
    this.shippingApi.getMethods().subscribe({
      next: (methods) => {
        this.shippingMethods.set(methods);
        this.refreshQuotes();
      },
      error: () => {
        this.toast.error('Falha ao carregar métodos de envio');
      },
    });
  }

  onAddressChange(addressId: string): void {
    this.selectedAddressId.set(addressId);
    this.refreshQuotes();
  }

  refreshQuotes(): void {
    const addressId = this.selectedAddressId();
    const activeMethods = this.shippingMethods().filter((method) => method.active);
    if (!addressId || activeMethods.length === 0 || !this.store.cart() || this.store.isEmpty()) {
      this.shippingQuotes.set(new Map());
      this.selectedMethodId.set(null);
      this.selectedQuote.set(null);
      this.quotedKey = '';
      this.quoteCycleFor = '';
      this.quoteCycleKey = '';
      this.quoteInFlightFor = '';
      this.loadingQuotes.set(false);
      return;
    }

    const subtotalCents = this.store.estimatedTotal();
    const quoteKey = `${addressId}:${subtotalCents}:${activeMethods.map((method) => method.id).join(',')}`;
    if (quoteKey === this.quotedKey || quoteKey === this.quoteInFlightFor) {
      return;
    }

    // A round of quotes shares one cycle key, and each quote of the round
    // derives its own key from it. A retry of the round therefore replays the
    // same quotes instead of pricing them again.
    if (this.quoteCycleFor !== quoteKey) {
      this.quoteCycleFor = quoteKey;
      this.quoteCycleKey = this.idempotency.newQuoteCycleKey();
    }

    this.loadingQuotes.set(true);
    this.quoteError.set(null);
    const requestId = ++this.quoteRequestId;
    const selectedMethodId = this.selectedMethodId();
    this.quoteInFlightFor = quoteKey;

    forkJoin(
      activeMethods.map((method) =>
        this.shippingApi
          .createQuote(method.id, addressId, subtotalCents, `${this.quoteCycleKey}:${method.id}`)
          .pipe(
            map((quote): QuoteResult => ({ methodId: method.id, quote, error: null })),
            catchError((err): Observable<QuoteResult> =>
              of({ methodId: method.id, quote: null, error: this.errorMessage(err) }),
            ),
          ),
      ),
    ).subscribe({
      next: (results) => {
        if (requestId !== this.quoteRequestId) {
          return;
        }
        this.quoteInFlightFor = '';
        const quotesMap = new Map<string, ShippingQuote>();
        let firstError: string | null = null;
        for (const result of results) {
          if (result.quote) {
            quotesMap.set(result.methodId, result.quote);
          }
          if (result.error && !firstError) {
            firstError = result.error;
          }
        }
        this.shippingQuotes.set(quotesMap);
        this.quoteError.set(firstError);
        const selected = selectedMethodId
          ? results.find((result) => result.methodId === selectedMethodId && result.quote)
          : undefined;
        const chosen = selected ?? results.find((result) => result.quote?.status === 'active');
        this.selectedMethodId.set(chosen?.methodId ?? null);
        this.selectedQuote.set(chosen?.quote ?? null);
        this.quotedKey = firstError ? '' : quoteKey;
        this.loadingQuotes.set(false);
      },
      error: (err) => {
        if (requestId !== this.quoteRequestId) {
          return;
        }
        this.quoteInFlightFor = '';
        this.quoteError.set(this.errorMessage(err));
        this.loadingQuotes.set(false);
      },
    });
  }

  onMethodSelect(methodId: string): void {
    this.selectedMethodId.set(methodId);
    const quote = this.shippingQuotes().get(methodId);
    this.selectedQuote.set(quote ?? null);
  }

  private errorMessage(err: unknown): string {
    const msg = isApiError(err)
      ? err.message
      : err instanceof Error
        ? err.message
        : String(err ?? '');
    if (isApiError(err)) {
      const lower = msg.toLowerCase();
      if (err.status === 0 || err.status >= 500) {
        return 'Não foi possível concluir a operação. Tente novamente.';
      }
      if (err.status === 409) {
        if (lower.includes('estoque insuficiente')) {
          return 'Estoque insuficiente para finalizar a compra. Ajuste as quantidades e tente novamente.';
        }
        if (lower.includes('finalização em andamento') || lower.includes('checkout in progress')) {
          return 'Já existe um checkout em andamento. Aguarde alguns instantes e tente novamente.';
        }
        if (
          lower.includes('optimistic') ||
          lower.includes('estado alterado') ||
          lower.includes('state changed') ||
          lower.includes('concurrent')
        ) {
          return 'O carrinho foi alterado em outra sessão. Recarregue a página e tente novamente.';
        }
      }
      if (err.status === 404) {
        if (lower.includes('carrinho') || lower.includes('cart')) {
          return 'Carrinho não encontrado.';
        }
        if (lower.includes('produto') || lower.includes('product')) {
          return 'Um dos produtos não está mais disponível. Atualize o carrinho.';
        }
      }
      if (
        err.status === 400 &&
        (lower.includes('carrinho vazio') || lower.includes('cart empty'))
      ) {
        return 'Seu carrinho está vazio.';
      }
      if (
        err.status === 400 &&
        (lower.includes('cotação de frete') || lower.includes('shipping quote'))
      ) {
        return 'Cotação de frete inválida ou expirada.';
      }
      if (err.status === 400 && (lower.includes('endereço') || lower.includes('address'))) {
        return 'Endereço inválido ou não encontrado.';
      }
    }
    return msg || 'Não foi possível concluir a operação. Tente novamente.';
  }

  updateQuantity(productId: string, quantity: number): void {
    this.updatingItem.set(productId);
    this.cartService.updateItem(productId, quantity).subscribe({
      next: (cart) => {
        this.store.setCart(cart);
        this.updatingItem.set(null);
        this.refreshQuotes();
      },
      error: (err) => {
        this.toast.error(this.errorMessage(err));
        this.updatingItem.set(null);
      },
    });
  }

  removeItem(productId: string): void {
    this.updatingItem.set(productId);
    this.cartService.removeItem(productId).subscribe({
      next: (cart) => {
        this.store.setCart(cart);
        this.updatingItem.set(null);
        this.refreshQuotes();
      },
      error: (err) => {
        this.toast.error(this.errorMessage(err));
        this.updatingItem.set(null);
      },
    });
  }

  clearCart(): void {
    this.cartService.clearCart().subscribe({
      next: () => {
        this.store.clear();
        this.refreshQuotes();
      },
      error: (err) => this.toast.error(this.errorMessage(err)),
    });
  }

  applyPromo(): void {
    const code = this.promoCode().trim().toUpperCase();
    if (!code) return;
    this.applyingPromo.set(true);
    // Simulate promo check
    setTimeout(() => {
      if (code === 'SAVE10' || code === 'WELCOME') {
        this.promoApplied.set(true);
        this.promoDiscount.set(Math.round(this.store.estimatedTotal() * 0.1));
      }
      this.applyingPromo.set(false);
    }, 500);
  }

  shippingCost(): number {
    return this.selectedQuote()?.shippingCostCents ?? 0;
  }

  totalWithShipping(): number {
    const subtotal = this.store.estimatedTotal();
    const discount = this.promoApplied() ? this.promoDiscount() : 0;
    return subtotal - discount + this.shippingCost();
  }

  checkout(): void {
    const cart = this.store.cart();
    const quote = this.selectedQuote();
    const addressId = this.selectedAddressId();

    if (!cart || !quote || !addressId) {
      this.toast.error('Selecione um endereço e método de envio');
      return;
    }
    if (quote.status !== 'active') {
      this.toast.error('Cotação de frete expirada ou inválida');
      return;
    }

    this.checkoutLoading.set(true);
    const key = this.idempotency.getCheckoutKey(this.authStore.userId()!, cart.id);
    this.cartService.checkout(key, quote.id, addressId).subscribe({
      next: (response) => {
        this.idempotency.clearCheckoutKey(this.authStore.userId()!, cart.id);
        this.store.clear();
        this.refreshQuotes();
        this.checkoutLoading.set(false);
        this.router.navigate(['/orders', response.orderId]);
      },
      error: (err) => {
        this.checkoutLoading.set(false);
        this.toast.error(this.errorMessage(err));
      },
    });
  }

  continueShopping(): void {
    this.router.navigate(['/products']);
  }

  goToProfile(): void {
    this.router.navigate(['/profile']);
  }

  private router = inject(Router);
}
