Feature: DeleteAddress deletes an address by ID

    All possible scenarios are:
        - Error path: Empty customer ID is rejected
        - Error path: Unknown ID is rejected
        - Error path: Address owned by another customer is not found
        - Happy path: Address deleted
        - Happy path: Deleted address no longer retrievable
        - Happy path: Deleted address no longer appears in list

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty customer ID is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I delete the address "addr-1" for customer ""
        Then I will get this error "address: id do cliente é obrigatório"

    Scenario: Unknown ID is rejected
        When I delete the address "unknown" for customer "cust-1"
        Then I will get this error "address: endereço não encontrado"

    Scenario: Address owned by another customer is not found
        Given I have an address "addr-1" for customer "cust-1"
        When I delete the address "addr-1" for customer "cust-2"
        Then I will get this error "address: endereço não encontrado"

    Scenario: Address deleted
        Given I have an address "addr-1" for customer "cust-1"
        When I delete the address "addr-1" for customer "cust-1"
        Then the address "addr-1" will be removed

    Scenario: Deleted address no longer retrievable
        Given I have an address "addr-1" for customer "cust-1"
        When I delete the address "addr-1" for customer "cust-1"
        And I get the address "addr-1" for customer "cust-1"
        Then I will get this error "address: endereço não encontrado"

    Scenario: Deleted address no longer appears in list
        Given I have an address "addr-1" for customer "cust-1" with label "Home"
        And I have an address "addr-2" for customer "cust-1" with label "Work"
        When I delete the address "addr-1" for customer "cust-1"
        And I list addresses for customer "cust-1"
        Then I will get a list with 1 address
        And the first address label is "Work"