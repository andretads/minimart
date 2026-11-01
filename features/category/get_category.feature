Feature: GetCategory retrieves category by CategoryID

    All possible scenarios are:
        - Error path: Unknown ID is rejected
        - Happy path: Category retrieved by ID
        - Happy path: Unchanged category answers 304 with If-None-Match

    Background: Clear the storage
        Given I have empty storage

    Scenario: Unknown ID is rejected
        When I get a category by ID "unknown"
        Then I will get this error "category: não encontrada"

    Scenario: Category retrieved by ID
        Given I have a category with name "Electronics" and icon "📱"
        When I get the category by its ID
        Then I will have a category with name "Electronics", icon "📱", version 1 and a created_at timestamp

    Scenario: Unchanged category answers 304
        Given I have a category with name "Electronics" and icon "📱"
        When I get the category by its ID sending the ETag for version 1 as If-None-Match
        Then I will get a not modified response with no body