Feature: UpdateItemQuantity updates an item's quantity in the cart

    Validates quantity > 0. Uses optimistic locking on cart version.
    Fetches product from catalog to ensure it still exists.

    All possible scenarios are:
        - Error path: Blank user_id is rejected
        - Error path: Quantity less than 1 is rejected
        - Error path: Unknown product in cart returns item not found
        - Happy path: Updates quantity and bumps version
        - Edge path: Stale version returns optimistic lock failed

    Background: Clear the storage
        Given I have empty cart storage
        And I have a catalog with product "p1" named "Widget" priced 500 cents
        And I have an active cart for user "u1" with 1 item: "Widget" x2 at 500 cents (version 1)

    Scenario: Blank user_id is rejected
        When I update quantity of product "p1" to 3 in cart for user ""
        Then I will get this error "cart: parâmetro de consulta user_id é obrigatório"

    Scenario: Quantity less than 1 is rejected
        When I update quantity of product "p1" to 0 in cart for user "u1"
        Then I will get this error "cart: quantidade inválida"

    Scenario: Unknown product in cart returns item not found
        When I update quantity of product "nope" to 1 in cart for user "u1"
        Then I will get this error "cart: item não encontrado"

    Scenario: Updates quantity and bumps version
        When I update quantity of product "p1" to 5 in cart for user "u1"
        Then I will have a cart for user "u1" with 1 item: "Widget" x5 at 500 cents
        And the cart version is 2

    Scenario: Stale version returns optimistic lock failed
        Given the cart for user "u1" is now at version 2 (updated by another request)
        When I update quantity of product "p1" to 3 in cart for user "u1" with expected version 1
        Then I will get this error "cart: falha no bloqueio otimista"