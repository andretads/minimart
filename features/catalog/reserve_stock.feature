Feature: ReserveStock reserves each requested product's quantity for an order

    The catalog holds stock on behalf of an order while the order is alive.
    Reserving is synchronous (so a sold-out product is rejected before an order exists).
    The reservation is all-or-nothing and idempotent: a shortage reserves nothing,
    and a repeated call for the same order does not reserve twice.

    All possible scenarios are:
        - Error path: Reservation has no items
        - Error path: Reservation quantity is not positive
        - Error path: Product does not exist
        - Error path: Product does not have enough stock
        - Edge path: The reservation is all-or-nothing
        - Edge path: Re-reserving the same order does not reserve twice
        - Happy path: Reserve decrements the product's stock

    Background: Clear the storage
        Given I have empty catalog storage
        And I have a product "p1" with 5 units in stock

    Scenario: Reservation has no items
        When order "o1" reserves no items
        Then I will get this error "reservation quantity must be greater than zero"

    Scenario: Reservation quantity is not positive
        When order "o1" reserves 0 units of product "p1"
        Then I will get this error "reservation quantity must be greater than zero"

    Scenario: Product does not exist
        When order "o1" reserves 1 unit of product "nope"
        Then I will get this error "produto não encontrado"

    Scenario: Product does not have enough stock
        When order "o1" reserves 6 units of product "p1"
        Then I will get this error "estoque insuficiente"
        And product "p1" still has 5 units in stock

    Scenario: The reservation is all-or-nothing
        Given I have a product "p2" with 1 unit in stock
        When order "o1" reserves 2 units of product "p1" and 2 units of product "p2"
        Then I will get this error "estoque insuficiente"
        And product "p1" still has 5 units in stock

    Scenario: Re-reserving the same order does not reserve twice
        When order "o1" reserves 2 units of product "p1"
        And order "o1" reserves 2 units of product "p1" again
        Then product "p1" has 3 units in stock

    Scenario: Reserve decrements the product's stock
        When order "o1" reserves 2 units of product "p1"
        Then product "p1" has 3 units in stock