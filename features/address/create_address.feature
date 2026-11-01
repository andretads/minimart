Feature: CreateAddress with a new address record

    All possible scenarios are:
        - Error path: Empty customer ID is rejected
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
        - Happy path: Address got created with all fields
        - Happy path: Address without phone gets created
        - Happy path: Address created as default
        - Happy path: Fields are trimmed on creation
        - Happy path: Country is normalized on creation

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty customer ID is rejected
        When I create an address with customer_id "" and label "Home"
        Then I will get this error "address: id do cliente é obrigatório"

    Scenario: Empty label is rejected
        When I create an address for customer "cust-1" with label "   "
        Then I will get this error "address: rótulo é obrigatório"

    Scenario: Label too long is rejected
        When I create an address for customer "cust-1" with label "a" repeated 51 times
        Then I will get this error "address: rótulo excede o tamanho máximo"

    Scenario: Empty full name is rejected
        When I create an address for customer "cust-1" with full_name "   "
        Then I will get this error "address: nome completo é obrigatório"

    Scenario: Full name too long is rejected
        When I create an address for customer "cust-1" with full_name "a" repeated 101 times
        Then I will get this error "address: nome completo excede o tamanho máximo"

    Scenario: Empty street is rejected
        When I create an address for customer "cust-1" with street "   "
        Then I will get this error "address: logradouro é obrigatório"

    Scenario: Street too long is rejected
        When I create an address for customer "cust-1" with street "a" repeated 201 times
        Then I will get this error "address: logradouro excede o tamanho máximo"

    Scenario: Empty city is rejected
        When I create an address for customer "cust-1" with city "   "
        Then I will get this error "address: cidade é obrigatória"

    Scenario: City too long is rejected
        When I create an address for customer "cust-1" with city "a" repeated 101 times
        Then I will get this error "address: cidade excede o tamanho máximo"

    Scenario: Empty state is rejected
        When I create an address for customer "cust-1" with state "   "
        Then I will get this error "address: estado é obrigatório"

    Scenario: State too long is rejected
        When I create an address for customer "cust-1" with state "a" repeated 101 times
        Then I will get this error "address: estado excede o tamanho máximo"

    Scenario: Empty zip is rejected
        When I create an address for customer "cust-1" with zip "   "
        Then I will get this error "address: CEP é obrigatório"

    Scenario: Zip too long is rejected
        When I create an address for customer "cust-1" with zip "a" repeated 21 times
        Then I will get this error "address: CEP excede o tamanho máximo"

    Scenario: Empty country is rejected
        When I create an address for customer "cust-1" with country "   "
        Then I will get this error "address: país é obrigatório"

    Scenario: Country too long is rejected
        When I create an address for customer "cust-1" with country "a" repeated 101 times
        Then I will get this error "address: país excede o tamanho máximo"

    Scenario: Phone too long is rejected
        When I create an address for customer "cust-1" with phone "a" repeated 31 times
        Then I will get this error "address: telefone excede o tamanho máximo"

    Scenario: Address got created with all fields
        When I create an address for customer "cust-1" with label "Home", full_name "John Doe", street "123 Main St", city "New York", state "NY", zip "10001", country "USA", phone "+1-555-1234", is_default false
        Then I will have a stored address with a generated id, customer_id "cust-1", label "Home", full_name "John Doe", street "123 Main St", city "New York", state "NY", zip "10001", country "USA", phone "+1-555-1234", is_default false, version 1 and a created_at timestamp

    Scenario: Address without phone gets created
        When I create an address for customer "cust-1" with label "Home", full_name "John Doe", street "123 Main St", city "New York", state "NY", zip "10001", country "USA", phone "", is_default false
        Then I will have a stored address with a generated id, customer_id "cust-1", label "Home", phone "", version 1 and a created_at timestamp

    Scenario: Address created as default
        When I create an address for customer "cust-1" with label "Home", full_name "John Doe", street "123 Main St", city "New York", state "NY", zip "10001", country "USA", phone "", is_default true
        Then I will have a stored address with a generated id, customer_id "cust-1", label "Home", is_default true, version 1 and a created_at timestamp

    Scenario: Fields are trimmed on creation
        When I create an address for customer "cust-1" with label "  Home  ", full_name "  John Doe  ", street "  123 Main St  ", city "  New York  ", state "  NY  ", zip "  10001  ", country "  USA  ", phone "  +1-555-1234  ", is_default false
        Then I will have a stored address with label "Home", full_name "John Doe", street "123 Main St", city "New York", state "NY", zip "10001", country "USA", phone "+1-555-1234"

    Scenario: Country is normalized on creation
        When I create an address for customer "cust-1" with label "Home", full_name "John Doe", street "123 Main St", city "New York", state "NY", zip "10001", country "brasil", phone "", is_default false
        Then I will have a stored address with country "BR"