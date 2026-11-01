Feature: PlaceOrder from the catalog prices

    Prices come from the catalog (never the client), stock is reserved through the
    catalog contract before the order is saved, and the order is written together
    with its OrderPlaced event via the transactional outbox. Placement is
    idempotent: the caller sends an Idempotency-Key, and a retry with the same key
    and parameters returns the original order.

    All possible scenarios are:
        - Error path: Order has no items
        - Error path: Item quantity is not positive
        - Error path: Product does not exist
        - Error path: Product does not have enough stock
        - Error path: Missing idempotency key
        - Error path: Storage failure releases the reservation
        - Edge path: Reusing a key with different parameters is a conflict
        - Edge path: A concurrent request is still in progress for the same key
        - Edge path: Redis is unavailable, so idempotency falls back to the database
        - Edge path: An order placed without shipping carries no shipping snapshot
        - Happy path: Order got placed, stock reserved, event outboxed
        - Happy path: A retry with the same key replays the stored order
        - Happy path: The shipping snapshot is stored with the order

    Background: Clear the storage
        Given I have empty storage
        And I have a catalog with product "p1" named "Widget" priced 500 cents and product "p2" named "Gadget" priced 250 cents

    Scenario: Order has no items
        When I place an order for user "u1" with no items
        Then I will get this error "pedido deve conter pelo menos um item"

    Scenario: Item quantity is not positive
        When I place an order for user "u1" with 0 units of product "p1"
        Then I will get this error "quantidade do item deve ser maior que zero"

    Scenario: Product does not exist
        When I place an order for user "u1" with 1 unit of product "nope"
        Then I will get this error "resolve product nope: catalog: produto não encontrado"

    Scenario: Product does not have enough stock
        Given product "p1" has 0 units in stock
        When I place an order for user "u1" with 1 unit of product "p1"
        Then I will get this error "catalog: estoque insuficiente"
        And no order is stored

    Scenario: Missing idempotency key
        When I place an order for user "u1" with 1 unit of product "p1" and no idempotency key
        Then I will get this error "chave de idempotência é obrigatória"

    Scenario: Storage failure releases the reservation
        Given the storage is failing
        When I place an order for user "u1" with 1 unit of product "p1"
        Then I will get this error "disk on fire"
        And the stock reservation for the order is released

    Scenario: Reusing a key with different parameters is a conflict
        When I place an order for user "u1" with 1 unit of product "p1" under idempotency key "k1"
        And I place an order for user "u1" with 2 units of product "p1" under idempotency key "k1"
        Then I will get this error "chave de idempotência reutilizada com parâmetros diferentes"

    Scenario: A concurrent request is still in progress for the same key
        Given an idempotency reservation exists for key "k1" for the same request parameters
        When I place an order for user "u1" with 1 unit of product "p1" under idempotency key "k1"
        Then I will get this error "já existe um pedido com esta chave de idempotência em processamento"

    Scenario: Redis is unavailable, so idempotency falls back to the database
        Given the idempotency cache is unavailable
        When I place an order for user "u1" with 1 unit of product "p1" under idempotency key "k1"
        Then I will have a stored order for user "u1" with 1 item and a total of 500 cents
        When I place an order for user "u1" with 1 unit of product "p1" under idempotency key "k1" again
        Then I will get the same order id as before

    Scenario: Order got placed, stock reserved, event outboxed
        When I place an order for user "u1" with 2 units of product "p1" and 1 unit of product "p2"
        Then I will have a stored order for user "u1" with 2 items and a total of 1250 cents
        And order stock was reserved for the order
        And a single OrderPlaced event is written to the outbox with order id matching the stored order

    Scenario: A retry with the same key replays the stored order
        When I place an order for user "u1" with 1 unit of product "p1" under idempotency key "k1"
        And I place an order for user "u1" with 1 unit of product "p1" under idempotency key "k1" again
        Then the same order id is returned
        And order stock was reserved exactly once

    Scenario: The shipping snapshot is stored with the order
        When I place an order for user "u1" with 1 unit of product "p1" shipping quote "q1" method "m1" costing 250 cents to "Ada Lovelace, 1 Main St, Testville, BR"
        Then I will have a stored order for user "u1" with a shipping quote "q1", a method "m1", a shipping cost of 250 cents and the destination "Ada Lovelace, 1 Main St, Testville, BR"
        And the order total is 750 cents

    Scenario: An order placed without shipping carries no shipping snapshot
        When I place an order for user "u1" with 1 unit of product "p1"
        Then I will have a stored order for user "u1" with no shipping quote and no shipping cost

    Scenario: A freshly placed order exposes its auto-cancel deadline
        Given I have an empty cart
        When I place an order
        Then the returned order will have status "placed"
        And the returned order will include an "auto_cancel_at" timestamp
