import { ok } from '../support/api';
import * as factories from '../support/factories';

const validPngDataUrl =
  'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==';

describe('profile', () => {
  beforeEach(() => {
    cy.loginAs();
    cy.intercept('GET', '/v1/customers?user_id=user-1', {
      body: ok(factories.customer()),
    }).as('customer');
    cy.intercept('GET', '/v1/avatars/user-1', {
      body: ok(factories.avatar()),
    }).as('avatar');
    cy.intercept('GET', '/v1/addresses', {
      body: ok([factories.address()]),
    }).as('addresses');
    cy.intercept('GET', '/v1/notifications/preferences', {
      body: ok(factories.notificationPreferences()),
    }).as('preferences');
  });

  it('loads the customer profile and address', () => {
    cy.visit('/profile');
    cy.wait(['@customer', '@avatar', '@addresses']);
    cy.contains('Cypress User').should('be.visible');
    cy.contains('Ativo').should('be.visible');
    cy.contains('customer-1').should('be.visible');

    cy.get('[data-cy="profile-tab-addresses"]').click();
    cy.get('[data-cy="address-card"]').should('have.length', 1);
    cy.contains('Rua Teste, 1').should('be.visible');
  });

  it('saves the display name with version', () => {
    cy.intercept('PATCH', '/v1/customers/customer-1', {
      body: ok(factories.customer({ display_name: 'Novo Nome', version: 2 })),
    }).as('updateCustomer');

    cy.visit('/profile');
    cy.wait(['@customer', '@avatar', '@addresses']);
    cy.get('input#displayName').clear().type('Novo Nome');
    cy.get('[data-cy="save-profile"]').click();
    cy.wait('@updateCustomer').its('request.body').should('deep.include', {
      display_name: 'Novo Nome',
      version: 1,
    });
    cy.contains('Alterações salvas').should('be.visible');
  });

  it('uploads an avatar', () => {
    cy.intercept('PUT', '/v1/avatars/user-1', {
      body: ok(factories.avatar({ image_url: validPngDataUrl })),
    }).as('uploadAvatar');

    cy.visit('/profile');
    cy.wait(['@customer', '@avatar', '@addresses']);
    cy.get('[data-cy="avatar-input"]').selectFile(
      {
        contents: Cypress.Buffer.from('fake-png'),
        fileName: 'avatar.png',
        mimeType: 'image/png',
      },
      { force: true },
    );
    cy.wait('@uploadAvatar');
    cy.get('[data-cy="avatar-button"] img').should('have.attr', 'src', validPngDataUrl);
  });

  it('changes the password', () => {
    cy.intercept('POST', '/v1/auth/change-password', {
      body: ok({ changed: true }),
    }).as('changePassword');

    cy.visit('/profile');
    cy.wait(['@customer', '@avatar', '@addresses']);
    cy.get('[data-cy="profile-tab-security"]').click();
    cy.get('input#currentPassword').type('oldpass');
    cy.get('input#newPassword').type('newpass');
    cy.get('input#confirmNewPassword').type('newpass');
    cy.get('[data-cy="change-password"]').click();
    cy.wait('@changePassword').its('request.body').should('deep.include', {
      current_password: 'oldpass',
      new_password: 'newpass',
      confirm_new_password: 'newpass',
    });
    cy.contains('[data-cy="toast-container"]', 'Senha alterada com sucesso').should('be.visible');
  });

  it('creates an address', () => {
    cy.intercept('POST', '/v1/addresses', {
      body: ok(factories.address({ id: 'address-2', label: 'Trabalho', is_default: false })),
    }).as('createAddress');

    cy.visit('/profile');
    cy.wait(['@customer', '@avatar', '@addresses']);
    cy.get('[data-cy="profile-tab-addresses"]').click();
    cy.get('[data-cy="add-address"]').click();
    cy.get('input#label').type('Trabalho');
    cy.get('input#fullName').type('Cypress User');
    cy.get('input#street').type('Av. Paulista, 1000');
    cy.get('input#city').type('São Paulo');
    cy.get('input#state').type('SP');
    cy.get('input#zip').type('01310-100');
    cy.get('input#country').clear().type('BR');
    cy.get('[data-cy="save-address"]').click();
    cy.wait('@createAddress').its('request.body').should('deep.include', {
      label: 'Trabalho',
      full_name: 'Cypress User',
      street: 'Av. Paulista, 1000',
      city: 'São Paulo',
      state: 'SP',
      zip: '01310-100',
      country: 'BR',
      is_default: false,
    });
    cy.contains('[data-cy="toast-container"]', 'Endereço criado').should('be.visible');
  });

  it('updates a notification preference', () => {
    cy.intercept('PUT', '/v1/notifications/preferences', {
      body: ok(factories.notificationPreferences({ sms_enabled: true })),
    }).as('updatePreference');

    cy.visit('/profile');
    cy.wait(['@customer', '@avatar', '@addresses']);
    cy.get('[data-cy="profile-tab-notifications"]').click();
    cy.wait('@preferences');
    cy.get('[data-cy="notification-setting-smsEnabled"]')
      .should('not.be.checked')
      .check({ force: true });
    cy.wait('@updatePreference').its('request.body').should('deep.include', {
      email_enabled: true,
      sms_enabled: true,
      push_enabled: false,
      marketing_enabled: false,
    });
    cy.get('[data-cy="notification-setting-smsEnabled"]').should('be.checked');
  });
});
