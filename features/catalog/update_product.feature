Feature: UpdateProduct name, price, SKU, description, and details

    All possible scenarios are:
        - Error path: Product does not exist
        - Error path: Blank name is rejected
        - Error path: Non-positive price is rejected
        - Error path: Storage failure
        - Error path: Missing version
        - Error path: Stale version (concurrent update)
        - Error path: Stale ETag in If-Match
        - Error path: Empty SKU is rejected
        - Happy path: Product got updated with no problems
        - Happy path: Product got updated with description
        - Happy path: Product got updated with SKU
        - Happy path: Product got updated with details

    Background: Clear the storage
        Given I have empty storage

    Scenario: Product does not exist
        Given I have no products stored
        When I update the product with id "missing" to name "Gadget" and price 750 cents
        Then I will get this error "produto não encontrado"

    Scenario: Blank name is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "   " and price 750 cents
        Then I will get this error "nome do produto não pode ser vazio"

    Scenario: Non-positive price is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "Gadget" and price 0 cents
        Then I will get this error "preço do produto deve ser maior que zero"

    Scenario: Storage failure
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        And the storage is failing on update
        When I update the product with id "p1" to name "Gadget" and price 750 cents
        Then I will get this error "disk on fire"

    Scenario: Missing version is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "Gadget" and price 750 cents without a version
        Then I will get this error "versão do produto deve ser maior que zero"

    Scenario: Stale version is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        And I hold a copy of the product with id "p1" at its current version
        And another writer updated the product with id "p1"
        When I update my stale copy of the product with id "p1" to name "Gadget" and price 750 cents
        Then I will get this error "conflito de bloqueio do produto"

    Scenario: Stale ETag is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "Gadget" and price 750 cents with If-Match tag for version 99
        Then I will get this error "conflito de bloqueio do produto"

    Scenario: Empty SKU is rejected
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "Gadget", price 750 cents and SKU "   "
        Then I will get this error "SKU do produto não pode ser vazio"

    Scenario: Product got updated with no problems
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "Gadget" and price 750 cents
        Then I will have a stored product with id "p1", name "Gadget", price 750 cents, stock 3 and version 2
        And the created_at of product "p1" is unchanged

    Scenario: Product is updated using the ETag from GET
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "Gadget" and price 750 cents with If-Match tag for version 1
        Then I will have a stored product with id "p1", name "Gadget", price 750 cents, stock 3 and version 2
        And the created_at of product "p1" is unchanged

    Scenario: Product name is trimmed on update
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "  Gadget  " and price 750 cents
        Then I will have a stored product with id "p1", name "Gadget", price 750 cents, stock 3 and version 2
        And the created_at of product "p1" is unchanged

    Scenario: Product got updated with description
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "Gadget", price 750 cents and description "Updated description"
        Then I will have a stored product with id "p1", name "Gadget", price 750 cents, stock 3, description "Updated description" and version 2
        And the created_at of product "p1" is unchanged

    Scenario: Product got updated with SKU
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "Gadget", price 750 cents and SKU "NEW-SKU"
        Then I will have a stored product with id "p1", name "Gadget", price 750 cents, stock 3, SKU "NEW-SKU" and version 2
        And the created_at of product "p1" is unchanged

    Scenario: Product got updated with details
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I update the product with id "p1" to name "Gadget", price 750 cents and details [{"name": "Color", "value": "Blue"}]
        Then I will have a stored product with id "p1", name "Gadget", price 750 cents, stock 3, details [{"name": "Color", "value": "Blue"}] and version 2
        And the created_at of product "p1" is unchanged