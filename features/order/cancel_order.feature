Feature: CancelOrder

    Cancelling is allowed only while the order is unpaid. A transition to
    cancelled marks the order and writes both an OrderCancelled event and an
    OrderStockRelease event to the outbox in the same transaction, so the catalog
    can give the reserved stock back.

    All possible scenarios are:
        - Error path: Order does not exist
        - Error path: Order is already cancelled
        - Error path: Order is paid (must be refunded/returned, not cancelled)
        - Error path: Storage failure
        - Happy path: Order got cancelled and its stock release is outboxed

    Background: Clear the storage
        Given I have empty storage

    Scenario: Order does not exist
        Given I have no orders stored
        When I cancel the order with id "missing"
        Then I will get this error "pedido não encontrado"

    Scenario: Order is already cancelled
        Given I have a stored order with id "o1" for user "u1" with a total of 500 cents
        And the order with id "o1" is already cancelled
        When I cancel the order with id "o1"
        Then I will get this error "pedido já está cancelado"

    Scenario: Order is paid
        Given I have a stored order with id "o1" for user "u1" with a total of 500 cents
        And the order with id "o1" is already paid
        When I cancel the order with id "o1"
        Then I will get this error "pedido não pode ser cancelado"
        And no order write was attempted

    Scenario: Storage failure
        Given I have a stored order with id "o1" for user "u1" with a total of 500 cents
        And the storage is failing
        When I cancel the order with id "o1"
        Then I will get this error "disk on fire"
        And no OrderCancelled event will be published

    Scenario: Order got cancelled and its stock release is outboxed
        Given I have a stored order with id "o1" for user "u1" with a total of 500 cents
        When I cancel the order with id "o1"
        Then I will have a stored order for user "u1" with status "cancelled"
        And a single OrderCancelled event will be published with order id "o1"
        And a single OrderStockRelease event will be published with order id "o1"
