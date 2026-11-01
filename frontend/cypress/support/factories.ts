const now = '2026-01-01T00:00:00.000Z';

export function user(overrides: Record<string, unknown> = {}) {
  return {
    id: 'user-1',
    username: 'cypress',
    email: 'cypress@example.com',
    role: 'customer',
    access_token_expires_at: Date.now() + 3_600_000,
    expires_in: 3600,
    ...overrides,
  };
}

export function category(overrides: Record<string, unknown> = {}) {
  return {
    id: 'cat-1',
    name: 'Bebidas',
    icon: 'icon-cat-1',
    version: 1,
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

export function product(overrides: Record<string, unknown> = {}) {
  return {
    id: 'product-1',
    name: 'Café MiniMart',
    price_cents: 1299,
    category_id: 'cat-1',
    stock: 10,
    sku: 'SKU-1',
    description: 'Café torrado',
    total_rating: 4.5,
    total_reviews: 2,
    image_url: null,
    images: [],
    ...overrides,
  };
}

export function cart(overrides: Record<string, unknown> = {}) {
  return {
    id: 'cart-1',
    customer_id: 'customer-1',
    status: 'active',
    items: [],
    estimated_total_cents: 0,
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

export function cartItem(overrides: Record<string, unknown> = {}) {
  return {
    product_id: 'product-1',
    product_name_snapshot: 'Café MiniMart',
    unit_price_snapshot_cents: 1299,
    quantity: 1,
    line_total_cents: 1299,
    ...overrides,
  };
}

export function address(overrides: Record<string, unknown> = {}) {
  return {
    id: 'address-1',
    customer_id: 'customer-1',
    label: 'Casa',
    full_name: 'Cypress User',
    street: 'Rua Teste, 1',
    city: 'São Paulo',
    state: 'SP',
    zip: '01000-000',
    country: 'BR',
    phone: '11999999999',
    is_default: true,
    version: 1,
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

export function review(overrides: Record<string, unknown> = {}) {
  return {
    id: 'review-1',
    product_id: 'product-1',
    customer_id: 'customer-1',
    description: 'Ótimo produto, recomendo.',
    rating: 5,
    status: 'approved',
    created_at: now,
    ...overrides,
  };
}

export function shippingMethod(overrides: Record<string, unknown> = {}) {
  return {
    id: 'method-1',
    name: 'Padrão',
    description: 'Entrega padrão',
    active: true,
    estimated_min_days: 3,
    estimated_max_days: 5,
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

export function shippingQuote(overrides: Record<string, unknown> = {}) {
  return {
    id: 'quote-1',
    customer_id: 'customer-1',
    method_id: 'method-1',
    destination_snapshot: {
      label: 'Casa',
      full_name: 'Cypress User',
      street: 'Rua Teste, 1',
      city: 'São Paulo',
      state: 'SP',
      zip: '01000-000',
      country: 'BR',
      phone: '11999999999',
    },
    shipping_cost_cents: 500,
    subtotal_cents: 1299,
    free_threshold_cents: 5000,
    estimated_min_days: 3,
    estimated_max_days: 5,
    status: 'active',
    expires_at: '2026-01-02T00:00:00.000Z',
    created_at: now,
    ...overrides,
  };
}

export function shipment(overrides: Record<string, unknown> = {}) {
  return {
    id: 'shipment-1',
    order_id: 'order-1',
    user_id: 'user-1',
    method_id: 'method-1',
    tracking_code: 'TRACK-1',
    carrier: 'MiniMart Express',
    shipping_cost_cents: 500,
    destination_snapshot: {
      label: 'Casa',
      full_name: 'Cypress User',
      street: 'Rua Teste, 1',
      city: 'São Paulo',
      state: 'SP',
      zip: '01000-000',
      country: 'BR',
      phone: '11999999999',
    },
    status: 'shipped',
    shipped_at: now,
    delivered_at: null,
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

export function trackingEvent(overrides: Record<string, unknown> = {}) {
  return {
    id: 'event-1',
    shipment_id: 'shipment-1',
    status: 'shipped',
    description: 'Pacote enviado para entrega',
    location: 'São Paulo/SP',
    occurred_at: now,
    created_at: now,
    ...overrides,
  };
}

export function order(overrides: Record<string, unknown> = {}) {
  return {
    id: 'order-1',
    user_id: 'user-1',
    items: [
      {
        product_id: 'product-1',
        name: 'Café MiniMart',
        unit_price_cents: 1299,
        quantity: 1,
        line_total_cents: 1299,
      },
    ],
    total_cents: 1799,
    status: 'placed',
    created_at: now,
    shipping_quote_id: 'quote-1',
    shipping_method_id: 'method-1',
    shipping_cost_cents: 500,
    shipping_method_name: 'Padrão',
    auto_cancel_at: '2026-01-01T00:05:00.000Z',
    destination_snapshot: {
      label: 'Casa',
      full_name: 'Cypress User',
      street: 'Rua Teste, 1',
      city: 'São Paulo',
      state: 'SP',
      zip: '01000-000',
      country: 'BR',
      phone: '11999999999',
    },
    ...overrides,
  };
}

export function checkoutResponse(overrides: Record<string, unknown> = {}) {
  return {
    order_id: 'order-1',
    user_id: 'user-1',
    customer_id: 'customer-1',
    status: 'placed',
    total_cents: 1799,
    created_at: now,
    ...overrides,
  };
}

export function payment(overrides: Record<string, unknown> = {}) {
  return {
    id: 'payment-1',
    order_id: 'order-1',
    user_id: 'user-1',
    amount_cents: 1799,
    currency: 'BRL',
    method: 'card',
    status: 'pending',
    provider: 'fake',
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

export function wishlist(overrides: Record<string, unknown> = {}) {
  return {
    id: 'wishlist-1',
    customer_id: 'customer-1',
    items: [],
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

export function notification(overrides: Record<string, unknown> = {}) {
  return {
    id: 'notification-1',
    type: 'order_confirmation',
    user_id: 'user-1',
    order_id: 'order-1',
    subject: 'Pedido confirmado',
    body: 'Seu pedido foi confirmado.',
    sent_at: now,
    created_at: now,
    ...overrides,
  };
}

export function notificationPreferences(overrides: Record<string, unknown> = {}) {
  return {
    email_enabled: true,
    sms_enabled: false,
    push_enabled: false,
    marketing_enabled: false,
    ...overrides,
  };
}

export function avatar(overrides: Record<string, unknown> = {}) {
  return {
    account_id: 'user-1',
    image_url: null,
    content_type: 'image/png',
    size_bytes: 0,
    version: 1,
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}

export function customer(overrides: Record<string, unknown> = {}) {
  return {
    id: 'customer-1',
    user_id: 'user-1',
    display_name: 'Cypress User',
    status: 'active',
    version: 1,
    created_at: now,
    updated_at: now,
    ...overrides,
  };
}
