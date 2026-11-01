Feature: ListNotifications for a user

    All possible scenarios are:
        - Error path: Missing user_id is rejected
        - Happy path: User with no notifications gets an empty page and a zero total
        - Happy path: Only the user's notifications are returned, newest first
        - Happy path: Pagination returns the requested page

    Background: Clear the storage
        Given I have empty storage

    Scenario: Missing user_id is rejected
        When I list the notifications without a user_id
        Then I will get this error "id do usuário da notificação não pode ser vazio"

    Scenario: User with no notifications gets an empty page and a zero total
        When I list the notifications for user "u1" on page 1 with page size 20
        Then I will get an empty list of notifications
        And the total will be 0

    Scenario: Only the user's notifications are returned, newest first
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        And I have a notification with id "n2", type "order_confirmation", order "o2" and user "u1"
        And I have a notification with id "n3", type "order_confirmation", order "o3" and user "u2"
        When I list the notifications for user "u1" on page 1 with page size 20
        Then I will get a list with 2 notifications
        And the total will be 2
        And the first notification will have id "n2"

    Scenario: Pagination returns the requested page
        Given I have a notification with id "n1", type "order_confirmation", order "o1" and user "u1"
        And I have a notification with id "n2", type "order_confirmation", order "o2" and user "u1"
        And I have a notification with id "n3", type "order_confirmation", order "o3" and user "u1"
        When I list the notifications for user "u1" on page 2 with page size 2
        Then I will get a list with 1 notification
        And the first notification will have id "n1"
