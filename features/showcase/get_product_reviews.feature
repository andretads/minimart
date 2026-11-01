Feature: ListProductReviews returns the approved reviews of a product

    The showcase provides a public reviews endpoint for a single product.
    Only approved reviews are returned, newest first. No authentication
    required.

    All possible scenarios are:
        - Happy path: Product with approved reviews returns them newest first
        - Happy path: Pending/rejected reviews are excluded
        - Happy path: Product without reviews returns an empty array
        - Error path: Repository failure returns 500

    Background:
        Given I have a running showcase service

    Scenario: Product with approved reviews returns them newest first
        Given I have product "p1"
        And the product has an approved review from customer "c1" with rating 5
        And the product has an approved review from customer "c2" with rating 3
        When I GET "/v1/showcase/products/p1/reviews"
        Then the response status is 200
        And the response body has success true
        And the data array has 2 items
        And the first item is the newest review
        And each item has id, product_id, customer_id, description, rating, status and created_at

    Scenario: Pending reviews are excluded
        Given I have product "p1"
        And the product has an approved review from customer "c1"
        And the product has a pending review from customer "c2"
        When I GET "/v1/showcase/products/p1/reviews"
        Then the response status is 200
        And the data array has 1 item
        And the item customer_id is "c1"

    Scenario: Product without reviews returns an empty array
        Given I have product "p1"
        When I GET "/v1/showcase/products/p1/reviews"
        Then the response status is 200
        And the response body has success true
        And the data array is empty

    Scenario: Repository failure returns 500
        Given the showcase repository fails
        When I GET "/v1/showcase/products/p1/reviews"
        Then the response status is 500
        And the response body has success false
        And the error.message is "erro interno"
