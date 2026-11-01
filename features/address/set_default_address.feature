Feature: SetDefaultAddress promotes an address to default

    All possible scenarios are:
        - Error path: Empty customer ID is rejected
        - Error path: Unknown ID is rejected
        - Error path: Address owned by another customer is not found
        - Happy path: Address set as default
        - Happy path: Version is bumped on set default
        - Happy path: Previous default is unset
        - Happy path: Setting an already default address keeps it default

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty customer ID is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I set the address "addr-1" as default for customer ""
        Then I will get this error "address: id do cliente é obrigatório"

    Scenario: Unknown ID is rejected
        When I set the address "unknown" as default for customer "cust-1"
        Then I will get this error "address: endereço não encontrado"

    Scenario: Address owned by another customer is not found
        Given I have an address "addr-1" for customer "cust-1"
        When I set the address "addr-1" as default for customer "cust-2"
        Then I will get this error "address: endereço não encontrado"

    Scenario: Address set as default
        Given I have an address "addr-1" for customer "cust-1" with is_default false
        When I set the address "addr-1" as default for customer "cust-1"
        Then I will have a stored address with id "addr-1", is_default true, version 2

    Scenario: Version is bumped on set default
        Given I have an address "addr-1" for customer "cust-1" with is_default false
        When I set the address "addr-1" as default for customer "cust-1"
        Then the stored address "addr-1" will have version 2

    Scenario: Previous default is unset
        Given I have an address "addr-1" for customer "cust-1" with is_default true
        And I have an address "addr-2" for customer "cust-1" with is_default false
        When I set the address "addr-2" as default for customer "cust-1"
        Then the address "addr-2" will be default
        And the address "addr-1" will no longer be default

    Scenario: Setting an already default address keeps it default
        Given I have an address "addr-1" for customer "cust-1" with is_default true
        When I set the address "addr-1" as default for customer "cust-1"
        Then I will have a stored address with id "addr-1", is_default true