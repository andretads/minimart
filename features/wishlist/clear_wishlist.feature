Feature: ClearWishlist removes all items from the wishlist

    All possible scenarios are:
        - Error path: Empty user ID is rejected
        - Happy path: All items removed
        - Happy path: Clearing an already empty wishlist succeeds
        - Happy path: Clearing a missing wishlist succeeds

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty user ID is rejected
        When I clear the wishlist for user ""
        Then I will get this error "wishlist: user_id é obrigatório"

    Scenario: All items removed
        Given I have a wishlist for user "user-1" with items "prod-1", "prod-2"
        When I clear the wishlist for user "user-1"
        Then I will have an empty wishlist for user "user-1"

    Scenario: Clearing an already empty wishlist succeeds
        Given I have an empty wishlist for user "user-1"
        When I clear the wishlist for user "user-1"
        Then I will have an empty wishlist for user "user-1"

    Scenario: Clearing a missing wishlist succeeds
        Given I have a customer for user "user-1"
        And I have no wishlist for user "user-1"
        When I clear the wishlist for user "user-1"
        Then no error will be returned