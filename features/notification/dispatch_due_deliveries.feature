Feature: DispatchDueDeliveries claims due deliveries and publishes their identifiers

    All possible scenarios are:
        - Happy path: No due deliveries publishes nothing
        - Happy path: A due delivery is claimed and published to its channel topic
        - Error path: A publish failure releases the lease and is reported
        - Happy path: A default batch size is used when none is requested

    Background: Prepare the notification worker
        Given the notification service is ready

    Scenario: No due deliveries publishes nothing
        When I dispatch due deliveries for worker "w1" with limit 10
        Then 0 deliveries will be published

    Scenario: A due delivery is claimed and published to its channel topic
        Given I have a pending email delivery "d1"
        When I dispatch due deliveries for worker "w1" with limit 10
        Then 1 delivery will be published
        And the published message key and payload will be "d1"
        And the published topic will be "notification-email"

    Scenario: A publish failure releases the lease and is reported
        Given I have a pending email delivery "d1"
        And the delivery publisher is failing
        When I dispatch due deliveries for worker "w1" with limit 10
        Then I will get this error "publish delivery"
        And 0 deliveries will be published
        And the delivery "d1" lease will be released

    Scenario: A default batch size is used when none is requested
        Given I have a pending email delivery "d1"
        When I dispatch due deliveries for worker "w1" without a limit
        Then 1 delivery will be published
