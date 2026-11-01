Feature: GetReviewSummary for a product

    All possible scenarios are:
        - Error path: Product does not exist
        - Happy path: Review summary got returned

    Background: Clear the storage
        Given I have empty storage

    Scenario: Product does not exist
        Given I have no products stored
        When I get review summary for product "missing"
        Then I will get this error "produto não encontrado"

    Scenario: Review summary got returned
        Given I have a product with id "p1", name "Widget", price 500 cents, stock 3, total_rating 20 and total_reviews 5
        When I get review summary for product "p1"
        Then I will get total_rating 20 and total_reviews 5