Feature: AddReview for a product

    The route requires the "reviews:write" permission, which the customer and
    admin roles hold. The customer is taken from the authenticated session, so
    a caller cannot post a review on behalf of another customer.

    All possible scenarios are:
        - Error path: Product does not exist
        - Error path: Invalid rating (too low)
        - Error path: Invalid rating (too high)
        - Error path: Empty description is rejected
        - Error path: Missing customer is rejected
        - Error path: Unauthenticated request is rejected
        - Error path: Storage failure
        - Happy path: Review got added with no problems
        - Happy path: Review starts as pending for moderation
        - Happy path: Review is attributed to the authenticated customer
        - Moderation path: Review is approved
        - Moderation path: Review is rejected
        - Moderation path: Review needs human review

    Background: Clear the storage
        Given I have empty storage

    Scenario: Product does not exist
        Given I have no products stored
        When I add a review for product "missing" by customer "cust-1" with description "Good" and rating 4
        Then I will get this error "produto não encontrado"

    Scenario: Invalid rating too low is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" by customer "cust-1" with description "Bad" and rating 0
        Then I will get this error "avaliação deve ser entre 1 e 5"

    Scenario: Invalid rating too high is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" by customer "cust-1" with description "Bad" and rating 6
        Then I will get this error "avaliação deve ser entre 1 e 5"

    Scenario: Empty description is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" by customer "cust-1" with description "   " and rating 5
        Then I will get this error "nome do produto não pode ser vazio"

    Scenario: Missing customer is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" by customer "" with description "Good" and rating 4
        Then I will get this error "id do cliente é obrigatório"

    Scenario: Unauthenticated request is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" without a session
        Then I will get this error "usuário não encontrado no contexto"

    Scenario: Storage failure
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        And the storage is failing on review save
        When I add a review for product "p1" by customer "cust-1" with description "Good" and rating 4
        Then I will get this error "disk on fire"

    Scenario: Review got added with no problems
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" by customer "cust-1" with description "Great product!" and rating 5
        Then I will have a review with a generated id, product "p1", customer "cust-1", description "Great product!", rating 5 and a created_at timestamp

    Scenario: Review starts as pending for moderation
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" by customer "cust-1" with description "Great product!" and rating 5
        Then the review will have status "pending"

    Scenario: Review is attributed to the authenticated customer
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" as the signed in customer "cust-1" with description "Great product!" and rating 5
        Then the review will have customer "cust-1"

    Scenario: Review is approved by moderation
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" by customer "cust-1" with description "Great product!" and rating 5
        And the moderation service is enabled and approves the review
        Then the review will have status "approved"
        And the review will have moderation metadata

    Scenario: Review is rejected by moderation
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" by customer "cust-1" with description "This product is f***ing garbage!" and rating 1
        And the moderation service is enabled and rejects the review
        Then the review will have status "rejected"
        And the review will not be visible in public listings

    Scenario: Review needs human review
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a review for product "p1" by customer "cust-1" with description "Hmm, not sure about this one" and rating 3
        And the moderation service is enabled and cannot confidently decide
        Then the review will have status "needs_review"
        And the review will not be visible in public listings