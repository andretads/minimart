Feature: ListProducts returns a paginated list of products, newest-first

    The showcase is a public read-only view of products. It uses its own reader
    queries (CQRS-style) over the shared products table. Pagination is keyset
    (cursor-based): page_size default 20, max 100. Results are ordered by
    created_at DESC, id DESC. Pass `cursor` from a previous response's
    meta.next_cursor to fetch the next page. No total/total_pages are returned.

    All possible scenarios are:
        - Happy path: Empty catalog returns an empty list
        - Happy path: Seeded catalog returns products newest-first with cursor metadata
        - Happy path: Page_size limits number of items returned
        - Happy path: Cursor advances to the next page
        - Happy path: Last page has has_more false and no next_cursor
        - Error path: Invalid page_size (< 1 or > 100) returns 400
        - Error path: Non-integer page_size returns 400
        - Error path: Invalid cursor returns 400

    Background:
        Given I have a running showcase service
        And the catalog is empty

    Scenario: Empty catalog returns an empty paginated list
        When I GET "/v1/showcase/products"
        Then the response status is 200
        And the response body has success true
        And the data array is empty
        And meta.has_more is false
        And meta.next_cursor is absent

    Scenario: Seeded catalog returns products newest-first with cursor metadata
        Given I have products "Product A" (1000), "Product B" (2000), "Product C" (3000) in the catalog
        When I GET "/v1/showcase/products"
        Then the response status is 200
        And the response body has success true
        And the data array has 3 items
        And the first item is "Product C" (newest first)
        And each item has stock, total_rating and total_reviews
        And meta.page_size is 20
        And meta.has_more is false

    Scenario: Products with images expose the primary image URL
        Given I have a product with two images ordered by position
        When I GET "/v1/showcase/products"
        Then the response status is 200
        And the first item image_url equals the first image url
        And the first item images field is absent

    Scenario: Page_size limits number of items returned
        Given I have products "Product A" (1000), "Product B" (2000), "Product C" (3000) in the catalog
        When I GET "/v1/showcase/products?page_size=2"
        Then the response status is 200
        And the data array has 2 items
        And meta.page_size is 2
        And meta.has_more is true
        And meta.next_cursor is present

    Scenario: Cursor advances to the next page
        Given I have products "Product A" (1000), "Product B" (2000), "Product C" (3000) in the catalog
        When I GET "/v1/showcase/products?page_size=2"
        Then I store meta.next_cursor as "cursor"
        When I GET "/v1/showcase/products?page_size=2&cursor={{cursor}}"
        Then the response status is 200
        And the data array has 1 item
        And the item is "Product A" (oldest)
        And meta.has_more is false
        And meta.next_cursor is absent

    Scenario: Invalid page_size returns 400
        When I GET "/v1/showcase/products?page_size=0"
        Then the response status is 400
        And I GET "/v1/showcase/products?page_size=101"
        Then the response status is 400

    Scenario: Non-integer page_size returns 400
        When I GET "/v1/showcase/products?page_size=xyz"
        Then the response status is 400

    Scenario: Invalid cursor returns 400
        When I GET "/v1/showcase/products?cursor=not-a-cursor"
        Then the response status is 400
