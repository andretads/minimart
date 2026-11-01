Feature: GetWishlist retrieves or creates a wishlist for a user

    All possible scenarios are:
        - Error path: Empty user ID is rejected
        - Happy path: Existing wishlist returned
        - Happy path: Missing wishlist is created empty
        - Happy path: Concurrent create returns the existing wishlist

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty user ID is rejected
        When I get the wishlist for user ""
        Then I will get this error "wishlist: user_id é obrigatório"

    Scenario: Existing wishlist returned
        Given I have a wishlist for user "user-1" with items "prod-1", "prod-2"
        When I get the wishlist for user "user-1"
        Then I will have a wishlist with customer for "user-1" and items "prod-1", "prod-2"

    Scenario: Missing wishlist is created empty
        Given I have a customer for user "user-1"
        And I have no wishlist for user "user-1"
        When I get the wishlist for user "user-1"
        Then I will have an empty wishlist for user "user-1"

    Scenario: Concurrent create returns the existing wishlist
        Given two concurrent get-wishlist calls for user "user-1"
        When both complete
        Then both will return the same wishlist