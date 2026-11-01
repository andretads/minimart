Feature: ApplyPaymentOutcome

    The order domain reacts to terminal payment outcomes published by the payment
    domain. It owns its status; the payment never writes into the order directly.
    A failed payment is terminal for fulfilment and releases the order's stock
    reservation; an unresolved payment holds it.

    All possible scenarios are:
        - Error path: Order does not exist
        - Edge path: A duplicate delivery must not change anything
        - Edge path: A cancelled order must not be resurrected by a late payment
        - Edge path: A failed payment must not be resurrected by a late success
        - Edge path: An unknown payment can still be resolved
        - Edge path: An order without shipping hands nothing over to the shipping domain
        - Happy path: Payment succeeded marks the order paid
        - Happy path: Payment failed marks the order payment_failed and releases stock
        - Happy path: Payment unknown marks the order payment_unknown and holds stock
        - Happy path: Payment succeeded on an order with shipping hands the snapshot over

    Background: Clear the storage
        Given I have empty storage

    Scenario: Order does not exist
        Given I have no orders stored
        When a "succeeded" payment outcome arrives for order "missing"
        Then I will get this error "pedido não encontrado"

    Scenario: A duplicate delivery must not change anything
        Given I have a stored order with id "o1" that is already paid
        When a "succeeded" payment outcome arrives for order "o1"
        Then the order with id "o1" has status "paid"
        And no order write was attempted

    Scenario: A cancelled order must not be resurrected by a late payment
        Given I have a stored order with id "o1" that is already cancelled
        When a "succeeded" payment outcome arrives for order "o1"
        Then the order with id "o1" has status "cancelled"
        And no order write was attempted

    Scenario: A failed payment must not be resurrected by a late success
        Given I have a stored order with id "o1" with status "payment_failed"
        When a "succeeded" payment outcome arrives for order "o1"
        Then the order with id "o1" has status "payment_failed"
        And no order write was attempted

    Scenario: An unknown payment can still be resolved
        Given I have a stored order with id "o1" with status "payment_unknown"
        When a "succeeded" payment outcome arrives for order "o1"
        Then the order with id "o1" has status "paid"

    Scenario: Payment succeeded marks the order paid
        Given I have a stored order with id "o1" with status "placed"
        When a "succeeded" payment outcome arrives for order "o1"
        Then the order with id "o1" has status "paid"

    Scenario: Payment failed marks the order payment_failed and releases stock
        Given I have a stored order with id "o1" with status "placed"
        When a "failed" payment outcome arrives for order "o1"
        Then the order with id "o1" has status "payment_failed"
        And a single OrderStockRelease event will be published with order id "o1"

    Scenario: Payment unknown marks the order payment_unknown and holds stock
        Given I have a stored order with id "o1" with status "placed"
        When an "unknown" payment outcome arrives for order "o1"
        Then the order with id "o1" has status "payment_unknown"
        And no OrderStockRelease event will be published

    Scenario: An order without shipping hands nothing over to the shipping domain
        Given I have a stored order with id "o1" with status "placed" and no shipping
        When a "succeeded" payment outcome arrives for order "o1"
        Then the order with id "o1" has status "paid"
        And no OrderPaid event will be published

    Scenario: Payment succeeded on an order with shipping hands the snapshot over
        Given I have a stored order with id "o1" with status "placed" shipping quote "q1" method "m1" to "Ada Lovelace, 1 Main St, Testville, BR"
        When a "succeeded" payment outcome arrives for order "o1"
        Then the order with id "o1" has status "paid"
        And a single OrderPaid event will be published with order id "o1", shipping quote "q1", method "m1" and the destination "Ada Lovelace, 1 Main St, Testville, BR"
