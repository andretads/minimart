Feature: Product images

    A product may have many images. Images are ordered by position; the lowest
    position is the primary image. Adding or removing an image bumps the owning
    product's version so optimistic-lock clients observe the change.

    All possible scenarios are:
        - Error path: Product does not exist
        - Error path: Image is missing
        - Error path: Unsupported image type
        - Error path: Image is too large
        - Error path: Image does not belong to the product
        - Happy path: First image is added at position 0
        - Happy path: Additional images are appended
        - Happy path: Product response includes its images
        - Happy path: Image is deleted

    Background: Clear the storage
        Given I have empty storage

    Scenario: Product does not exist
        Given I have no products stored
        When I add an image to product "missing"
        Then I will get this error "produto não encontrado"

    Scenario: Image is missing
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add an image to product "p1" without a file
        Then I will get this error "imagem do produto é obrigatória"

    Scenario: Unsupported image type
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add a text file as an image to product "p1"
        Then I will get this error "formato de imagem inválido, deve ser uma imagem"

    Scenario: Image is too large
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add an image larger than the configured limit to product "p1"
        Then I will get this error "imagem excede o tamanho máximo permitido"

    Scenario: Image does not belong to the product
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        And I have a product with id "p2", name "Gadget", price 500 cents and stock 3
        And product "p2" has an image
        When I delete that image through product "p1"
        Then I will get this error "imagem do produto não encontrada"

    Scenario: First image is added at position 0
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        When I add an image to product "p1"
        Then the image has position 0 and product "p1" has version 2

    Scenario: Additional images are appended
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        And product "p1" has an image at position 0
        When I add another image to product "p1"
        Then the new image has position 1 and product "p1" has version 3

    Scenario: Product response includes its images
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        And product "p1" has an image at position 0
        When I get the product with id "p1"
        Then the response includes the product's images ordered by position

    Scenario: Image is deleted
        Given I have a product with id "p1", name "Widget", price 500 cents and stock 3
        And product "p1" has an image at position 0
        When I delete the image from product "p1"
        Then the image is removed and product "p1" has version 3
