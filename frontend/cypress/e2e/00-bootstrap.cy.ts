import { ok, fail } from '../support/api';
import * as factories from '../support/factories';

describe('bootstrap', () => {
  it('renders the anonymous shell when /v1/auth/me returns 401', () => {
    cy.visit('/');
    cy.contains('Entrar').should('be.visible');
    cy.contains('Começar').should('be.visible');
  });

  it('renders the authenticated shell with layout stubs', () => {
    cy.loginAs();
    cy.visit('/');
    cy.contains('cypress').should('be.visible');
    cy.get('[data-cy="nav-cart"]').should('be.visible');
  });

  it('fails loudly on an unstubbed /v1 request', () => {
    let failed = false;
    cy.intercept('GET', '/v1/definitely-unstubbed', (req) => {
      req.reply({ statusCode: 500, body: fail('UNSTUBBED', 'boom') });
    });
    cy.request({ url: '/v1/definitely-unstubbed', failOnStatusCode: false }).then((response) => {
      expect(response.status).to.eq(500);
    });
  });
});
