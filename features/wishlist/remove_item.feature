Feature: RemoveItem removes a product from the wishlist

    All possible scenarios are:
        - Error path: Empty user ID is rejected
        - Error path: Empty product ID is rejected
        - Error path: Wishlist not found
        - Error path: Product not in wishlist
        - Happy path: Product removed
        - Happy path: Other items remain

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty user ID is rejected
        When I remove product "prod-1" from the wishlist for user ""
        Then I will get this error "wishlist: user_id é obrigatório"

    Scenario: Empty product ID is rejected
        When I remove product "" from the wishlist for user "user-1"
        Then I will get this error "wishlist: product_id é obrigatório"

    Scenario: Wishlist not found
        Given I have a customer for user "user-1"
        And I have no wishlist for user "user-1"
        When I remove product "prod-1" from the wishlist for user "user-1"
        Then I will get this error "wishlist: lista de desejos não encontrada"

    Scenario: Product not in wishlist
        Given I have a wishlist for user "user-1" with items "prod-1"
        When I remove product "prod-2" from the wishlist for user "user-1"
        Then I will get this error "wishlist: produto não está na lista de desejos"

    Scenario: Product removed
        Given I have a wishlist for user "user-1" with items "prod-1", "prod-2"
        When I remove product "prod-1" from the wishlist for user "user-1"
        Then I will have a wishlist with items "prod-2"

    Scenario: Other items remain
        Given I have a wishlist for user "user-1" with items "prod-1", "prod-2", "prod-3"
        When I remove product "prod-2" from the wishlist for user "user-1"
        Then I will have a wishlist with 2 items