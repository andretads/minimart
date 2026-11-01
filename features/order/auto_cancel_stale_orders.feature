Feature: AutoCancelStaleOrders

    The worker cancels placed orders that remain unpaid for 5 minutes,
    releasing their reserved stock like a manual cancellation.

    All possible scenarios are:
        - Happy path: Stale placed orders are cancelled and stock is released
        - Edge case: Paid or already cancelled orders are skipped
        - Edge case: A concurrent status change is not clobbered

    Background: Clear the storage
        Given I have empty storage

    Scenario: Stale placed orders are cancelled
        Given I have a placed order with id "o1" created 6 minutes ago
        When the worker runs the auto-cancel scan
        Then the order with id "o1" will have status "cancelled"
        And a single OrderCancelled event will be published with order id "o1"
        And a single OrderStockRelease event will be published with order id "o1"

    Scenario: Fresh placed orders are left alone
        Given I have a placed order with id "o2" created 1 minute ago
        When the worker runs the auto-cancel scan
        Then the order with id "o2" will still have status "placed"

    Scenario: Concurrent status change is not clobbered
        Given I have a placed order with id "o3" created 6 minutes ago
        And the order with id "o3" is paid before the scan persists
        When the worker runs the auto-cancel scan
        Then the order with id "o3" will keep status "paid"
