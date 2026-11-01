Feature: UpdateCategory updates category name and icon

    All possible scenarios are:
        - Error path: Category does not exist
        - Error path: Blank name is rejected
        - Error path: Name too long is rejected
        - Error path: Icon too long is rejected
        - Error path: Duplicate name is rejected (case-insensitive)
        - Error path: Missing version is rejected
        - Error path: Stale version (concurrent update) is rejected
        - Error path: Stale ETag in If-Match is rejected
        - Happy path: Category got updated with no problems
        - Happy path: Category is updated using the ETag from GET
        - Happy path: Name and icon are trimmed on update
        - Happy path: Created_at of category is unchanged

    Background: Clear the storage
        Given I have empty storage

    Scenario: Category does not exist
        Given I have no categories stored
        When I update the category with id "missing" to name "Gadgets" and icon "🔧"
        Then I will get this error "category: não encontrada"

    Scenario: Blank name is rejected
        Given I have a category with name "Electronics" and icon "📱"
        When I update the category with id "p1" to name "   " and icon "🔧"
        Then I will get this error "category: nome é obrigatório"

    Scenario: Name too long is rejected
        Given I have a category with name "Electronics" and icon "📱"
        When I update the category with id "p1" to name "a" repeated 101 times and icon "🔧"
        Then I will get this error "category: nome muito longo"

    Scenario: Icon too long is rejected
        Given I have a category with name "Electronics" and icon "📱"
        When I update the category with id "p1" to name "Gadgets" and icon "a" repeated 501 times
        Then I will get this error "category: ícone muito longo"

    Scenario: Duplicate name is rejected
        Given I have a category with name "Electronics" and icon "📱"
        And I have a category with name "Books" and icon "📚"
        When I update the category "Books" to name "electronics" and icon "📖"
        Then I will get this error "category: já existe"

    Scenario: Missing version is rejected
        Given I have a category with name "Electronics" and icon "📱"
        When I update the category with id "p1" to name "Gadgets" and icon "🔧" without a version
        Then I will get this error "category: versão deve ser maior que zero"

    Scenario: Stale version is rejected
        Given I have a category with name "Electronics" and icon "📱"
        And I hold a copy of the category with id "p1" at its current version
        And another writer updated the category with id "p1"
        When I update my stale copy of the category with id "p1" to name "Gadgets" and icon "🔧"
        Then I will get this error "category: conflito de bloqueio"

    Scenario: Stale ETag is rejected
        Given I have a category with name "Electronics" and icon "📱"
        When I update the category with id "p1" to name "Gadgets" and icon "🔧" with If-Match tag for version 99
        Then I will get this error "category: conflito de bloqueio"

    Scenario: Category got updated with no problems
        Given I have a category with name "Electronics" and icon "📱"
        When I update the category with id "p1" to name "Gadgets" and icon "🔧"
        Then I will have a stored category with id "p1", name "Gadgets", icon "🔧", version 2
        And the created_at of category "p1" is unchanged

    Scenario: Category is updated using the ETag from GET
        Given I have a category with name "Electronics" and icon "📱"
        When I update the category with id "p1" to name "Gadgets" and icon "🔧" with If-Match tag for version 1
        Then I will have a stored category with id "p1", name "Gadgets", icon "🔧", version 2
        And the created_at of category "p1" is unchanged

    Scenario: Name and icon are trimmed on update
        Given I have a category with name "Electronics" and icon "📱"
        When I update the category with id "p1" to name "  Gadgets  " and icon "  🔧  "
        Then I will have a stored category with id "p1", name "Gadgets", icon "🔧", version 2
        And the created_at of category "p1" is unchanged