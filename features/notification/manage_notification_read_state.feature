Feature: Manage notification read state and deletion

    All possible scenarios are:
        - Mark a notification as read
        - Mark a notification as unread
        - Mark all notifications as read
        - Delete a notification

    Background: Clear the storage
        Given I have empty storage

    Scenario: Mark a notification as read
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        When I mark the notification "n1" as read for user "u1"
        Then the notification "n1" will have a read_at timestamp

    Scenario: Mark a notification as unread
        Given I have a read notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        When I mark the notification "n1" as unread for user "u1"
        Then the notification "n1" will have no read_at timestamp

    Scenario: Mark all notifications as read
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        And I have a notification with id "n2", type "order_confirmation", order "o2" and user "u1"
        When I mark all notifications as read for user "u1"
        Then both notifications will have a read_at timestamp

    Scenario: Delete a notification
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        When I delete the notification "n1" for user "u1"
        Then the notification "n1" will no longer exist
