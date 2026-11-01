import { ok } from '../support/api';
import * as factories from '../support/factories';

describe('catalog', () => {
  it('renders products and pt-BR money', () => {
    cy.intercept('GET', '/v1/showcase/products*', {
      body: ok([factories.product()], { page_size: 20, next_cursor: null, has_more: false }),
    }).as('products');

    cy.visit('/');
    cy.wait('@products');
    cy.get('[data-cy="product-card"]').should('have.length', 1);
    cy.get('[data-cy="product-name"]').should('contain.text', 'Café MiniMart');
    cy.contains('R$ 12,99').should('be.visible');
  });

  it('filters by category query', () => {
    cy.intercept('GET', '/v1/showcase/products*', (req) => {
      req.reply(ok([factories.product()], { page_size: 20, next_cursor: null, has_more: false }));
    }).as('products');

    cy.visit('/products?category_id=cat-1');
    cy.wait('@products');
    cy.get('@products').its('request.url').should('include', 'category_id=cat-1');
  });

  it('appends the next page with load more', () => {
    let call = 0;
    cy.intercept('GET', '/v1/showcase/products*', (req) => {
      call += 1;
      if (call === 1) {
        req.reply(
          ok([factories.product()], { page_size: 20, next_cursor: 'cursor-2', has_more: true }),
        );
      } else {
        req.reply(
          ok([factories.product({ id: 'product-2', name: 'Suco MiniMart' })], {
            page_size: 20,
            next_cursor: null,
            has_more: false,
          }),
        );
      }
    }).as('products');

    cy.visit('/');
    cy.wait('@products').its('request.url').should('not.include', 'cursor=');
    cy.contains('button', 'Carregar mais').click();
    cy.wait('@products').its('request.url').should('include', 'cursor=cursor-2');
    cy.get('[data-cy="product-name"]').should('have.length', 2);
    cy.contains('Suco MiniMart').should('be.visible');
  });

  it('shows the empty state', () => {
    cy.intercept('GET', '/v1/showcase/products*', {
      body: ok([], { page_size: 20, next_cursor: null, has_more: false }),
    }).as('products');

    cy.visit('/');
    cy.wait('@products');
    cy.get('[data-cy="empty-state"]').should('be.visible');
  });

  it('retries after a product load error', () => {
    let call = 0;
    cy.intercept('GET', '/v1/showcase/products*', (req) => {
      call += 1;
      if (call === 1) {
        req.reply({
          statusCode: 500,
          body: { success: false, error: { code: 'BOOM', message: 'Falha ao carregar produtos' } },
        });
      } else {
        req.reply(ok([factories.product()], { page_size: 20, next_cursor: null, has_more: false }));
      }
    }).as('products');

    cy.visit('/');
    cy.contains('Tentar novamente').click();
    cy.wait('@products');
    cy.get('[data-cy="product-card"]').should('have.length', 1);
  });

  it('opens product details', () => {
    cy.intercept('GET', '/v1/showcase/products/product-1', {
      body: ok(factories.product()),
    }).as('product');
    cy.intercept('GET', '/v1/showcase/products/product-1/reviews*', {
      body: ok([], { page: 1, page_size: 20, total: 0, total_pages: 0 }),
    }).as('reviews');

    cy.visit('/products/product-1');
    cy.wait(['@product', '@reviews']);
    cy.contains('Café MiniMart').should('be.visible');
    cy.contains('R$ 12,99').should('be.visible');
    cy.contains('SKU SKU-1').should('be.visible');
    cy.contains('Café torrado').should('be.visible');
    cy.contains('10 em estoque').should('be.visible');
  });

  it('renders approved reviews and hides pending ones', () => {
    cy.intercept('GET', '/v1/showcase/products/product-1', {
      body: ok(
        factories.product({
          total_rating: 4.5,
          total_reviews: 1,
        }),
      ),
    }).as('product');
    cy.intercept('GET', '/v1/showcase/products/product-1/reviews*', {
      body: ok([factories.review()], { page: 1, page_size: 20, total: 1, total_pages: 1 }),
    }).as('reviews');

    cy.visit('/products/product-1');
    cy.wait(['@product', '@reviews']);
    cy.get('label[for="tab-reviews"]').click();
    cy.get('[data-cy="review-item"]').should('have.length', 1);
    cy.contains('Ótimo produto, recomendo.').should('be.visible');
    cy.contains('Nenhuma avaliação aprovada ainda.').should('not.exist');
  });

  it('submits a review for moderation', () => {
    cy.loginAs();
    cy.intercept('GET', '/v1/showcase/products/product-1', {
      body: ok(factories.product()),
    }).as('product');
    cy.intercept('GET', '/v1/showcase/products/product-1/reviews*', {
      body: ok([], { page: 1, page_size: 20, total: 0, total_pages: 0 }),
    }).as('reviews');
    cy.intercept('POST', '/v1/products/product-1/reviews', {
      statusCode: 201,
      body: ok(factories.review({ status: 'pending' })),
    }).as('createReview');

    cy.visit('/products/product-1');
    cy.wait(['@product', '@reviews']);
    cy.get('label[for="tab-reviews"]').click();
    cy.get('[data-cy="review-form-open"]').click();
    cy.get('[data-cy="review-star-5"]').click();
    cy.get('[data-cy="review-description"]').type('Produto excelente, comprei novamente.');
    cy.get('[data-cy="review-submit"]').click();
    cy.wait('@createReview').its('request.body').should('deep.include', {
      description: 'Produto excelente, comprei novamente.',
      rating: 5,
    });
    cy.contains('[data-cy="toast-container"]', 'Avaliação enviada para moderação').should(
      'be.visible',
    );
  });

  it('adds an authenticated product to the cart', () => {
    cy.loginAs();
    cy.intercept('GET', '/v1/showcase/products*', {
      body: ok([factories.product()], { page_size: 20, next_cursor: null, has_more: false }),
    }).as('products');
    cy.intercept('POST', '/v1/cart/items', {
      body: ok(
        factories.cart({
          items: [factories.cartItem()],
          estimated_total_cents: 1299,
        }),
      ),
    }).as('addItem');

    cy.visit('/');
    cy.wait('@products');
    cy.get('[data-cy="add-to-cart"]').first().click();
    cy.wait('@addItem').its('request.body').should('deep.include', {
      product_id: 'product-1',
      quantity: 1,
    });
    cy.contains('[data-cy="toast-container"]', 'Produto adicionado ao carrinho').should(
      'be.visible',
    );
  });

  it('redirects anonymous add-to-cart to login with returnUrl', () => {
    cy.intercept('GET', '/v1/showcase/products*', {
      body: ok([factories.product()], { page_size: 20, next_cursor: null, has_more: false }),
    }).as('products');

    cy.visit('/');
    cy.wait('@products');
    cy.get('[data-cy="add-to-cart"]').first().click();
    cy.url().should('include', '/login');
    cy.url().should('include', 'returnUrl');
  });
});
