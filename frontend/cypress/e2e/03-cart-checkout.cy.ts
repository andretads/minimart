import { ok } from '../support/api';
import * as factories from '../support/factories';

describe('cart and checkout', () => {
  beforeEach(() => {
    cy.loginAs();
    cy.intercept('GET', '/v1/cart', {
      body: ok(factories.cart({ items: [factories.cartItem()], estimated_total_cents: 1299 })),
    }).as('cart');
    cy.intercept('GET', '/v1/addresses', { body: ok([factories.address()]) }).as('addresses');
    cy.intercept('GET', '/v1/shipping/methods', { body: ok([factories.shippingMethod()]) }).as(
      'shippingMethods',
    );
    cy.intercept('POST', '/v1/shipping/quotes', { body: ok(factories.shippingQuote()) }).as(
      'shippingQuote',
    );
  });

  it('loads cart, updates quantity, and removes an item', () => {
    cy.intercept('PATCH', '/v1/cart/items/product-1', {
      body: ok(
        factories.cart({
          items: [factories.cartItem({ quantity: 2, line_total_cents: 2598 })],
          estimated_total_cents: 2598,
        }),
      ),
    }).as('updateItem');
    cy.intercept('DELETE', '/v1/cart/items/product-1', {
      body: ok(factories.cart({ items: [], estimated_total_cents: 0 })),
    }).as('removeItem');

    cy.visit('/cart');
    cy.wait(['@cart', '@addresses', '@shippingMethods', '@shippingQuote']);
    cy.contains('Café MiniMart').should('be.visible');

    cy.get('[data-cy="cart-quantity"]').first().find('button').contains('+').click();
    cy.wait('@updateItem').its('request.body').should('deep.include', { quantity: 2 });
    cy.get('[data-cy="cart-line-total"]')
      .first()
      .invoke('text')
      .should('match', /R\$\s*25,98/);

    cy.get('[data-cy="cart-remove"]').first().click();
    cy.wait('@removeItem');
    cy.contains('Seu carrinho está vazio').should('be.visible');
  });

  it('clears the cart', () => {
    cy.intercept('DELETE', '/v1/cart', { statusCode: 204, body: '' }).as('clearCart');

    cy.visit('/cart');
    cy.wait(['@cart', '@addresses', '@shippingMethods', '@shippingQuote']);
    cy.get('[data-cy="cart-clear"]').click();
    cy.wait('@clearCart');
    cy.contains('Seu carrinho está vazio').should('be.visible');
  });

  it('requires address and shipping before checkout', () => {
    cy.intercept('GET', '/v1/addresses', { body: ok([]) }).as('addresses');
    cy.intercept('POST', '/v1/cart/checkout', { body: ok(factories.checkoutResponse()) }).as(
      'checkout',
    );

    cy.visit('/cart');
    cy.wait(['@cart', '@addresses', '@shippingMethods']);
    cy.get('[data-cy="checkout-button"]').click();
    cy.contains('[data-cy="toast-container"]', 'Selecione um endereço e método de envio').should(
      'be.visible',
    );
    cy.get('@checkout.all').should('have.length', 0);
  });

  it('requests a shipping quote from the cart subtotal and shows free shipping', () => {
    cy.intercept('GET', '/v1/cart', {
      body: ok(
        factories.cart({
          items: [factories.cartItem({ line_total_cents: 6000 })],
          estimated_total_cents: 6000,
        }),
      ),
    }).as('cart');
    cy.intercept('POST', '/v1/shipping/quotes', {
      body: ok(factories.shippingQuote({ shipping_cost_cents: 0 })),
    }).as('shippingQuote');

    cy.visit('/cart');
    cy.wait(['@cart', '@addresses', '@shippingMethods']);
    cy.wait('@shippingQuote').its('request.body').should('deep.include', {
      method_id: 'method-1',
      address_id: 'address-1',
      subtotal_cents: 6000,
    });
    cy.contains('Frete grátis liberado').should('be.visible');
    cy.get('[data-cy="cart-total"]')
      .invoke('text')
      .should('match', /R\$\s*60,00/);
  });

  it('reuses the checkout idempotency key after a failed attempt', () => {
    let attempts = 0;
    cy.intercept('POST', '/v1/cart/checkout', (req) => {
      attempts += 1;
      if (attempts === 1) {
        req.reply({
          statusCode: 500,
          body: { success: false, error: { code: 'INTERNAL', message: 'Falha temporária' } },
        });
        return;
      }
      req.reply({ body: ok(factories.checkoutResponse()) });
    }).as('checkout');
    cy.intercept('GET', '/v1/orders/order-1', { body: ok(factories.order()) }).as('order');

    cy.visit('/cart');
    cy.wait(['@cart', '@addresses', '@shippingMethods', '@shippingQuote']);
    cy.get('[data-cy="checkout-button"]').click();
    cy.wait('@checkout').its('response.statusCode').should('eq', 500);
    cy.contains(
      '[data-cy="toast-container"]',
      'Não foi possível concluir a operação. Tente novamente.',
    ).should('be.visible');

    cy.get('[data-cy="checkout-button"]').click();
    cy.wait('@checkout').its('response.statusCode').should('eq', 200);
    cy.get('@checkout.all').then((requests) => {
      const keys = requests.map((request) => request.request.headers['idempotency-key']);
      expect(keys).to.have.length(2);
      expect(keys[0]).to.be.a('string').and.not.be.empty;
      expect(keys[1]).to.equal(keys[0]);
    });
    cy.url().should('include', '/orders/order-1');
  });

  it('checks out with the selected quote, address, and idempotency key', () => {
    cy.intercept('POST', '/v1/cart/checkout', {
      body: ok(factories.checkoutResponse()),
    }).as('checkout');
    cy.intercept('GET', '/v1/orders/order-1', { body: ok(factories.order()) }).as('order');

    cy.visit('/cart');
    cy.wait(['@cart', '@addresses', '@shippingMethods', '@shippingQuote']);
    cy.get('[data-cy="checkout-button"]').click();
    cy.wait('@checkout').then((interception) => {
      expect(interception.request.headers['idempotency-key']).to.exist;
      expect(interception.request.body).to.deep.include({
        shipping_quote_id: 'quote-1',
        address_id: 'address-1',
      });
    });
    cy.url().should('include', '/orders/order-1');
  });

  it('shows a checkout error and keeps the cart', () => {
    cy.intercept('POST', '/v1/cart/checkout', {
      statusCode: 409,
      body: { success: false, error: { code: 'CONFLICT', message: 'Estoque insuficiente' } },
    }).as('checkout');

    cy.visit('/cart');
    cy.wait(['@cart', '@addresses', '@shippingMethods', '@shippingQuote']);
    cy.get('[data-cy="checkout-button"]').click();
    cy.wait('@checkout');
    cy.contains('Estoque insuficiente').should('be.visible');
  });
});
