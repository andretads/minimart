Feature: MarkNotificationRead for a notification

    All possible scenarios are:
        - Error path: Empty notification id is rejected
        - Error path: Empty user_id is rejected
        - Error path: Notification does not exist
        - Error path: Another user's notification is not found
        - Happy path: The notification is marked as read

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty notification id is rejected
        When I mark the notification "" as read for user "u1"
        Then I will get this error "id da notificação não pode ser vazio"

    Scenario: Empty user_id is rejected
        When I mark the notification "n1" as read for user ""
        Then I will get this error "id do usuário da notificação não pode ser vazio"

    Scenario: Notification does not exist
        When I mark the notification "missing" as read for user "u1"
        Then I will get this error "notificação não encontrada"

    Scenario: Another user's notification is not found
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        When I mark the notification "n1" as read for user "u2"
        Then I will get this error "notificação não encontrada"

    Scenario: The notification is marked as read
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        When I mark the notification "n1" as read for user "u1"
        Then the notification "n1" will have a read_at timestamp
