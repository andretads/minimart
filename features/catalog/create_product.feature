Feature: CreateProduct with a new catalog entry

    All possible scenarios are:
        - Error path: Blank name is rejected
        - Error path: Non-positive price is rejected
        - Error path: Negative stock is rejected
        - Error path: Storage failure
        - Error path: Empty SKU is rejected
        - Error path: Duplicate SKU is rejected
        - Happy path: Product got created with no problems
        - Happy path: Product got created with description
        - Happy path: Product got created with SKU
        - Happy path: Product got created with details

    Background: Clear the storage
        Given I have empty storage

    Scenario: Blank name is rejected
        When I create a product with name "   ", price 500 cents and stock 3
        Then I will get this error "nome do produto não pode ser vazio"

    Scenario: Non-positive price is rejected
        When I create a product with name "Widget", price 0 cents and stock 3
        Then I will get this error "preço do produto deve ser maior que zero"

    Scenario: Negative stock is rejected
        When I create a product with name "Widget", price 500 cents and stock -1
        Then I will get this error "estoque do produto não pode ser negativo"

    Scenario: Storage failure
        Given the storage is failing
        When I create a product with name "Widget", price 500 cents and stock 3
        Then I will get this error "disk on fire"

    Scenario: Empty SKU is rejected
        When I create a product with name "Widget", price 500 cents, stock 3 and SKU "   "
        Then I will get this error "SKU do produto não pode ser vazio"

    Scenario: Duplicate SKU is rejected
        Given I have a product with SKU "SKU-123"
        When I create a product with name "Widget", price 500 cents, stock 3 and SKU "SKU-123"
        Then I will get this error "SKU do produto já existe"

    Scenario: Product got created with no problems
        When I create a product with name "Widget", price 500 cents and stock 3
        Then I will have a stored product with a generated id, name "Widget", price 500 cents, stock 3 and a created_at timestamp

    Scenario: Product name is trimmed on creation
        When I create a product with name "  Widget  ", price 500 cents and stock 3
        Then I will have a stored product with a generated id, name "Widget", price 500 cents, stock 3 and a created_at timestamp

    Scenario: Product got created with description
        When I create a product with name "Widget", price 500 cents, stock 3 and description "A great widget"
        Then I will have a stored product with a generated id, name "Widget", price 500 cents, stock 3, description "A great widget" and a created_at timestamp

    Scenario: Product got created with SKU
        When I create a product with name "Widget", price 500 cents, stock 3 and SKU "SKU-123"
        Then I will have a stored product with a generated id, name "Widget", price 500 cents, stock 3, SKU "SKU-123" and a created_at timestamp

    Scenario: Product got created with details
        When I create a product with name "Widget", price 500 cents, stock 3 and details [{"name": "Color", "value": "Red"}, {"name": "Size", "value": "Large"}]
        Then I will have a stored product with a generated id, name "Widget", price 500 cents, stock 3, details [{"name": "Color", "value": "Red"}, {"name": "Size", "value": "Large"}] and a created_at timestamp