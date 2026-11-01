import { ok, fail } from './api';
import * as factories from './factories';

declare global {
  namespace Cypress {
    interface Chainable {
      mockApi(): Chainable<void>;
      loginAs(user?: Record<string, unknown>): Chainable<void>;
      visitAuthenticated(path: string): Chainable<Cypress.AUTWindow>;
      expectToast(type?: string): Chainable<void>;
      assertNoAuthHeader(alias: string): Chainable<void>;
    }
  }
}

Cypress.Commands.add('mockApi', () => {
  cy.intercept({ method: 'GET', pathname: '/v1/**' }, (req) => {
    req.reply({ statusCode: 500, body: fail('UNSTUBBED', `Unstubbed GET ${req.url}`) });
  }).as('unstubbedGet');
  cy.intercept({ method: 'POST', pathname: '/v1/**' }, (req) => {
    req.reply({ statusCode: 500, body: fail('UNSTUBBED', `Unstubbed POST ${req.url}`) });
  }).as('unstubbedPost');
  cy.intercept({ method: 'PATCH', pathname: '/v1/**' }, (req) => {
    req.reply({ statusCode: 500, body: fail('UNSTUBBED', `Unstubbed PATCH ${req.url}`) });
  }).as('unstubbedPatch');
  cy.intercept({ method: 'DELETE', pathname: '/v1/**' }, (req) => {
    req.reply({ statusCode: 500, body: fail('UNSTUBBED', `Unstubbed DELETE ${req.url}`) });
  }).as('unstubbedDelete');

  cy.intercept('GET', '/v1/auth/me', {
    statusCode: 401,
    body: fail('UNAUTHORIZED', 'Unauthorized'),
  }).as('authMe');
  cy.intercept('GET', '/v1/showcase/categories', { body: ok([factories.category()]) }).as(
    'categories',
  );
  cy.intercept('GET', '/v1/cart', { body: ok(factories.cart()) }).as('cart');
  cy.intercept('GET', '/v1/orders*', {
    body: ok([], { total: 0, page: 1, page_size: 20, total_pages: 0 }),
  }).as('orders');
  cy.intercept('GET', '/v1/avatars/*', { body: ok(factories.avatar()) }).as('avatar');
  cy.intercept('GET', '/v1/wishlist', { body: ok(factories.wishlist()) }).as('wishlist');
  cy.intercept('GET', '/v1/notifications*', { body: ok({ data: [] }) }).as('notifications');
  cy.intercept('GET', '/v1/notifications/preferences', {
    body: ok(factories.notificationPreferences()),
  }).as('notificationPreferences');
  cy.intercept('GET', '/v1/customers*', { body: ok(factories.customer()) }).as('customer');
  cy.intercept('GET', '/v1/addresses', { body: ok([]) }).as('addresses');
  cy.intercept('GET', '/v1/shipping/methods', { body: ok([]) }).as('shippingMethods');
  cy.intercept('GET', '/v1/shipping/shipments*', { body: ok({ data: [] }) }).as('shipments');
});

Cypress.Commands.add('loginAs', (userOverrides = {}) => {
  const u = factories.user(userOverrides);
  cy.intercept('GET', '/v1/auth/me', { body: ok(u) }).as('authMe');
  cy.window().then((win) => {
    win.sessionStorage.clear();
  });
});

Cypress.Commands.add('visitAuthenticated', (path: string) => {
  cy.visit(path);
});

Cypress.Commands.add('expectToast', (type = 'success') => {
  cy.get('[data-cy="toast-container"]').should('be.visible');
  cy.get(`.toast.${type}`).should('be.visible');
});

Cypress.Commands.add('assertNoAuthHeader', (alias: string) => {
  cy.get(alias).its('request.headers.authorization').should('be.undefined');
});
