Feature: CheckoutCart creates an order from the cart items

    State machine on the cart: active -> checkout_pending -> consumed.
    Requires an Idempotency-Key. Delegates order placement to the order contract.
    On failure (insufficient stock, product not found, etc.), releases the pending state
    back to active. Mark-consumed update failure is non-fatal (order still returned).

    All possible scenarios are:
        - Error path: Blank user_id is rejected
        - Error path: Missing Idempotency-Key is rejected
        - Error path: Empty cart returns cart empty
        - Happy path: Places order via contract, cart becomes consumed with order id
        - Happy path: The shipping is priced against the cart subtotal
        - Happy path: A cart at the free shipping threshold ships free
        - Edge path: Retry with same key on consumed cart replays the same order
        - Edge path: Different key while checkout_pending returns checkout in progress
        - Edge path: Same key while checkout_pending retries the placement
        - Edge path: Insufficient stock releases pending state back to active
        - Edge path: Mark-consumed update failure is non-fatal (order returned)

    Background: Clear the storage
        Given I have empty cart storage
        And I have a catalog with product "p1" named "Widget" priced 500 cents (stock 10)
        And I have a cart for user "u1" with 1 item: "Widget" x2
        And I have a shipping quote "quote-user-1" for customer "cust-user-1" costing 599 cents with a free threshold of 5000 cents

    Scenario: Blank user_id is rejected
        When I checkout cart for user "" with idempotency key "k1"
        Then I will get this error "cart: parâmetro de consulta user_id é obrigatório"

    Scenario: Missing Idempotency-Key is rejected
        When I checkout cart for user "u1" with no idempotency key
        Then I will get this error "chave de idempotência é obrigatória"

    Scenario: Empty cart returns cart empty
        Given I have an active cart for user "u2" with 0 items
        When I checkout cart for user "u2" with idempotency key "k1"
        Then I will get this error "cart: carrinho vazio"

    Scenario: Places order via contract, cart becomes consumed with order id
        When I checkout cart for user "u1" with idempotency key "k1"
        Then I will have an order with 1 item: "Widget" x2 at 500 cents, total 1000 cents
        And the cart for user "u1" is consumed with that order id

    Scenario: The shipping is priced against the cart subtotal
        When I checkout cart for user "u1" with idempotency key "k1"
        Then I will have an order with 1 item: "Widget" x2 at 500 cents, total 1000 cents and shipping 599 cents

    Scenario: A cart at the free shipping threshold ships free
        Given I have a cart for user "u1" with 1 item: "Widget" x12
        When I checkout cart for user "u1" with idempotency key "k1"
        Then I will have an order with 1 item: "Widget" x12 at 500 cents, total 6000 cents and shipping 0 cents

    Scenario: Retry with same key on consumed cart replays the same order
        When I checkout cart for user "u1" with idempotency key "k1"
        And I checkout cart for user "u1" with idempotency key "k1" again
        Then both calls return the same order
        And the cart for user "u1" is consumed with that order id

    Scenario: Different key while checkout_pending returns checkout in progress
        Given a checkout is in progress for user "u1" with key "k1"
        When I checkout cart for user "u1" with idempotency key "k2"
        Then I will get this error "cart: finalização em andamento"

    Scenario: Same key while checkout_pending retries the placement
        Given a checkout is in progress for user "u1" with key "k1"
        When I checkout cart for user "u1" with idempotency key "k1"
        Then the placement is retried and the same order is returned

    Scenario: Insufficient stock releases pending state back to active
        Given product "p1" has 0 units in stock
        And I have a cart for user "u1" with 1 item: "Widget" x2
        When I checkout cart for user "u1" with idempotency key "k1"
        Then I will get this error "catalog: estoque insuficiente"
        And the cart for user "u1" is back to status "active"

    Scenario: Mark-consumed update failure is non-fatal
        Given the storage fails on the cart update after order placement
        When I checkout cart for user "u1" with idempotency key "k1"
        Then the order is still returned
        And a warning is logged for mark-consumed failure