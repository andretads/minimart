Feature: DeliverNotification sends a queued delivery through its provider

    All possible scenarios are:
        - Error path: Delivery does not exist
        - Error path: Notification does not exist
        - Happy path: A successful send marks the delivery and notification as sent
        - Happy path: A confirmed delivery marks the delivery and notification as delivered
        - Happy path: A transient provider error schedules a retry
        - Error path: A permanent provider error fails the delivery permanently
        - Error path: An exhausted retry budget fails the delivery permanently
        - Happy path: A terminal delivery is not processed again

    Background: Prepare the notification service
        Given the notification service is ready

    Scenario: Delivery does not exist
        When I deliver the delivery "missing"
        Then I will get this error "notification: delivery not found"

    Scenario: Notification does not exist
        Given I have a pending email delivery "d1" for a notification that no longer exists
        When I deliver the delivery "d1"
        Then I will get this error "notificação não encontrada"

    Scenario: A successful send marks the delivery and notification as sent
        Given I have a pending email delivery "d1"
        When I deliver the delivery "d1"
        Then the delivery "d1" will be "SENT"
        And the delivery "d1" will have a provider message id
        And the notification will be "SENT"

    Scenario: A confirmed delivery marks the delivery and notification as delivered
        Given I have a pending email delivery "d1" and the provider confirms delivery
        When I deliver the delivery "d1"
        Then the delivery "d1" will be "DELIVERED"
        And the notification will be "DELIVERED"

    Scenario: A transient provider error schedules a retry
        Given I have a pending email delivery "d1" and the provider returns a transient error
        When I deliver the delivery "d1"
        Then I will get no error
        And the delivery "d1" will be "RETRY_PENDING"
        And the delivery "d1" will have a next attempt timestamp
        And the notification will be "RETRY_PENDING"

    Scenario: A permanent provider error fails the delivery permanently
        Given I have a pending email delivery "d1" and the provider returns a permanent error
        When I deliver the delivery "d1"
        Then I will get a permanent error
        And the delivery "d1" will be "FAILED"
        And the notification will be "FAILED"

    Scenario: An exhausted retry budget fails the delivery permanently
        Given I have a pending email delivery "d1" at its last allowed attempt and the provider returns a transient error
        When I deliver the delivery "d1"
        Then I will get a permanent error
        And the delivery "d1" will be "FAILED"
        And the notification will be "FAILED"

    Scenario: A terminal delivery is not processed again
        Given I have a sent email delivery "d1"
        When I deliver the delivery "d1"
        Then I will get no error
        And the delivery "d1" will still be "SENT"
        And the provider will not be called
