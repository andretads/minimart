Feature: ClearCart removes all items from the cart

    Uses optimistic locking on cart version.
    Cart remains active after clearing.

    All possible scenarios are:
        - Error path: Blank user_id is rejected
        - Error path: Already-empty cart returns cart empty
        - Happy path: Removes all items, cart stays active

    Background: Clear the storage
        Given I have empty cart storage
        And I have a catalog with product "p1" named "Widget" priced 500 cents

    Scenario: Blank user_id is rejected
        When I clear cart for user ""
        Then I will get this error "cart: parâmetro de consulta user_id é obrigatório"

    Scenario: Already-empty cart returns cart empty
        Given I have an active cart for user "u1" with 0 items
        When I clear cart for user "u1"
        Then I will get this error "cart: carrinho vazio"

    Scenario: Removes all items, cart stays active
        Given I have an active cart for user "u1" with 3 items: "Widget" x2, "Gadget" x1
        When I clear cart for user "u1"
        Then I will have an empty active cart for user "u1" with version 2