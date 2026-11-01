import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { of, Subject, throwError } from 'rxjs';
import { vi } from 'vitest';
import { Cart } from '@shared/models/cart.models';
import { Address } from '@shared/models/address.models';
import { ShippingMethod, ShippingQuote } from '@shared/models/shipping.models';
import { CartComponent } from './cart.component';
import { CartService } from './cart.service';
import { CartStore } from './cart.store';
import { AuthStore } from '@core/auth/auth.store';
import { CustomerApiService } from '@features/customer/customer-api.service';
import { ShippingApiService } from '@features/shipping/shipping-api.service';
import { IdempotencyKeyService } from '@core/idempotency/idempotency-key.service';
import { ToastService } from '@core/toast/toast.service';
import { ApiError } from '@core/api/api-error';

const cart: Cart = {
  id: 'c1',
  customerId: 'cust-1',
  status: 'active',
  items: [
    {
      productId: 'p1',
      productNameSnapshot: 'Widget',
      unitPriceSnapshotCents: 500,
      quantity: 2,
      lineTotalCents: 1000,
    },
  ],
  estimatedTotalCents: 1000,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const checkoutResponse = {
  orderId: 'o1',
  userId: 'u1',
  customerId: 'cust-1',
  status: 'placed',
  totalCents: 1000,
  createdAt: '2026-01-01T00:00:00Z',
};

const activeQuote: ShippingQuote = {
  id: 'quote-1',
  customerId: 'cust-1',
  methodId: 'method-1',
  destinationSnapshot: {
    label: 'Casa',
    fullName: 'Ana Silva',
    street: 'Rua das Flores, 10',
    city: 'São Paulo',
    state: 'SP',
    zip: '01001000',
    country: 'BR',
  },
  shippingCostCents: 1500,
  subtotalCents: 1000,
  freeThresholdCents: 5000,
  estimatedMinDays: 2,
  estimatedMaxDays: 5,
  status: 'active',
  expiresAt: '2026-01-02T00:00:00Z',
  createdAt: '2026-01-01T00:00:00Z',
};

const defaultAddress: Address = {
  id: 'addr-1',
  customerId: 'cust-1',
  label: 'Casa',
  fullName: 'Ana Silva',
  street: 'Rua das Flores, 10',
  city: 'São Paulo',
  state: 'SP',
  zip: '01001000',
  country: 'BR',
  isDefault: true,
  version: 1,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const standardMethod: ShippingMethod = {
  id: 'method-1',
  name: 'Sedex',
  description: 'Entrega rápida',
  active: true,
  estimatedMinDays: 2,
  estimatedMaxDays: 5,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

describe('CartComponent', () => {
  let cartService: {
    getCart: ReturnType<typeof vi.fn>;
    updateItem: ReturnType<typeof vi.fn>;
    removeItem: ReturnType<typeof vi.fn>;
    clearCart: ReturnType<typeof vi.fn>;
    checkout: ReturnType<typeof vi.fn>;
  };
  let store: {
    setCart: ReturnType<typeof vi.fn>;
    setLoading: ReturnType<typeof vi.fn>;
    setError: ReturnType<typeof vi.fn>;
    clear: ReturnType<typeof vi.fn>;
    cart: ReturnType<typeof vi.fn>;
    loading: ReturnType<typeof vi.fn>;
    error: ReturnType<typeof vi.fn>;
    isEmpty: ReturnType<typeof vi.fn>;
    itemCount: ReturnType<typeof vi.fn>;
    estimatedTotal: ReturnType<typeof vi.fn>;
  };
  let authStore: { userId: ReturnType<typeof vi.fn>; isAuthenticated: ReturnType<typeof vi.fn> };
  let idempotency: {
    getCheckoutKey: ReturnType<typeof vi.fn>;
    clearCheckoutKey: ReturnType<typeof vi.fn>;
    newQuoteCycleKey: ReturnType<typeof vi.fn>;
  };
  let toast: { error: ReturnType<typeof vi.fn>; success: ReturnType<typeof vi.fn> };
  let router: { navigate: ReturnType<typeof vi.fn> };
  let customerApi: { getAddresses: ReturnType<typeof vi.fn> };
  let shippingApi: { getMethods: ReturnType<typeof vi.fn>; createQuote: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    cartService = {
      getCart: vi.fn(() => of(cart)),
      updateItem: vi.fn(() => of(cart)),
      removeItem: vi.fn(() => of(cart)),
      clearCart: vi.fn(() => of(void 0)),
      checkout: vi.fn(() => of(checkoutResponse)),
    };
    store = {
      setCart: vi.fn(),
      setLoading: vi.fn(),
      setError: vi.fn(),
      clear: vi.fn(),
      cart: vi.fn(() => cart),
      loading: vi.fn(() => false),
      error: vi.fn(() => null),
      isEmpty: vi.fn(() => false),
      itemCount: vi.fn(() => 2),
      estimatedTotal: vi.fn(() => 1000),
    };
    authStore = {
      userId: vi.fn(() => 'u1'),
      isAuthenticated: vi.fn(() => true),
    };
    idempotency = {
      getCheckoutKey: vi.fn(() => 'idem-key'),
      clearCheckoutKey: vi.fn(),
      newQuoteCycleKey: vi.fn(() => 'quote-cycle-key'),
    };
    toast = {
      error: vi.fn(),
      success: vi.fn(),
    };
    router = { navigate: vi.fn() };
    customerApi = { getAddresses: vi.fn(() => of([])) };
    shippingApi = { getMethods: vi.fn(() => of([])), createQuote: vi.fn(() => of(activeQuote)) };

    await TestBed.configureTestingModule({
      imports: [CartComponent],
      providers: [
        { provide: CartService, useValue: cartService },
        { provide: CartStore, useValue: store },
        { provide: AuthStore, useValue: authStore },
        { provide: CustomerApiService, useValue: customerApi },
        { provide: ShippingApiService, useValue: shippingApi },
        { provide: IdempotencyKeyService, useValue: idempotency },
        { provide: ToastService, useValue: toast },
        { provide: Router, useValue: router },
      ],
    }).compileComponents();
  });

  it('loads cart on init and clears load error on success', () => {
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    expect(cartService.getCart).toHaveBeenCalled();
    expect(store.setLoading).toHaveBeenCalledWith(true);
    expect(store.setError).toHaveBeenCalledWith(null);
    expect(store.setCart).toHaveBeenCalledWith(cart);
    expect(store.setLoading).toHaveBeenCalledWith(false);
    expect(store.setError).toHaveBeenCalledWith(null);
  });

  it('shows toast on checkout insufficient stock and does not set store error', () => {
    const apiError = new Error('order: estoque insuficiente') as ApiError;
    apiError.status = 409;
    cartService.checkout.mockReturnValue(throwError(() => apiError));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    fixture.componentInstance.selectedAddressId.set('addr-1');
    fixture.componentInstance.selectedQuote.set(activeQuote);
    fixture.componentInstance.checkout();

    expect(toast.error).toHaveBeenCalledWith(
      'Estoque insuficiente para finalizar a compra. Ajuste as quantidades e tente novamente.',
    );
    expect(store.setError).not.toHaveBeenCalledWith(expect.any(String));
    expect(store.error).not.toHaveBeenCalledWith(expect.stringContaining('insufficient'));
    expect(router.navigate).not.toHaveBeenCalled();
  });

  it('shows toast on checkout checkout in progress', () => {
    const apiError = new Error('cart: finalização em andamento') as ApiError;
    apiError.status = 409;
    cartService.checkout.mockReturnValue(throwError(() => apiError));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    fixture.componentInstance.selectedAddressId.set('addr-1');
    fixture.componentInstance.selectedQuote.set(activeQuote);
    fixture.componentInstance.checkout();

    expect(toast.error).toHaveBeenCalledWith(
      'Já existe um checkout em andamento. Aguarde alguns instantes e tente novamente.',
    );
  });

  it('shows toast when checkout has no address or shipping quote selected', () => {
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    fixture.componentInstance.checkout();

    expect(toast.error).toHaveBeenCalledWith('Selecione um endereço e método de envio');
    expect(cartService.checkout).not.toHaveBeenCalled();
  });

  it('checks out with the selected quote and address', () => {
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    fixture.componentInstance.selectedAddressId.set('addr-1');
    fixture.componentInstance.selectedQuote.set(activeQuote);
    fixture.componentInstance.checkout();

    expect(cartService.checkout).toHaveBeenCalledWith('idem-key', 'quote-1', 'addr-1');
    expect(idempotency.clearCheckoutKey).toHaveBeenCalledWith('u1', 'c1');
    expect(router.navigate).toHaveBeenCalledWith(['/orders', 'o1']);
  });

  it('sets store error on load failure and does not toast', () => {
    cartService.getCart.mockReturnValue(throwError(() => new Error('network error')));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    expect(store.setError).toHaveBeenCalledWith('network error');
    expect(toast.error).not.toHaveBeenCalled();
  });

  it('shows toast on updateQuantity failure', () => {
    const apiError = new Error('order: estoque insuficiente') as ApiError;
    apiError.status = 409;
    cartService.updateItem.mockReturnValue(throwError(() => apiError));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    fixture.componentInstance.updateQuantity('p1', 5);

    expect(toast.error).toHaveBeenCalledWith(
      'Estoque insuficiente para finalizar a compra. Ajuste as quantidades e tente novamente.',
    );
    expect(store.setError).not.toHaveBeenCalledWith(expect.any(String));
  });

  it('shows toast on removeItem failure', () => {
    const apiError = new Error('network error') as ApiError;
    apiError.status = 0;
    cartService.removeItem.mockReturnValue(throwError(() => apiError));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    fixture.componentInstance.removeItem('p1');

    expect(toast.error).toHaveBeenCalledWith(
      'Não foi possível concluir a operação. Tente novamente.',
    );
  });

  it('shows toast on clearCart failure', () => {
    const apiError = new Error('network error') as ApiError;
    apiError.status = 0;
    cartService.clearCart.mockReturnValue(throwError(() => apiError));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    fixture.componentInstance.clearCart();

    expect(toast.error).toHaveBeenCalledWith(
      'Não foi possível concluir a operação. Tente novamente.',
    );
    expect(store.setCart).not.toHaveBeenCalledWith(null);
  });

  it('quotes shipping for the default address and selects the first active quote', () => {
    customerApi.getAddresses.mockReturnValue(of([defaultAddress]));
    shippingApi.getMethods.mockReturnValue(of([standardMethod]));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.selectedAddressId()).toBe('addr-1');
    expect(shippingApi.createQuote).toHaveBeenCalledWith(
      'method-1',
      'addr-1',
      1000,
      'quote-cycle-key:method-1',
    );
    expect(fixture.componentInstance.selectedMethodId()).toBe('method-1');
    expect(fixture.componentInstance.selectedQuote()?.id).toBe('quote-1');
    expect(fixture.componentInstance.shippingCost()).toBe(activeQuote.shippingCostCents);
  });

  it('clears the stale quote when the delivery address changes', () => {
    customerApi.getAddresses.mockReturnValue(of([defaultAddress]));
    shippingApi.getMethods.mockReturnValue(of([standardMethod]));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    fixture.componentInstance.onAddressChange('addr-2');

    expect(fixture.componentInstance.selectedAddressId()).toBe('addr-2');
    expect(shippingApi.createQuote).toHaveBeenCalledWith(
      'method-1',
      'addr-2',
      1000,
      'quote-cycle-key:method-1',
    );
    expect(fixture.componentInstance.selectedMethodId()).toBe('method-1');
    expect(fixture.componentInstance.selectedQuote()?.id).toBe('quote-1');
  });

  it('quotes again when the cart total changes', () => {
    customerApi.getAddresses.mockReturnValue(of([defaultAddress]));
    shippingApi.getMethods.mockReturnValue(of([standardMethod]));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    expect(shippingApi.createQuote).toHaveBeenCalledTimes(1);

    store.estimatedTotal.mockReturnValue(2500);
    fixture.componentInstance.updateQuantity('p1', 5);

    expect(shippingApi.createQuote).toHaveBeenCalledTimes(2);
    expect(shippingApi.createQuote).toHaveBeenLastCalledWith(
      'method-1',
      'addr-1',
      2500,
      'quote-cycle-key:method-1',
    );
  });

  it('drives the free shipping bar from the threshold the quote carries', () => {
    customerApi.getAddresses.mockReturnValue(of([defaultAddress]));
    shippingApi.getMethods.mockReturnValue(of([standardMethod]));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.freeThresholdCents()).toBe(5000);
    expect(fixture.componentInstance.freeShippingProgress()).toBe(20);
  });

  it('completes the free shipping progress when the cart reaches the threshold', () => {
    customerApi.getAddresses.mockReturnValue(of([defaultAddress]));
    shippingApi.getMethods.mockReturnValue(of([standardMethod]));
    store.estimatedTotal.mockReturnValue(6000);

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.freeThresholdCents()).toBe(5000);
    expect(fixture.componentInstance.freeShippingProgress()).toBe(100);
  });

  it('reports a quote error when shipping quotes cannot be created', () => {
    const apiError = new Error('network error') as ApiError;
    apiError.status = 0;
    customerApi.getAddresses.mockReturnValue(of([defaultAddress]));
    shippingApi.getMethods.mockReturnValue(of([standardMethod]));
    shippingApi.createQuote.mockReturnValue(throwError(() => apiError));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.quoteError()).toBe(
      'Não foi possível concluir a operação. Tente novamente.',
    );
    expect(fixture.componentInstance.selectedQuote()).toBeNull();
    expect(fixture.componentInstance.loadingQuotes()).toBe(false);
  });

  it('does not fire a second quote while one is in flight', () => {
    const quoteSubject = new Subject<ShippingQuote>();
    customerApi.getAddresses.mockReturnValue(of([defaultAddress]));
    shippingApi.getMethods.mockReturnValue(of([standardMethod]));
    shippingApi.createQuote.mockReturnValue(quoteSubject.asObservable());

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    expect(shippingApi.createQuote).toHaveBeenCalledTimes(1);

    // A re-trigger with the same key must be ignored while in flight.
    fixture.componentInstance.refreshQuotes();
    expect(shippingApi.createQuote).toHaveBeenCalledTimes(1);

    quoteSubject.next(activeQuote);
    quoteSubject.complete();

    expect(fixture.componentInstance.selectedQuote()?.id).toBe('quote-1');
    expect(fixture.componentInstance.loadingQuotes()).toBe(false);
  });

  it('allows a retry after a failed quote round', () => {
    const apiError = new Error('network error') as ApiError;
    apiError.status = 0;
    customerApi.getAddresses.mockReturnValue(of([defaultAddress]));
    shippingApi.getMethods.mockReturnValue(of([standardMethod]));
    shippingApi.createQuote.mockReturnValueOnce(throwError(() => apiError));
    shippingApi.createQuote.mockReturnValueOnce(of(activeQuote));

    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();

    expect(shippingApi.createQuote).toHaveBeenCalledTimes(1);
    expect(fixture.componentInstance.quoteError()).toBe(
      'Não foi possível concluir a operação. Tente novamente.',
    );

    // The failed round cleared the in-flight guard, so a retry fires again.
    fixture.componentInstance.refreshQuotes();
    expect(shippingApi.createQuote).toHaveBeenCalledTimes(2);
  });

  it('continueShopping navigates to products', () => {
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();
    fixture.componentInstance.continueShopping();
    expect(router.navigate).toHaveBeenCalledWith(['/products']);
  });

  it('goToProfile navigates to profile', () => {
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();
    fixture.componentInstance.goToProfile();
    expect(router.navigate).toHaveBeenCalledWith(['/profile']);
  });

  it('applyPromo ignores blank codes', () => {
    vi.useFakeTimers();
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();
    fixture.componentInstance.promoCode.set('   ');
    fixture.componentInstance.applyPromo();
    vi.advanceTimersByTime(500);
    expect(fixture.componentInstance.promoApplied()).toBe(false);
    vi.useRealTimers();
  });

  it('applyPromo applies a known discount code', () => {
    vi.useFakeTimers();
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();
    fixture.componentInstance.promoCode.set('save10');
    fixture.componentInstance.applyPromo();
    expect(fixture.componentInstance.applyingPromo()).toBe(true);
    vi.advanceTimersByTime(500);
    expect(fixture.componentInstance.promoApplied()).toBe(true);
    expect(fixture.componentInstance.promoDiscount()).toBe(100);
    expect(fixture.componentInstance.applyingPromo()).toBe(false);
    vi.useRealTimers();
  });

  it('applyPromo leaves unknown codes unapplied', () => {
    vi.useFakeTimers();
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();
    fixture.componentInstance.promoCode.set('NOPE');
    fixture.componentInstance.applyPromo();
    vi.advanceTimersByTime(500);
    expect(fixture.componentInstance.promoApplied()).toBe(false);
    vi.useRealTimers();
  });

  it('totalWithShipping subtracts the applied discount', () => {
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();
    fixture.componentInstance.promoApplied.set(true);
    fixture.componentInstance.promoDiscount.set(200);
    fixture.componentInstance.selectedQuote.set(activeQuote);
    expect(fixture.componentInstance.totalWithShipping()).toBe(1000 - 200 + 1500);
  });

  it('totalWithShipping ignores discount when not applied', () => {
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();
    fixture.componentInstance.promoDiscount.set(200);
    fixture.componentInstance.selectedQuote.set(activeQuote);
    expect(fixture.componentInstance.totalWithShipping()).toBe(1000 + 1500);
  });

  it('checkout rejects an expired quote', () => {
    const fixture = TestBed.createComponent(CartComponent);
    fixture.detectChanges();
    fixture.componentInstance.selectedAddressId.set('addr-1');
    fixture.componentInstance.selectedQuote.set({ ...activeQuote, status: 'expired' });
    fixture.componentInstance.checkout();
    expect(toast.error).toHaveBeenCalledWith('Cotação de frete expirada ou inválida');
    expect(cartService.checkout).not.toHaveBeenCalled();
  });
});
