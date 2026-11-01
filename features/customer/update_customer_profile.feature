Feature: UpdateCustomerProfile updates customer profile fields

    All possible scenarios are:
        - Error path: Blank display name is allowed (clears it)
        - Error path: Display name too long is rejected
        - Error path: Unknown ID is rejected
        - Error path: Missing version is rejected
        - Error path: Stale version (concurrent update) is rejected
        - Error path: Stale ETag in If-Match is rejected
        - Happy path: Customer display name updated
        - Happy path: Version is bumped on successful update

    Background: Clear the storage
        Given I have empty storage

    Scenario: Display name too long is rejected
        Given I have a customer with user_id "user-1"
        When I update the customer with display_name "a" repeated 201 times
        Then I will get this error "customer: nome de exibição muito longo"

    Scenario: Unknown ID is rejected
        When I update customer "unknown" with display_name "John"
        Then I will get this error "customer: não encontrado"

    Scenario: Missing version is rejected
        Given I have a customer with user_id "user-1" and display_name "John"
        When I update the customer with display_name "Jane" without a version
        Then I will get this error "customer: versão deve ser maior que zero"

    Scenario: Stale version is rejected
        Given I have a customer with user_id "user-1" and display_name "John"
        And I hold a copy of the customer at its current version
        And another writer updates the customer to display_name "Jane"
        When I update the customer with display_name "Jane2" using my stale copy
        Then I will get this error "customer: conflito de bloqueio"

    Scenario: Stale ETag is rejected
        Given I have a customer with user_id "user-1" and display_name "John"
        When I update the customer with display_name "Jane" with If-Match tag for version 99
        Then I will get this error "customer: conflito de bloqueio"

    Scenario: Customer display name updated
        Given I have a customer with user_id "user-1" and display_name "John"
        When I update the customer with display_name "Jane"
        Then I will have a customer with user_id "user-1", display_name "Jane", status "active" and version 2

    Scenario: Customer is updated using the ETag from GET
        Given I have a customer with user_id "user-1" and display_name "John"
        When I update the customer with display_name "Jane" with If-Match tag for version 1
        Then I will have a customer with user_id "user-1", display_name "Jane", status "active" and version 2

    Scenario: Display name cleared
        Given I have a customer with user_id "user-1" and display_name "John"
        When I update the customer with display_name ""
        Then I will have a customer with user_id "user-1", display_name "", status "active" and version 2