import { ok } from '../support/api';
import * as factories from '../support/factories';

describe('orders and payments', () => {
  beforeEach(() => {
    cy.loginAs();
  });

  it('lists newest orders first and opens details', () => {
    cy.intercept('GET', '/v1/orders*', {
      body: ok(
        [
          factories.order({ id: 'order-1', created_at: '2026-01-02T00:00:00.000Z' }),
          factories.order({ id: 'order-2', created_at: '2026-01-01T00:00:00.000Z' }),
        ],
        { page: 1, page_size: 5, total: 2, total_pages: 1 },
      ),
    }).as('orders');

    cy.visit('/orders');
    cy.wait('@orders');
    cy.get('[data-cy="order-card"]').should('have.length', 2);
    cy.get('[data-cy="order-card"]').first().should('contain.text', 'ORDER-1');
    cy.get('[data-cy="order-card"]').last().should('contain.text', 'ORDER-2');
    cy.get('[data-cy="order-details-link"]').first().click();
    cy.url().should('include', '/orders/order-1');
  });

  it('loads the second page of orders', () => {
    cy.intercept('GET', '/v1/orders*', (req) => {
      const pageTwo = String(req.query.page) === '2';
      req.alias = pageTwo ? 'ordersPage2' : 'ordersPage1';
      req.reply({
        body: ok(
          [
            factories.order({
              id: pageTwo ? 'order-6' : 'order-1',
              created_at: pageTwo ? '2025-12-31T00:00:00.000Z' : '2026-01-02T00:00:00.000Z',
            }),
          ],
          { page: pageTwo ? 2 : 1, page_size: 5, total: 6, total_pages: 2 },
        ),
      });
    });

    cy.visit('/orders');
    cy.wait('@ordersPage1');
    cy.get('[data-cy="order-card"]').should('contain.text', 'ORDER-1');
    cy.contains('button', 'Próxima').click();
    cy.wait('@ordersPage2');
    cy.get('[data-cy="order-card"]').should('contain.text', 'ORDER-6');
  });

  it('opens order details and cancels', () => {
    cy.intercept('GET', '/v1/orders/order-1', { body: ok(factories.order()) }).as('order');
    cy.intercept('GET', '/v1/shipping/shipments*', { body: ok({ data: [] }) }).as('shipments');
    cy.intercept('POST', '/v1/orders/order-1/cancel', {
      body: ok(factories.order({ status: 'cancelled' })),
    }).as('cancel');

    cy.visit('/orders/order-1');
    cy.wait(['@order', '@shipments']);
    cy.get('[data-cy="cancel-order"]').click();
    cy.get('[data-cy="confirm-dialog"]').should('be.visible');
    cy.get('[data-cy="confirm-button"]').click();
    cy.wait('@cancel');
    cy.contains('Cancelado').should('be.visible');
  });

  it('keeps the order when the cancel dialog is dismissed', () => {
    cy.intercept('GET', '/v1/orders/order-1', { body: ok(factories.order()) }).as('order');
    cy.intercept('GET', '/v1/shipping/shipments*', { body: ok({ data: [] }) }).as('shipments');
    cy.intercept('POST', '/v1/orders/order-1/cancel', {
      body: ok(factories.order({ status: 'cancelled' })),
    }).as('cancel');

    cy.visit('/orders/order-1');
    cy.wait(['@order', '@shipments']);
    cy.get('[data-cy="cancel-order"]').click();
    cy.get('[data-cy="cancel-confirmation"]').click();
    cy.get('[data-cy="confirm-dialog"]').should('not.exist');
    cy.get('[data-cy="order-status"]').should('contain.text', 'Aguardando pagamento');
    cy.get('@cancel.all').should('have.length', 0);
  });

  it('creates a payment and stores its id', () => {
    cy.intercept('GET', '/v1/orders/order-1', { body: ok(factories.order()) }).as('order');
    cy.intercept('GET', '/v1/shipping/shipments*', { body: ok({ data: [] }) }).as('shipments');
    cy.intercept('POST', '/v1/payments', {
      body: ok(factories.payment()),
    }).as('payment');

    cy.visit('/orders/order-1');
    cy.wait(['@order', '@shipments']);
    cy.get('[data-cy="pay-now"]').click();
    cy.wait('@payment').its('request.headers.idempotency-key').should('exist');
    cy.assertNoAuthHeader('@payment');
    cy.get('@payment').its('request.body').should('deep.include', {
      order_id: 'order-1',
      user_id: 'user-1',
      amount_cents: 1799,
      currency: 'BRL',
      method: 'card',
    });
    cy.contains('Pagamento criado').should('be.visible');
  });

  it('reuses the payment idempotency key after a failed attempt', () => {
    let attempts = 0;
    cy.intercept('GET', '/v1/orders/order-1', { body: ok(factories.order()) }).as('order');
    cy.intercept('GET', '/v1/shipping/shipments*', { body: ok({ data: [] }) }).as('shipments');
    cy.intercept('POST', '/v1/payments', (req) => {
      attempts += 1;
      if (attempts === 1) {
        req.reply({
          statusCode: 500,
          body: { success: false, error: { code: 'INTERNAL', message: 'Falha temporária' } },
        });
        return;
      }
      req.reply({ body: ok(factories.payment({ status: 'succeeded' })) });
    }).as('payment');

    cy.visit('/orders/order-1');
    cy.wait(['@order', '@shipments']);
    cy.get('[data-cy="pay-now"]').click();
    cy.wait('@payment').its('response.statusCode').should('eq', 500);
    cy.get('[data-cy="pay-now"]').click();
    cy.wait('@payment').its('response.statusCode').should('eq', 200);
    cy.get('@payment.all').then((requests) => {
      const keys = requests.map((request) => request.request.headers['idempotency-key']);
      expect(keys).to.have.length(2);
      expect(keys[0]).to.be.a('string').and.not.be.empty;
      expect(keys[1]).to.equal(keys[0]);
    });
  });

  it('polls a placed order until payment succeeds', () => {
    let orderLoads = 0;
    cy.intercept('GET', '/v1/orders/order-1', (req) => {
      orderLoads += 1;
      req.reply({
        body: ok(factories.order({ status: orderLoads === 1 ? 'placed' : 'paid' })),
      });
    }).as('order');
    cy.intercept('GET', '/v1/shipping/shipments*', { body: ok({ data: [] }) }).as('shipments');
    cy.intercept('GET', '/v1/payments/payment-1', {
      body: ok(factories.payment({ status: 'pending' })),
    }).as('payment');

    cy.visit('/orders/order-1', {
      onBeforeLoad(win) {
        win.sessionStorage.setItem('mm_payment_id_order-1', 'payment-1');
      },
    });
    cy.wait(['@order', '@shipments', '@payment']);
    cy.get('[data-cy="order-status"]').should('contain.text', 'Aguardando pagamento');
    cy.wait('@order');
    cy.get('[data-cy="order-status"]').should('contain.text', 'Pagamento confirmado');
    cy.contains('[data-cy="toast-container"]', 'Pagamento confirmado.').should('be.visible');
  });

  it('shows the automatic cancellation outcome after the deadline', () => {
    let orderLoads = 0;
    cy.intercept('GET', '/v1/orders/order-1', (req) => {
      orderLoads += 1;
      req.reply({
        body: ok(factories.order({ status: orderLoads === 1 ? 'placed' : 'cancelled' })),
      });
    }).as('order');
    cy.intercept('GET', '/v1/shipping/shipments*', { body: ok({ data: [] }) }).as('shipments');

    cy.visit('/orders/order-1');
    cy.wait(['@order', '@shipments']);
    cy.wait('@order');
    cy.get('[data-cy="order-status"]').should('contain.text', 'Cancelado');
    cy.contains(
      '[data-cy="toast-container"]',
      'Pedido cancelado automaticamente por falta de pagamento.',
    ).should('be.visible');
  });

  it('shows shipment tracking for a paid order', () => {
    cy.intercept('GET', '/v1/orders/order-1', {
      body: ok(factories.order({ status: 'paid' })),
    }).as('order');
    cy.intercept('GET', '/v1/shipping/shipments*', {
      body: ok({ data: [factories.shipment()] }),
    }).as('shipments');
    cy.intercept('GET', '/v1/shipping/shipments/shipment-1/events', {
      body: ok([
        factories.trackingEvent(),
        factories.trackingEvent({
          id: 'event-2',
          status: 'out_for_delivery',
          description: 'Saiu para entrega',
          location: 'São Paulo/SP',
        }),
      ]),
    }).as('trackingEvents');

    cy.visit('/orders/order-1');
    cy.wait(['@order', '@shipments', '@trackingEvents']);
    cy.get('[data-cy="tracking-events"] [data-cy="tracking-event"]').should('have.length', 2);
    cy.contains('TRACK-1').should('be.visible');
    cy.contains('MiniMart Express').should('be.visible');
    cy.contains('Saiu para entrega').should('be.visible');
  });

  it('shows payment status and refreshes', () => {
    let loads = 0;
    cy.intercept('GET', '/v1/payments/payment-1', (req) => {
      loads += 1;
      req.reply({
        body: ok(factories.payment({ status: loads === 1 ? 'pending' : 'succeeded' })),
      });
    }).as('payment');

    cy.visit('/payments/payment-1');
    cy.wait('@payment');
    cy.get('[data-cy="payment-status"]').should('contain.text', 'Pendente');
    cy.get('[data-cy="refresh-payment"]').click();
    cy.wait('@payment');
    cy.get('[data-cy="payment-status"]').should('contain.text', 'Aprovado');
    cy.assertNoAuthHeader('@payment');
  });
});
