Feature: EnsureAdmins grants the admin role to listed usernames

    Called at server boot. Usernames not found in the accounts table are skipped.
    Already-admin accounts are tolerated. Empty list is a no-op.

    All possible scenarios are:
        - Edge path: Empty list is a no-op
        - Happy path: Listed usernames get admin role
        - Edge path: Already-admin is tolerated
        - Edge path: Unknown usernames are skipped

    Background: Clear the storage
        Given I have empty auth storage
        And the system has seeded roles "customer", "support", "admin"

    Scenario: Empty list is a no-op
        When I ensure admins for usernames []
        Then no error is returned
        And no roles are assigned

    Scenario: Listed usernames get admin role
        Given I have accounts with usernames "alice", "bob", "carol"
        When I ensure admins for usernames ["alice", "bob"]
        Then account "alice" has role "admin"
        And account "bob" has role "admin"
        And account "carol" does not have role "admin"

    Scenario: Already-admin is tolerated
        Given I have an account "alice" with role "admin"
        When I ensure admins for usernames ["alice"]
        Then no error is returned
        And account "alice" has role "admin"

    Scenario: Unknown usernames are skipped
        Given I have an account "alice"
        When I ensure admins for usernames ["alice", "unknown-user"]
        Then no error is returned
        And account "alice" has role "admin"
        And no account is created for "unknown-user"