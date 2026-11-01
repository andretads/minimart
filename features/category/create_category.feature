Feature: CreateCategory with a new category record

    All possible scenarios are:
        - Error path: Blank name is rejected
        - Error path: Name too long is rejected
        - Error path: Icon too long is rejected
        - Error path: Duplicate name is rejected (case-insensitive)
        - Happy path: Category got created with no problems
        - Happy path: Category without icon gets created
        - Happy path: Name and icon are trimmed on creation

    Background: Clear the storage
        Given I have empty storage

    Scenario: Blank name is rejected
        When I create a category with name "   " and icon "📱"
        Then I will get this error "category: nome é obrigatório"

    Scenario: Name too long is rejected
        When I create a category with name "a" repeated 101 times and icon "📱"
        Then I will get this error "category: nome muito longo"

    Scenario: Icon too long is rejected
        When I create a category with name "Electronics" and icon "a" repeated 501 times
        Then I will get this error "category: ícone muito longo"

    Scenario: Duplicate name is rejected case-insensitively
        Given I have a category with name "Electronics"
        When I create a category with name "electronics" and icon "📱"
        Then I will get this error "category: já existe"

    Scenario: Category got created with no problems
        When I create a category with name "Electronics" and icon "📱"
        Then I will have a stored category with a generated id, name "Electronics", icon "📱", version 1 and a created_at timestamp

    Scenario: Category without icon gets created
        When I create a category with name "Books" and icon ""
        Then I will have a stored category with a generated id, name "Books", icon "", version 1 and a created_at timestamp

    Scenario: Name and icon are trimmed on creation
        When I create a category with name "  Electronics  " and icon "  📱  "
        Then I will have a stored category with a generated id, name "Electronics", icon "📱", version 1 and a created_at timestamp