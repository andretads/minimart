Feature: GetOrCreateCart returns the user's cart or creates a new one

    Uses EnsureCustomer to provision a customer, then gets or creates the active cart.
    Concurrent creates for the same user are safe: the second call re-fetches.

    All possible scenarios are:
        - Error path: Blank user_id is rejected
        - Happy path: Creates cart when none exists
        - Happy path: Returns existing active cart
        - Edge path: Concurrent create for same user returns same cart

    Background: Clear the storage
        Given I have empty cart storage
        And I have a catalog with product "p1" named "Widget" priced 500 cents

    Scenario: Blank user_id is rejected
        When I get or create cart for user ""
        Then I will get this error "cart: parâmetro de consulta user_id é obrigatório"

    Scenario: Creates cart when none exists
        When I get or create cart for user "u1"
        Then I will have an active cart for user "u1" with 0 items and version 1

    Scenario: Returns existing active cart
        Given I have an active cart for user "u1" with 1 item: "Widget" x2
        When I get or create cart for user "u1"
        Then I will have the same cart with 1 item: "Widget" x2

    Scenario: Concurrent create for same user returns same cart
        When I get or create cart for user "u-race"
        And I get or create cart for user "u-race" again
        Then both calls return the same cart ID
        And exactly one active cart exists for user "u-race"