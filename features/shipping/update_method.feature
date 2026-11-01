Feature: UpdateMethod of a shipping method

    All possible scenarios are:
        - Error path: Method does not exist
        - Error path: Blank name is rejected
        - Error path: Invalid day estimates are rejected
        - Happy path: Method got updated

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Method does not exist
        When I update shipping method "missing" with name "Standard" and description "desc" and active true and min days 3 and max days 5
        Then I will get this error "método de frete não encontrado"

    Scenario: Blank name is rejected
        Given I have a shipping method "sm1" named "Standard" that is active
        When I update shipping method "sm1" with name "" and description "desc" and active true and min days 3 and max days 5
        Then I will get this error "entrada inválida"

    Scenario: Invalid day estimates are rejected
        Given I have a shipping method "sm1" named "Standard" that is active
        When I update shipping method "sm1" with name "Standard" and description "desc" and active true and min days 5 and max days 3
        Then I will get this error "transição de estado inválida"

    Scenario: Method got updated
        Given I have a shipping method "sm1" named "Standard" that is active
        When I update shipping method "sm1" with name "Express" and description "Express delivery" and active false and min days 1 and max days 2
        Then the shipping method "sm1" will have name "Express", description "Express delivery", active false, estimated min days 1 and estimated max days 2
