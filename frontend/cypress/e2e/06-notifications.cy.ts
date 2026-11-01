import { ok } from '../support/api';
import * as factories from '../support/factories';

describe('notifications', () => {
  beforeEach(() => {
    cy.loginAs();
    cy.intercept('GET', '/v1/notifications', {
      body: ok({
        data: [
          factories.notification(),
          factories.notification({
            id: 'notification-2',
            type: 'payment_succeeded',
            subject: 'Pagamento aprovado',
            body: 'Seu pagamento foi aprovado.',
            read_at: '2026-01-01T00:01:00.000Z',
          }),
        ],
        meta: { page: 1, page_size: 20, total: 2, total_pages: 1 },
      }),
    }).as('notifications');
  });

  it('lists notifications and filters by read state', () => {
    cy.visit('/notifications');
    cy.wait('@notifications');
    cy.get('[data-cy="notification-card"]').should('have.length', 2);
    cy.get('[data-cy="notification-card"]').first().should('contain.text', 'Pedido confirmado');
    cy.get('[data-cy="unread-dot"]').should('have.length', 1);

    cy.get('[data-cy="notification-filter-unread"]').click();
    cy.get('[data-cy="notification-card"]').should('have.length', 1);
    cy.get('[data-cy="notification-filter-read"]').click();
    cy.get('[data-cy="notification-card"]').should('have.length', 1);
    cy.get('[data-cy="notification-card"]').should('contain.text', 'Pagamento aprovado');
    cy.get('[data-cy="notification-filter-all"]').click();
    cy.get('[data-cy="notification-card"]').should('have.length', 2);
  });

  it('marks a single notification as read', () => {
    cy.intercept('POST', '/v1/notifications/notification-1/read', {
      statusCode: 204,
      body: '',
    }).as('markRead');

    cy.visit('/notifications');
    cy.wait('@notifications');
    cy.get('[data-cy="notification-card"]').first().find('[data-cy="mark-as-read"]').click();
    cy.wait('@markRead');
    cy.get('[data-cy="unread-dot"]').should('not.exist');
    cy.contains('[data-cy="toast-container"]', 'Marcada como lida').should('be.visible');
  });

  it('marks all notifications as read', () => {
    cy.intercept('POST', '/v1/notifications/read-all', {
      statusCode: 204,
      body: '',
    }).as('markAllRead');

    cy.visit('/notifications');
    cy.wait('@notifications');
    cy.get('[data-cy="mark-all-read"]').click();
    cy.wait('@markAllRead');
    cy.get('[data-cy="unread-dot"]').should('not.exist');
    cy.contains('[data-cy="toast-container"]', 'Todas as notificações marcadas como lidas').should(
      'be.visible',
    );
  });

  it('deletes a notification', () => {
    cy.intercept('DELETE', '/v1/notifications/notification-1', {
      statusCode: 204,
      body: '',
    }).as('deleteNotification');

    cy.visit('/notifications');
    cy.wait('@notifications');
    cy.get('[data-cy="notification-card"]').first().find('[data-cy="delete-notification"]').click();
    cy.wait('@deleteNotification');
    cy.get('[data-cy="notification-card"]').should('have.length', 1);
    cy.contains('[data-cy="toast-container"]', 'Notificação excluída').should('be.visible');
  });

  it('marks a notification as read when opening its order', () => {
    cy.intercept('POST', '/v1/notifications/notification-1/read', {
      statusCode: 204,
      body: '',
    }).as('markRead');
    cy.intercept('GET', '/v1/orders/order-1', { body: ok(factories.order()) }).as('order');
    cy.intercept('GET', '/v1/shipping/shipments*', { body: ok({ data: [] }) }).as('shipments');

    cy.visit('/notifications');
    cy.wait('@notifications');
    cy.get('[data-cy="notification-card"]')
      .first()
      .find('[data-cy="open-notification-order"]')
      .click();
    cy.wait('@markRead');
    cy.url().should('include', '/orders/order-1');
    cy.get('[data-cy="toast-container"] [data-cy="toast"]').should('not.exist');
  });
});
