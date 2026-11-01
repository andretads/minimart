Feature: GetNotification by id

    All possible scenarios are:
        - Error path: Notification does not exist
        - Happy path: Notification got returned with no problems

    Background: Clear the storage
        Given I have empty storage

    Scenario: Notification does not exist
        When I get the notification with id "missing"
        Then I will get this error "notificação não encontrada"

    Scenario: Notification got returned with no problems
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        When I get the notification with id "n1"
        Then I will get a notification with id "n1", type "order_confirmation", order "o1", user "u1" and a sent_at timestamp
