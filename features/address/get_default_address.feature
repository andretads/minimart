Feature: GetDefaultAddress retrieves the default address for a customer

    All possible scenarios are:
        - Error path: Empty customer ID is rejected
        - Error path: No default address exists
        - Happy path: Default address returned

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty customer ID is rejected
        When I get the default address for customer ""
        Then I will get this error "address: id do cliente é obrigatório"

    Scenario: No default address exists
        Given I have an address "addr-1" for customer "cust-1" with is_default false
        When I get the default address for customer "cust-1"
        Then I will get this error "address: endereço não encontrado"

    Scenario: Default address returned
        Given I have an address "addr-1" for customer "cust-1" with label "Home" and is_default true
        When I get the default address for customer "cust-1"
        Then I will have an address with id "addr-1", label "Home", is_default true