Feature: ListOrders by user

    Orders are listed newest first for a single user. The list is paginated with a
    1-based page number, a page size capped at 100, and total/total-pages metadata.
    Only orders owned by the requested user are returned.

    All possible scenarios are:
        - Error path: user id is missing
        - Error path: page is invalid
        - Error path: page size is invalid
        - Happy path: user has no orders
        - Happy path: user's orders are listed newest first
        - Happy path: pagination returns the requested window and metadata
        - Edge path: another user's orders are never returned

    Background: Clear the storage
        Given I have empty storage

    Scenario: user id is missing
        When I list orders without a user id
        Then I will get this error "id do usuário é obrigatório"

    Scenario: page is invalid
        When I list orders for user "u1" with page "0"
        Then I will get this error "página deve ser maior que zero"

    Scenario: page size is invalid
        When I list orders for user "u1" with page size "101"
        Then I will get this error "tamanho da página deve estar entre 1 e 100"

    Scenario: user has no orders
        When I list orders for user "u1"
        Then I will get an empty order list

    Scenario: user's orders are listed newest first
        Given I have a stored order with id "o1" for user "u1" with a total of 500 cents
        And I have a stored order with id "o2" for user "u1" with a total of 750 cents
        When I list orders for user "u1"
        Then I will get 2 orders with the newest order first

    Scenario: pagination returns the requested window and metadata
        Given I have 3 stored orders for user "u1"
        When I list orders for user "u1" with page "2" and page size "2"
        Then I will get 1 order and 2 total pages

    Scenario: another user's orders are never returned
        Given I have a stored order with id "o1" for user "u1" with a total of 500 cents
        And I have a stored order with id "o2" for user "u2" with a total of 750 cents
        When I list orders for user "u1"
        Then I will get 1 order owned by user "u1"

    Scenario: Placed orders in a list expose their auto-cancel deadline
        Given I have a placed order with id "o1" for user "u1"
        When I list orders for user "u1"
        Then each placed order will include an "auto_cancel_at" timestamp
