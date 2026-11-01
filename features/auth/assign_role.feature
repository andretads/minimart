Feature: AssignRole assigns a role to an account

    Idempotent: assigning an already-assigned role returns nil (no error).
    The role must exist in the system (seeded at migration).

    All possible scenarios are:
        - Happy path: Role assigned to account
        - Edge path: Assigning twice is idempotent
        - Error path: Unknown role returns role not found

    Background: Clear the storage
        Given I have empty auth storage
        And the system has seeded roles "customer", "support", "admin"

    Scenario: Role assigned to account
        Given I have an account with ID "acc-1"
        When I assign role "support" to account "acc-1"
        Then the account "acc-1" has role "support"

    Scenario: Assigning twice is idempotent
        Given I have an account with ID "acc-1"
        When I assign role "support" to account "acc-1"
        And I assign role "support" to account "acc-1" again
        Then no error is returned
        And the account "acc-1" has role "support" exactly once

    Scenario: Unknown role returns role not found
        Given I have an account with ID "acc-1"
        When I assign role "unknown-role" to account "acc-1"
        Then I will get this error "auth: role not found"