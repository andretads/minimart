Feature: GetCustomerByID retrieves customer by CustomerID

    All possible scenarios are:
        - Error path: Unknown ID is rejected
        - Happy path: Customer retrieved by ID

    Background: Clear the storage
        Given I have empty storage

    Scenario: Unknown ID is rejected
        When I get a customer by ID "unknown"
        Then I will get this error "customer: não encontrado"

    Scenario: Customer retrieved by ID
        Given I have a customer with user_id "user-1" and display_name "John"
        When I get the customer by its ID
        Then I will have a customer with user_id "user-1", display_name "John", status "active"