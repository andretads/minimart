Feature: UpdateAddress updates an address record

    All possible scenarios are:
        - Error path: Empty customer ID is rejected
        - Error path: Invalid version is rejected
        - Error path: Unknown ID is rejected
        - Error path: Address owned by another customer is not found
        - Error path: Stale version is rejected
        - Error path: Empty label is rejected
        - Error path: Label too long is rejected
        - Error path: Empty full name is rejected
        - Error path: Full name too long is rejected
        - Error path: Empty street is rejected
        - Error path: Street too long is rejected
        - Error path: Empty city is rejected
        - Error path: City too long is rejected
        - Error path: Empty state is rejected
        - Error path: State too long is rejected
        - Error path: Empty zip is rejected
        - Error path: Zip too long is rejected
        - Error path: Empty country is rejected
        - Error path: Country too long is rejected
        - Error path: Phone too long is rejected
        - Happy path: Address got updated with no problems
        - Happy path: Version is bumped on successful update
        - Happy path: Created_at is unchanged
        - Happy path: Fields are trimmed on update
        - Happy path: Setting is_default promotes the address to default

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty customer ID is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "" to label "Work"
        Then I will get this error "address: id do cliente é obrigatório"

    Scenario: Invalid version is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to label "Work" with version 0
        Then I will get this error "address: versão inválida"

    Scenario: Unknown ID is rejected
        When I update the address "missing" for customer "cust-1" to label "Work" with version 1
        Then I will get this error "address: endereço não encontrado"

    Scenario: Address owned by another customer is not found
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-2" to label "Work" with version 1
        Then I will get this error "address: endereço não encontrado"

    Scenario: Stale version is rejected
        Given I have an address "addr-1" for customer "cust-1"
        And I hold a copy of the address "addr-1" at its current version
        And another writer updated the address "addr-1"
        When I update my stale copy of the address "addr-1" to label "Work" with version 1
        Then I will get this error "address: versão inválida"

    Scenario: Empty label is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to label "   " with version 1
        Then I will get this error "address: rótulo é obrigatório"

    Scenario: Label too long is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to label "a" repeated 51 times with version 1
        Then I will get this error "address: rótulo excede o tamanho máximo"

    Scenario: Empty full name is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to full_name "   " with version 1
        Then I will get this error "address: nome completo é obrigatório"

    Scenario: Full name too long is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to full_name "a" repeated 101 times with version 1
        Then I will get this error "address: nome completo excede o tamanho máximo"

    Scenario: Empty street is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to street "   " with version 1
        Then I will get this error "address: logradouro é obrigatório"

    Scenario: Street too long is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to street "a" repeated 201 times with version 1
        Then I will get this error "address: logradouro excede o tamanho máximo"

    Scenario: Empty city is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to city "   " with version 1
        Then I will get this error "address: cidade é obrigatória"

    Scenario: City too long is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to city "a" repeated 101 times with version 1
        Then I will get this error "address: cidade excede o tamanho máximo"

    Scenario: Empty state is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to state "   " with version 1
        Then I will get this error "address: estado é obrigatório"

    Scenario: State too long is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to state "a" repeated 101 times with version 1
        Then I will get this error "address: estado excede o tamanho máximo"

    Scenario: Empty zip is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to zip "   " with version 1
        Then I will get this error "address: CEP é obrigatório"

    Scenario: Zip too long is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to zip "a" repeated 21 times with version 1
        Then I will get this error "address: CEP excede o tamanho máximo"

    Scenario: Empty country is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to country "   " with version 1
        Then I will get this error "address: país é obrigatório"

    Scenario: Country too long is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to country "a" repeated 101 times with version 1
        Then I will get this error "address: país excede o tamanho máximo"

    Scenario: Phone too long is rejected
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to phone "a" repeated 31 times with version 1
        Then I will get this error "address: telefone excede o tamanho máximo"

    Scenario: Address got updated with no problems
        Given I have an address "addr-1" for customer "cust-1" with label "Home"
        When I update the address "addr-1" for customer "cust-1" to label "Work", full_name "Jane Smith", street "456 Office Ave", city "San Francisco", state "CA", zip "94102", country "USA", phone "+1-555-5678" with version 1
        Then I will have a stored address with id "addr-1", label "Work", full_name "Jane Smith", street "456 Office Ave", city "San Francisco", state "CA", zip "94102", country "USA", phone "+1-555-5678", version 2

    Scenario: Version is bumped on successful update
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to label "Work" with version 1
        Then the stored address "addr-1" will have version 2

    Scenario: Created_at is unchanged
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to label "Work" with version 1
        Then the created_at of address "addr-1" is unchanged

    Scenario: Fields are trimmed on update
        Given I have an address "addr-1" for customer "cust-1"
        When I update the address "addr-1" for customer "cust-1" to label "  Work  " with version 1
        Then I will have a stored address with label "Work"

    Scenario: Setting is_default promotes the address to default
        Given I have an address "addr-1" for customer "cust-1" with is_default false
        When I update the address "addr-1" for customer "cust-1" to label "Work" with is_default true and version 1
        Then I will have a stored address with id "addr-1", is_default true, version 2