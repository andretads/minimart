Feature: GetAddress retrieves an address by ID

    All possible scenarios are:
        - Error path: Empty customer ID is rejected
        - Error path: Unknown ID is rejected
        - Error path: Address owned by another customer is not found
        - Happy path: Address retrieved by ID

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty customer ID is rejected
        When I get an address by ID "addr-1" for customer ""
        Then I will get this error "address: id do cliente é obrigatório"

    Scenario: Unknown ID is rejected
        When I get an address by ID "unknown" for customer "cust-1"
        Then I will get this error "address: endereço não encontrado"

    Scenario: Address owned by another customer is not found
        Given I have an address "addr-1" for customer "cust-1"
        When I get an address by ID "addr-1" for customer "cust-2"
        Then I will get this error "address: endereço não encontrado"

    Scenario: Address retrieved by ID
        Given I have an address "addr-1" for customer "cust-1" with label "Home" and full_name "John Doe"
        When I get an address by ID "addr-1" for customer "cust-1"
        Then I will have an address with id "addr-1", label "Home", full_name "John Doe", version 1 and a created_at timestamp