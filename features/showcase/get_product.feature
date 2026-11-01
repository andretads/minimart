Feature: GetProduct returns a single product by ID

    The showcase provides a public detail endpoint for a single product.
    No authentication required.

    All possible scenarios are:
        - Happy path: Get product by ID returns product details
        - Error path: Get product not found returns 404

    Background:
        Given I have a running showcase service

    Scenario: Get product by ID returns product details
        Given I have product "p1" with name "Product A" and price 1000
        And the product has stock 5, sku "SKU-1" and description "A fine product"
        When I GET "/v1/showcase/products/p1"
        Then the response status is 200
        And the response body has success true
        And the data.id is "p1"
        And the data.name is "Product A"
        And the data.price_cents is 1000
        And the data.stock is 5
        And the data.sku is "SKU-1"
        And the data.description is "A fine product"
        And the data.details is an array
        And the data.total_rating is 0
        And the data.total_reviews is 0
        And the data.images is an array

    Scenario: Get product by ID includes images with presigned URLs
        Given I have product "p1" with two images ordered by position
        When I GET "/v1/showcase/products/p1"
        Then the response status is 200
        And the data.images has 2 items ordered by position
        And each image has a non-empty url
        And the data.image_url equals the first image url

    Scenario: Get product not found returns 404
        When I GET "/v1/showcase/products/nonexistent"
        Then the response status is 404
        And the response body has success false
        And the error.message is "produto não encontrado"