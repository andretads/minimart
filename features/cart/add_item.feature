Feature: AddItem adds an item to the cart or increments quantity if already present

    Validates the user, ensures a customer exists, fetches the product name/price
    from the catalog contract (never the client), and updates the cart.
    Uses optimistic locking on the cart version.

    All possible scenarios are:
        - Error path: Blank user_id is rejected
        - Error path: Non-positive quantity is rejected
        - Error path: Unknown product returns catalog: product not found
        - Happy path: Creates cart with catalog price/name snapshot on first add
        - Happy path: Second add of same product increments quantity
        - Edge path: Add to consumed cart returns cart not active

    Background: Clear the storage
        Given I have empty cart storage
        And I have a catalog with product "p1" named "Widget" priced 500 cents

    Scenario: Blank user_id is rejected
        When I add 1 unit of product "p1" to cart for user ""
        Then I will get this error "cart: parâmetro de consulta user_id é obrigatório"

    Scenario: Non-positive quantity is rejected
        When I add 0 units of product "p1" to cart for user "u1"
        Then I will get this error "cart: quantidade inválida"

    Scenario: Unknown product returns catalog error
        When I add 1 unit of product "nope" to cart for user "u1"
        Then I will get this error "catalog: produto não encontrado"

    Scenario: Creates cart with catalog price/name snapshot on first add
        When I add 2 units of product "p1" to cart for user "u1"
        Then I will have a cart for user "u1" with 1 item: "Widget" x2 at 500 cents each
        And the cart status is "active"

    Scenario: Second add of same product increments quantity
        When I add 1 unit of product "p1" to cart for user "u1"
        And I add 3 units of product "p1" to cart for user "u1"
        Then I will have a cart for user "u1" with 1 item: "Widget" x4 at 500 cents each

    Scenario: Add to consumed cart returns cart not active
        Given I have a consumed cart for user "u1"
        When I add 1 unit of product "p1" to cart for user "u1"
        Then I will get this error "cart: carrinho não ativo"