Feature: EnsureCustomer gets or creates a customer for a user ID

    This is the get-or-create use case used by other domains (orders, cart)
    when they need a customer record to exist. It is concurrency-safe:
    if two calls race, the second re-fetches the created customer.

    All possible scenarios are:
        - Error path: Empty user_id is rejected
        - Happy path: Existing customer returned unchanged
        - Happy path: Missing customer created with active status and empty display name
        - Edge path: Concurrent create for same user_id returns the same customer

    Background: Clear the storage
        Given I have empty customer storage

    Scenario: Empty user_id is rejected
        When I ensure a customer for user_id ""
        Then I will get this error "customer: user_id é obrigatório"

    Scenario: Existing customer returned unchanged
        Given I have a customer with user_id "user-1" and display_name "John"
        When I ensure a customer for user_id "user-1"
        Then I will have a customer with user_id "user-1", display_name "John", status "active"

    Scenario: Missing customer created with active status and empty display name
        When I ensure a customer for user_id "user-new"
        Then I will have a customer with user_id "user-new", display_name "", status "active", version 1

    Scenario: Concurrent create for same user_id returns the same customer
        When I ensure a customer for user_id "user-race"
        And I ensure a customer for user_id "user-race" again
        Then both calls return the same customer ID
        And exactly one customer is stored for user_id "user-race"