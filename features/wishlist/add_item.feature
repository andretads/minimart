Feature: AddItem adds a product to the wishlist

    All possible scenarios are:
        - Error path: Empty user ID is rejected
        - Error path: Empty product ID is rejected
        - Error path: Product does not exist
        - Error path: Product already in wishlist
        - Happy path: Product added to existing wishlist
        - Happy path: Wishlist created on first add
        - Happy path: Duplicate concurrent create resolves to one wishlist

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty user ID is rejected
        When I add product "prod-1" to the wishlist for user ""
        Then I will get this error "wishlist: user_id é obrigatório"

    Scenario: Empty product ID is rejected
        When I add product "" to the wishlist for user "user-1"
        Then I will get this error "wishlist: product_id é obrigatório"

    Scenario: Product does not exist
        Given I have a customer for user "user-1"
        When I add product "missing-prod" to the wishlist for user "user-1"
        Then I will get an error

    Scenario: Product already in wishlist
        Given I have a wishlist for user "user-1" with items "prod-1"
        When I add product "prod-1" to the wishlist for user "user-1"
        Then I will get this error "wishlist: produto já está na lista de desejos"

    Scenario: Product added to existing wishlist
        Given I have a wishlist for user "user-1" with items "prod-1"
        When I add product "prod-2" to the wishlist for user "user-1"
        Then I will have a wishlist with items "prod-1", "prod-2"

    Scenario: Wishlist created on first add
        Given I have a customer for user "user-1"
        And I have no wishlist for user "user-1"
        When I add product "prod-1" to the wishlist for user "user-1"
        Then I will have a wishlist with items "prod-1"