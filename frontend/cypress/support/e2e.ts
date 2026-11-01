import './commands';
import './api';

beforeEach(() => {
  if (Cypress.expose('live')) {
    return;
  }
  cy.mockApi();
});
