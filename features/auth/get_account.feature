Feature: GetAccount retrieves an account by ID

    Pure passthrough to the repository. No side effects.

    All possible scenarios are:
        - Error path: Unknown ID returns account not found
        - Happy path: Existing account returned

    Background: Clear the storage
        Given I have empty auth storage

    Scenario: Unknown ID returns account not found
        When I get the account with ID "missing"
        Then I will get this error "auth: conta não encontrada"

    Scenario: Existing account returned
        Given I have an account with ID "acc-1" and username "john"
        When I get the account with ID "acc-1"
        Then I will have an account with ID "acc-1", username "john"