Feature: GetCart returns the user's active cart (read-only, does not create)

    Uses GetCustomerByUserID to resolve the user to a customer, then fetches the active cart.
    Does not create a customer or cart if missing.

    All possible scenarios are:
        - Error path: Blank user_id is rejected
        - Error path: Customer has no cart returns cart not found
        - Happy path: Returns cart with items

    Background: Clear the storage
        Given I have empty cart storage
        And I have a catalog with product "p1" named "Widget" priced 500 cents

    Scenario: Blank user_id is rejected
        When I get cart for user ""
        Then I will get this error "cart: parâmetro de consulta user_id é obrigatório"

    Scenario: Customer has no cart returns cart not found
        When I get cart for user "u-no-cart"
        Then I will get this error "cart: carrinho não encontrado"

    Scenario: Returns cart with items
        Given I have an active cart for user "u1" with 2 items: "Widget" x2 at 500 cents
        When I get cart for user "u1"
        Then I will have a cart for user "u1" with 2 items: "Widget" x2 at 500 cents each