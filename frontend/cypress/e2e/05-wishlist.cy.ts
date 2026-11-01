import { ok } from '../support/api';
import * as factories from '../support/factories';

describe('wishlist', () => {
  beforeEach(() => {
    cy.loginAs();
    cy.intercept('GET', '/v1/wishlist', {
      body: ok(
        factories.wishlist({
          items: [{ product_id: 'product-1', added_at: '2026-01-01T00:00:00.000Z' }],
        }),
      ),
    }).as('wishlist');
    cy.intercept('GET', '/v1/showcase/products/product-1', { body: ok(factories.product()) }).as(
      'product',
    );
  });

  it('shows wishlist products', () => {
    cy.visit('/wishlist');
    cy.wait(['@wishlist', '@product']);
    cy.get('[data-cy="product-card"]').should('have.length', 1);
    cy.contains('Café MiniMart').should('be.visible');
  });

  it('clears the wishlist', () => {
    cy.intercept('DELETE', '/v1/wishlist', { statusCode: 204, body: '' }).as('clearWishlist');

    cy.visit('/wishlist');
    cy.wait(['@wishlist', '@product']);
    cy.get('[data-cy="wishlist-clear"]').click();
    cy.wait('@clearWishlist');
    cy.get('[data-cy="empty-state"]').should('be.visible');
  });

  it('adds a wishlisted product to the cart', () => {
    cy.intercept('POST', '/v1/cart/items', {
      body: ok(
        factories.cart({
          items: [factories.cartItem()],
          estimated_total_cents: 1299,
        }),
      ),
    }).as('addItem');

    cy.visit('/wishlist');
    cy.wait(['@wishlist', '@product']);
    cy.get('[data-cy="add-to-cart"]').first().click();
    cy.wait('@addItem').its('request.body').should('deep.include', {
      product_id: 'product-1',
      quantity: 1,
    });
    cy.contains('[data-cy="toast-container"]', 'Produto adicionado ao carrinho').should(
      'be.visible',
    );
  });

  it('removes an item from the product detail toggle', () => {
    cy.intercept('DELETE', '/v1/wishlist/items/product-1', {
      body: ok(factories.wishlist()),
    }).as('removeWishlist');

    cy.visit('/products/product-1');
    cy.wait(['@product', '@wishlist']);
    cy.get('[data-cy="detail-wishlist-toggle"]').click();
    cy.wait('@removeWishlist');
  });

  it('toggles wishlist from the catalog card', () => {
    cy.intercept('GET', '/v1/wishlist', { body: ok(factories.wishlist()) }).as('wishlist');
    cy.intercept('GET', '/v1/showcase/products*', {
      body: ok([factories.product()], { page_size: 20, next_cursor: null, has_more: false }),
    }).as('products');
    cy.intercept('POST', '/v1/wishlist/items', {
      body: ok(
        factories.wishlist({
          items: [{ product_id: 'product-1', added_at: '2026-01-01T00:00:00.000Z' }],
        }),
      ),
    }).as('addWishlist');

    cy.visit('/');
    cy.wait('@products');
    cy.get('[data-cy="nav-login"]').should('not.exist');
    cy.get('[data-cy="wishlist-toggle"]').first().click();
    cy.wait('@addWishlist');
  });

  it('removes a wishlisted product from the catalog card', () => {
    cy.intercept('GET', '/v1/showcase/products*', {
      body: ok([factories.product()], { page_size: 20, next_cursor: null, has_more: false }),
    }).as('products');
    cy.intercept('DELETE', '/v1/wishlist/items/product-1', {
      body: ok(factories.wishlist()),
    }).as('removeWishlist');

    cy.visit('/');
    cy.wait('@products');
    cy.get('[data-cy="wishlist-toggle"]').first().click();
    cy.wait('@removeWishlist');
  });
});
