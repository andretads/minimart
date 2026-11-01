Feature: ListAddresses returns all addresses for a customer

    All possible scenarios are:
        - Error path: Empty customer ID is rejected
        - Happy path: Empty storage returns an empty list
        - Happy path: All addresses for the customer are returned
        - Happy path: Default address comes first, then by created_at descending
        - Happy path: Addresses of other customers are not included

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty customer ID is rejected
        When I list addresses for customer ""
        Then I will get this error "address: id do cliente é obrigatório"

    Scenario: Empty storage returns an empty list
        When I list addresses for customer "cust-1"
        Then I will get an empty list of addresses

    Scenario: All addresses for the customer are returned
        Given I have an address "addr-1" for customer "cust-1" with label "Home"
        And I have an address "addr-2" for customer "cust-1" with label "Work"
        When I list addresses for customer "cust-1"
        Then I will get a list with 2 addresses

    Scenario: Default address comes first, then by created_at descending
        Given I have an address "addr-1" for customer "cust-1" with label "Work" and is_default false
        And I have an address "addr-2" for customer "cust-1" with label "Home" and is_default true
        And I have an address "addr-3" for customer "cust-1" with label "Other" and is_default false
        When I list addresses for customer "cust-1"
        Then I will get a list with 3 addresses
        And the first address label is "Home"
        And the first address is default
        And the second address label is "Other"
        And the third address label is "Work"

    Scenario: Addresses of other customers are not included
        Given I have an address "addr-1" for customer "cust-1" with label "Home"
        And I have an address "addr-2" for customer "cust-2" with label "Work"
        When I list addresses for customer "cust-1"
        Then I will get a list with 1 address
        And the first address label is "Home"