Feature: ListProducts for the whole catalog

    All possible scenarios are:
        - Happy path: Empty catalog returns an empty list
        - Happy path: All stored products are returned

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty catalog returns an empty list
        Given I have no products stored
        When I list the products
        Then I will get an empty list of products

    Scenario: All stored products are returned
        Given I have a product with id "p1", name "Widget", price 100 cents and stock 1
        And I have a product with id "p2", name "Gadget", price 100 cents and stock 1
        When I list the products
        Then I will get a list with 2 products
