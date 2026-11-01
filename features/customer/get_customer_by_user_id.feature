Feature: GetCustomerByUserID resolves user to their CustomerID

    All possible scenarios are:
        - Error path: Unknown user_id is rejected
        - Happy path: Customer resolved by user_id

    Background: Clear the storage
        Given I have empty storage

    Scenario: Unknown user_id is rejected
        When I get a customer by user_id "unknown"
        Then I will get this error "customer: não encontrado"

    Scenario: Customer resolved by user_id
        Given I have a customer with user_id "user-1" and display_name "John"
        When I get the customer by user_id "user-1"
        Then I will have a customer with user_id "user-1", display_name "John", status "active"