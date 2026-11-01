Feature: PermissionsForAccount returns the permission IDs for an account

    Reads the RBAC tables: account -> roles -> permissions.
    Returns an empty list for unknown/role-less accounts (not an error).

    All possible scenarios are:
        - Happy path: Account with roles returns seeded permission IDs
        - Edge path: Unknown account returns empty list
        - Edge path: Role-less account returns empty list

    Background: Clear the storage
        Given I have empty auth storage
        And the system has seeded permissions:
            | permission         |
            | catalog:read       |
            | catalog:write      |
            | customers:read     |
            | customers:write    |
            | orders:read        |
            | orders:write       |
            | payments:read      |
            | payments:write     |
            | notifications:read |
            | accounts:manage    |
            | reviews:write      |

    Scenario: Account with roles returns seeded permission IDs
        Given I have an account "acc-1" with role "support"
        When I get permissions for account "acc-1"
        Then the returned permissions include "catalog:read", "orders:read", "customers:read"

    Scenario: Customer role includes customer read
        Given I have an account "acc-customer" with role "customer"
        When I get permissions for account "acc-customer"
        Then the returned permissions include "customers:read", "customers:write"

    Scenario: Unknown account returns empty list
        When I get permissions for account "unknown"
        Then the returned permissions list is empty

    Scenario: Role-less account returns empty list
        Given I have an account "acc-1" with no roles
        When I get permissions for account "acc-1"
        Then the returned permissions list is empty