Feature: CreateCustomer with a new customer record

    All possible scenarios are:
        - Error path: Blank user_id is rejected
        - Error path: Display name too long is rejected
        - Happy path: Customer got created with no problems
        - Happy path: Customer without display name gets created
        - Error path: Duplicate user_id is rejected

    Background: Clear the storage
        Given I have empty storage

    Scenario: Blank user_id is rejected
        When I create a customer with user_id "" and display_name "John"
        Then I will get this error "customer: user_id é obrigatório"

    Scenario: Display name too long is rejected
        When I create a customer with user_id "user-1" and display_name "a" repeated 201 times
        Then I will get this error "customer: nome de exibição muito longo"

    Scenario: Customer got created with no problems
        When I create a customer with user_id "user-1" and display_name "John"
        Then I will have a stored customer with a generated id, user_id "user-1", display_name "John", status "active" and a created_at timestamp

    Scenario: Customer without display name gets created
        When I create a customer with user_id "user-2" and display_name ""
        Then I will have a stored customer with a generated id, user_id "user-2", display_name "", status "active" and a created_at timestamp

    Scenario: Duplicate user_id is rejected
        Given I have a customer with user_id "user-1"
        When I create a customer with user_id "user-1" and display_name "Jane"
        Then I will get this error "customer: já existe para o usuário"