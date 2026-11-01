const live = Cypress.expose('live');
const spec = live ? describe : describe.skip;
spec('live auth cookie', () => {
  it('signs up, logs in, and survives a reload', () => {
    const username = `cypress_${Date.now()}`;

    cy.request({
      method: 'POST',
      url: '/v1/auth/signup',
      body: {
        username,
        email: `${username}@example.com`,
        password: 'password123',
      },
    });

    cy.visit('/login');
    cy.get('#username input').type(username);
    cy.get('#password input').type('password123');
    cy.contains('button', 'Entrar').click();
    cy.url().should('not.include', '/login');
    cy.reload();
    cy.contains(username).should('be.visible');
  });
});
