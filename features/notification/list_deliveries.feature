Feature: ListDeliveries for a notification

    All possible scenarios are:
        - Error path: Empty notification id is rejected
        - Error path: Notification does not exist
        - Error path: Another user's notification is not found
        - Happy path: The notification's deliveries are returned

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty notification id is rejected
        When I list the deliveries for notification "" and user "u1"
        Then I will get this error "notification id cannot be empty"

    Scenario: Notification does not exist
        When I list the deliveries for notification "missing" and user "u1"
        Then I will get this error "notificação não encontrada"

    Scenario: Another user's notification is not found
        Given user "u1" has a notification "n1" with a pending email delivery
        When I list the deliveries for notification "n1" and user "u2"
        Then I will get this error "notificação não encontrada"

    Scenario: The notification's deliveries are returned
        Given user "u1" has a notification "n1" with a pending email delivery
        When I list the deliveries for notification "n1" and user "u1"
        Then I will get a list with 1 delivery
        And the delivery will be "PENDING" on the "email" channel
