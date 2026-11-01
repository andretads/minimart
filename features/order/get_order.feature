Feature: GetOrder by id

    All possible scenarios are:
        - Error path: Order does not exist
        - Happy path: Order got returned with no problems
        - Happy path: The shipping snapshot of the order is returned

    Background: Clear the storage
        Given I have empty storage

    Scenario: Order does not exist
        Given I have no orders stored
        When I get the order with id "missing"
        Then I will get this error "pedido não encontrado"

    Scenario: Order got returned with no problems
        Given I have a stored order with id "o1" for user "u1" with a total of 500 cents
        When I get the order with id "o1"
        Then I will get an order with id "o1" for user "u1" with a total of 500 cents

    Scenario: The shipping snapshot of the order is returned
        Given I have a stored order with id "o1" for user "u1" with a total of 750 cents shipping quote "q1" method "m1" costing 250 cents to "Ada Lovelace, 1 Main St, Testville, BR"
        When I get the order with id "o1"
        Then I will get an order with id "o1" with a shipping quote "q1", a method "m1", a shipping cost of 250 cents and the destination "Ada Lovelace, 1 Main St, Testville, BR"

    Scenario: A placed order exposes its auto-cancel deadline
        Given I have a stored order with id "o1" for user "u1" with a total of 500 cents
        And the order with id "o1" has status "placed"
        When I get the order with id "o1"
        Then I will get an order with an "auto_cancel_at" timestamp 5 minutes after its creation
