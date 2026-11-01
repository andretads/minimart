Feature: Authenticate verifies credentials and returns the account

    Lookup by username; wrong or missing password returns the same error
    (no user enumeration). On success, best-effort ensures a customer exists.

    All possible scenarios are:
        - Error path: Unknown username returns invalid credentials
        - Error path: Wrong password returns invalid credentials
        - Happy path: Valid credentials return the account
        - Edge path: Login best-effort ensures customer

    Background: Clear the storage
        Given I have empty auth storage

    Scenario: Unknown username returns invalid credentials
        When I authenticate with username "unknown" and password "secret123"
        Then I will get this error "auth: credenciais inválidas"

    Scenario: Wrong password returns invalid credentials
        Given I have an account with username "john" and password "secret123"
        When I authenticate with username "john" and password "wrong"
        Then I will get this error "auth: credenciais inválidas"

    Scenario: Valid credentials return the account
        Given I have an account with username "john" and password "secret123"
        When I authenticate with username "john" and password "secret123"
        Then I will get the account with username "john"

    Scenario: Login best-effort ensures customer
        Given I have an account with username "john" and password "secret123"
        And the customer service is unavailable
        When I authenticate with username "john" and password "secret123"
        Then I will get the account with username "john"
        And a warning is logged for customer ensure failure