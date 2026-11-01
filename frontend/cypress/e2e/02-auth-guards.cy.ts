import { ok } from '../support/api';
import * as factories from '../support/factories';

describe('auth and guards', () => {
  for (const username of ['unknown-user', 'cypress']) {
    it(`shows the same invalid-credentials error for ${username}`, () => {
      cy.intercept('POST', '/v1/auth/login', {
        statusCode: 401,
        body: {
          success: false,
          error: { code: 'INVALID_CREDENTIALS', message: 'Credenciais inválidas' },
        },
      }).as('login');

      cy.visit('/login');
      cy.get('input#username').type(username);
      cy.get('input#password').type('wrongpass');
      cy.contains('button', 'Entrar').click();
      cy.wait('@login');
      cy.contains('Credenciais inválidas').should('be.visible');
    });
  }

  it('logs in and honors returnUrl', () => {
    cy.intercept('POST', '/v1/auth/login', {
      body: ok(factories.user({ username: 'cypress' })),
    }).as('login');

    cy.visit('/login?returnUrl=/cart');
    cy.get('input#username').type('cypress');
    cy.get('input#password').type('password123');
    cy.contains('button', 'Entrar').click();
    cy.wait('@login');
    cy.url().should('include', '/cart');
    cy.get('[data-cy="account-button"]').should('contain.text', 'cypress');
  });

  it('registers a new account', () => {
    cy.intercept('POST', '/v1/auth/signup', {
      statusCode: 201,
      body: ok({
        id: 'new-user',
        username: 'newuser',
        email: 'new@example.com',
        created_at: '2026-01-01T00:00:00.000Z',
      }),
    }).as('signup');

    cy.visit('/register');
    cy.get('input#username').type('newuser');
    cy.get('input#email').type('new@example.com');
    cy.get('input#password').type('password123');
    cy.get('input#confirmPassword').type('password123');
    cy.get('input[type="checkbox"]').check();
    cy.contains('button', 'Criar conta').click();
    cy.wait('@signup');
    cy.contains('Conta criada com sucesso').should('be.visible');
  });

  it('shows an error for a duplicate username on signup', () => {
    cy.intercept('POST', '/v1/auth/signup', {
      statusCode: 409,
      body: {
        success: false,
        error: { code: 'USERNAME_TAKEN', message: 'Nome de usuário já está em uso' },
      },
    }).as('signup');

    cy.visit('/register');
    cy.get('input#username').type('existinguser');
    cy.get('input#email').type('existing@example.com');
    cy.get('input#password').type('password123');
    cy.get('input#confirmPassword').type('password123');
    cy.get('input[type="checkbox"]').check();
    cy.contains('button', 'Criar conta').click();
    cy.wait('@signup');
    cy.contains('Nome de usuário já está em uso').should('be.visible');
  });

  it('redirects guests away from protected routes', () => {
    cy.visit('/cart');
    cy.url().should('include', '/login');
    cy.url().should('include', 'returnUrl=%2Fcart');
  });

  it('redirects authenticated users away from guest routes', () => {
    cy.loginAs();
    cy.visit('/login');
    cy.url().should('not.include', '/login');
  });

  it('rehydrates the session after reload', () => {
    cy.loginAs();
    cy.visit('/');
    cy.reload();
    cy.contains('cypress').should('be.visible');
  });

  it('logs out', () => {
    cy.loginAs();
    cy.intercept('POST', '/v1/auth/logout', { statusCode: 204, body: '' }).as('logout');
    cy.visit('/');
    cy.get('[data-cy="account-button"]').then(($btn) => $btn[0].click());
    cy.get('[data-cy="account-button"]').should('have.attr', 'aria-expanded', 'true');
    cy.get('[data-cy="logout-button"]').then(($btn) => $btn[0].click());
    cy.wait('@logout');
    cy.get('[data-cy="nav-login"]').should('be.visible');
  });
});
