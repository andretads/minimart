Feature: GetReviews for a product

    The route requires the "catalog:read" permission, which the customer,
    support and admin roles hold. The public read path is the showcase
    endpoint, which needs no permission at all.

    All possible scenarios are:
        - Error path: Product does not exist
        - Happy path: Approved reviews got returned
        - Happy path: Pending reviews are not returned
        - Authorization path: The route is governed by the catalog:read permission

    Background: Clear the storage
        Given I have empty storage

    Scenario: Product does not exist
        Given I have no products stored
        When I get reviews for product "missing"
        Then I will get this error "produto não encontrado"

    Scenario: Approved reviews got returned
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        And I have an approved review with id "r1", product "p1", customer "c1", description "Good", rating 5
        And I have an approved review with id "r2", product "p1", customer "c2", description "Bad", rating 2
        When I get reviews for product "p1"
        Then I will get reviews with id "r1", "r2" sorted by created_at descending

    Scenario: Pending reviews are not returned
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        And I have an approved review with id "r1", product "p1", customer "c1", description "Good", rating 5
        And I have a pending review with id "r3", product "p1", customer "c3", description "Pending", rating 3
        When I get reviews for product "p1"
        Then I will get only the approved review with id "r1"

    Scenario: The route is governed by the catalog:read permission
        Given the route "GET /v1/products/{id}/reviews" requires the permission "catalog:read"
        Then the route "GET /v1/products/{id}/reviews" is not public