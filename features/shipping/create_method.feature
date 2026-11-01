Feature: CreateMethod with a new shipping method

    All possible scenarios are:
        - Error path: Blank name is rejected
        - Error path: Invalid day estimates are rejected
        - Happy path: Method got created with all fields

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Blank name is rejected
        When I create a shipping method with name "" and description "desc" and active true and min days 3 and max days 5
        Then I will get this error "entrada inválida"

    Scenario: Invalid day estimates are rejected
        When I create a shipping method with name "Standard" and description "desc" and active true and min days 5 and max days 3
        Then I will get this error "transição de estado inválida"

    Scenario: Method got created with all fields
        When I create a shipping method with name "Standard" and description "Standard delivery" and active true and min days 3 and max days 5
        Then I will have a stored shipping method with a generated id, name "Standard", description "Standard delivery", active true, estimated min days 3, estimated max days 5 and a created_at timestamp
