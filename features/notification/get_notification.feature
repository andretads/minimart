Feature: GetNotification by id

    All possible scenarios are:
        - Error path: Empty notification id is rejected
        - Error path: Empty user_id is rejected
        - Error path: Notification does not exist
        - Error path: Another user's notification is not found
        - Happy path: Notification got returned with no problems

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty notification id is rejected
        When I get the notification with id "" for user "u1"
        Then I will get this error "id da notificação não pode ser vazio"

    Scenario: Empty user_id is rejected
        When I get the notification with id "n1" for user ""
        Then I will get this error "id do usuário da notificação não pode ser vazio"

    Scenario: Notification does not exist
        When I get the notification with id "missing" for user "u1"
        Then I will get this error "notificação não encontrada"

    Scenario: Another user's notification is not found
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        When I get the notification with id "n1" for user "u2"
        Then I will get this error "notificação não encontrada"

    Scenario: Notification got returned with no problems
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        When I get the notification with id "n1" for user "u1"
        Then I will get a notification with id "n1", type "order_confirmation", order "o1", user "u1" and a sent_at timestamp
