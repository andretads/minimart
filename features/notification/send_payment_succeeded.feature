Feature: SendPaymentSucceeded in reaction to a PaymentSucceeded event

    All possible scenarios are:
        - Error path: Malformed event payload
        - Error path: Storage failure
        - Happy path: Payment-success email got sent with no problems
        - Happy path: Redelivered event does not duplicate the notification (idempotent send)
        - Happy path: Order updates disabled, so the payment-success notification is suppressed (no email)
        - Edge path: Preferences storage unavailable, so the payment-success notification is still sent
        - Edge path: User has no email address, so the payment-success notification is skipped

    Background: Prepare the notification service
        Given the notification service is ready

    Scenario: Malformed event payload
        When the notification handler receives the malformed payload "not json"
        Then I will get an error
        And no notification will be sent

    Scenario: Storage failure
        Given the notification storage is failing
        When the notification handler receives a PaymentSucceeded event for order "o1", user "u1" and amount 1250 cents
        Then I will get an error
        And no notification will be sent

    Scenario: Payment-success email got sent with no problems
        When the notification handler receives a PaymentSucceeded event for order "o1", user "u1" and amount 1250 cents
        Then a notification will be sent with success
        And an email will be sent to the user's email address
        And I will have a stored notification with a generated id, type "payment_succeeded", order "o1", user "u1", subject mentioning the order and a sent_at timestamp

    Scenario: Redelivered event does not duplicate the notification
        Given the notification handler receives a PaymentSucceeded event for order "o1", user "u1" and amount 1250 cents
        When the notification handler receives the same PaymentSucceeded event again
        Then a notification will be sent with success
        And exactly 1 email will be sent (idempotent)
        And I will have exactly 1 notification stored for order "o1"

    Scenario: Order updates disabled, so the payment-success notification is suppressed
        Given user "u1" has order updates disabled
        When the notification handler receives a PaymentSucceeded event for order "o1", user "u1" and amount 1250 cents
        Then the notification will be suppressed
        And no email will be sent
        And no notification will be recorded for order "o1"

    Scenario: Preferences storage unavailable, so the payment-success notification is still sent
        Given the notification preferences storage is failing
        When the notification handler receives a PaymentSucceeded event for order "o1", user "u1" and amount 1250 cents
        Then a notification will be sent with success
        And an email will be sent to the user's email address
        And I will have a stored notification with a generated id, type "payment_succeeded", order "o1", user "u1", subject mentioning the order and a sent_at timestamp

    Scenario: User has no email address, so the payment-success notification is skipped
        Given user "u1" has no email address on file
        When the notification handler receives a PaymentSucceeded event for order "o1", user "u1" and amount 1250 cents
        Then the notification will be skipped (no email, no audit row)
        And a warning will be logged
