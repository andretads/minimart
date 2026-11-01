Feature: DeactivateCustomer prevents a customer from performing new operations

    All possible scenarios are:
        - Error path: Unknown ID is rejected
        - Happy path: Customer deactivated
        - Happy path: Deactivating already deactivated is idempotent
        - Happy path: Version is bumped on deactivate

    Background: Clear the storage
        Given I have empty storage

    Scenario: Unknown ID is rejected
        When I deactivate customer "unknown"
        Then I will get this error "customer: não encontrado"

    Scenario: Customer deactivated
        Given I have a customer with user_id "user-1" and display_name "John"
        When I deactivate the customer
        Then I will have a customer with user_id "user-1", status "deactivated", a deactivated_at timestamp and version 2

    Scenario: Deactivating already deactivated is idempotent
        Given I have a customer with user_id "user-1"
        When I deactivate the customer
        And I deactivate the customer again
        Then I will have a customer with status "deactivated" and the original deactivated_at timestamp is preserved and version 3