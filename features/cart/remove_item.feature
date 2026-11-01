Feature: RemoveItem removes an item from the cart

    Uses optimistic locking on cart version.

    All possible scenarios are:
        - Error path: Blank user_id is rejected
        - Error path: Unknown product in cart returns item not found
        - Happy path: Removes item from cart
        - Happy path: Removing last item leaves an empty active cart

    Background: Clear the storage
        Given I have empty cart storage
        And I have a catalog with product "p1" named "Widget" priced 500 cents
        And I have an active cart for user "u1" with 1 item: "Widget" x2 at 500 cents (version 1)

    Scenario: Blank user_id is rejected
        When I remove product "p1" from cart for user ""
        Then I will get this error "cart: parâmetro de consulta user_id é obrigatório"

    Scenario: Unknown product in cart returns item not found
        When I remove product "nope" from cart for user "u1"
        Then I will get this error "cart: item não encontrado"

    Scenario: Removes item from cart
        When I remove product "p1" from cart for user "u1"
        Then I will have an empty active cart for user "u1" with version 2

    Scenario: Removing last item leaves an empty active cart
        Given I have an active cart for user "u2" with 1 item: "Widget" x1
        When I remove product "p1" from cart for user "u2"
        Then I will have an empty active cart for user "u2" with status "active"