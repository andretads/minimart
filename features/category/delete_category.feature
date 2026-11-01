Feature: DeleteCategory deletes a category by ID

    Uses optimistic locking with exact If-Match version.

    All possible scenarios are:
        - Error path: Unknown ID is rejected
        - Error path: Missing If-Match is rejected
        - Error path: Malformed If-Match is rejected
        - Error path: Wildcard If-Match is rejected
        - Error path: Stale version (concurrent delete) is rejected
        - Happy path: Category deleted with correct If-Match
        - Happy path: Deleted category no longer retrievable
        - Happy path: Deleted category no longer appears in list

    Background: Clear the storage
        Given I have empty storage

    Scenario: Unknown ID is rejected
        When I delete category "unknown" with If-Match "1"
        Then I will get this error "category: não encontrada"

    Scenario: Missing If-Match is rejected
        Given I have a category with name "Electronics"
        When I delete category "p1" without If-Match
        Then I will get this error "cabeçalho If-Match obrigatório"

    Scenario: Malformed If-Match is rejected
        Given I have a category with name "Electronics"
        When I delete category "p1" with If-Match "not-a-valid-tag"
        Then I will get this error "cabeçalho If-Match inválido"

    Scenario: Wildcard If-Match is rejected
        Given I have a category with name "Electronics"
        When I delete category "p1" with If-Match "*"
        Then I will get this error "cabeçalho If-Match inválido"

    Scenario: Stale version is rejected
        Given I have a category with name "Electronics"
        And I hold the ETag for version 1
        And another writer deleted the category with id "p1"
        When I delete category "p1" with If-Match tag for version 1
        Then I will get this error "category: conflito de bloqueio"

    Scenario: Category deleted with correct If-Match
        Given I have a category with name "Electronics" and icon "📱"
        When I delete category "p1" with If-Match tag for version 1
        Then I will get a 204 no content response

    Scenario: Deleted category no longer retrievable
        Given I have a category with name "Electronics"
        When I delete category "p1" with If-Match tag for version 1
        And I get the category by ID "p1"
        Then I will get this error "category: não encontrada"

    Scenario: Deleted category no longer appears in list
        Given I have a category with name "Electronics"
        And I have a category with name "Books"
        When I delete category "Electronics" with If-Match tag for version 1
        And I list the categories
        Then I will get a list with 1 category
        And the category name is "Books"