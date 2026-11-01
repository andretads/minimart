Feature: GetProduct by id

    All possible scenarios are:
        - Error path: Product does not exist
        - Happy path: Product got returned with no problems
        - Happy path: Unchanged product answers 304 with If-None-Match

    Background: Clear the storage
        Given I have empty storage

    Scenario: Product does not exist
        Given I have no products stored
        When I get the product with id "missing"
        Then I will get this error "produto não encontrado"

    Scenario: Product got returned with no problems
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I get the product with id "p1"
        Then I will get a product with id "p1", name "Widget", price 500 cents, stock 3, version 1 and a created_at timestamp
        And the response carries the ETag for version 1

    Scenario: Product with description returned
        Given I have a product with id "p1", name "Widget", price 500 cents, stock 3 and description "A great widget"
        When I get the product with id "p1"
        Then I will get a product with id "p1", name "Widget", price 500 cents, stock 3, version 1, description "A great widget" and a created_at timestamp
        And the response carries the ETag for version 1

    Scenario: Product with SKU returned
        Given I have a product with id "p1", name "Widget", price 500 cents, stock 3 and SKU "SKU-123"
        When I get the product with id "p1"
        Then I will get a product with id "p1", name "Widget", price 500 cents, stock 3, version 1, SKU "SKU-123" and a created_at timestamp
        And the response carries the ETag for version 1

    Scenario: Product with details returned
        Given I have a product with id "p1", name "Widget", price 500 cents, stock 3 and details [{"name": "Color", "value": "Red"}]
        When I get the product with id "p1"
        Then I will get a product with id "p1", name "Widget", price 500 cents, stock 3, version 1, details [{"name": "Color", "value": "Red"}] and a created_at timestamp
        And the response carries the ETag for version 1

    Scenario: Unchanged product answers 304
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I get the product with id "p1" sending the ETag for version 1 as If-None-Match
        Then I will get a not modified response with no body