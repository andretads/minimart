Feature: MarkAllNotificationsRead for a user

    All possible scenarios are:
        - Error path: Empty user_id is rejected
        - Happy path: A user with no notifications gets a no-op
        - Happy path: All of the user's notifications are marked as read
        - Happy path: Another user's notifications are left untouched

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty user_id is rejected
        When I mark all notifications as read for user ""
        Then I will get this error "id do usuário da notificação não pode ser vazio"

    Scenario: A user with no notifications gets a no-op
        When I mark all notifications as read for user "u1"
        Then no notifications will be marked as read

    Scenario: All of the user's notifications are marked as read
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        And I have a notification with id "n2", type "order_confirmation", order "o2" and user "u1"
        When I mark all notifications as read for user "u1"
        Then both notifications will have a read_at timestamp

    Scenario: Another user's notifications are left untouched
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        And I have a notification with id "n2", type "order_confirmation", order "o2" and user "u2"
        When I mark all notifications as read for user "u1"
        Then the notification "n2" will have no read_at timestamp
