Feature: ReleaseStock returns any outstanding stock held for an order

    Releasing is driven by the order's outbox event and is idempotent under
    at-least-once delivery. It is safe to call more than once for the same order.

    All possible scenarios are:
        - Edge path: Releasing twice is idempotent
        - Happy path: Release restores the product's stock

    Background: Clear the storage
        Given I have empty catalog storage
        And I have a product "p1" with 5 units in stock

    Scenario: Releasing twice is idempotent
        When order "o1" reserves 2 units of product "p1"
        And order "o1" releases its reservation
        And order "o1" releases its reservation again
        Then product "p1" has 5 units in stock

    Scenario: Release restores the product's stock
        When order "o1" reserves 2 units of product "p1"
        And order "o1" releases its reservation
        Then product "p1" has 5 units in stock