Feature: ListCategories returns all categories

    All possible scenarios are:
        - Happy path: Empty storage returns an empty list
        - Happy path: All stored categories are returned ordered by name

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty storage returns an empty list
        Given I have no categories stored
        When I list the categories
        Then I will get an empty list of categories

    Scenario: All stored categories are returned ordered by name
        Given I have a category with name "Electronics" and icon "📱"
        And I have a category with name "Books" and icon "📚"
        And I have a category with name "Clothing" and icon "👕"
        When I list the categories
        Then I will get a list with 3 categories
        And the first category name is "Books"
        And the second category name is "Clothing"
        And the third category name is "Electronics"