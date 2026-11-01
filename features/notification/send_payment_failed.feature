Feature: SendPaymentFailed in reaction to a PaymentFailed event

    All possible scenarios are:
        - Error path: Malformed event payload
        - Error path: Storage failure
        - Happy path: Payment-failure email got sent with no problems
        - Happy path: Redelivered event does not duplicate the notification (idempotent send)
        - Happy path: Order updates disabled, so the payment-failure notification is suppressed (no email)
        - Edge path: Preferences storage unavailable, so the payment-failure notification is still sent
        - Edge path: User has no email address, so the payment-failure notification is skipped

    Background: Prepare the notification service
        Given the notification service is ready

    Scenario: Malformed event payload
        When the notification handler receives the malformed payload "not json"
        Then I will get an error
        And no notification will be sent

    Scenario: Storage failure
        Given the notification storage is failing
        When the notification handler receives a PaymentFailed event for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then I will get an error
        And no notification will be sent

    Scenario: Payment-failure email got sent with no problems
        When the notification handler receives a PaymentFailed event for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then a notification will be sent with success
        And an email will be sent to the user's email address
        And I will have a stored notification with a generated id, type "payment_failed", order "o1", user "u1", subject mentioning the order and a sent_at timestamp
        And the stored notification body will mention "insufficient funds"

    Scenario: Redelivered event does not duplicate the notification
        Given the notification handler receives a PaymentFailed event for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        When the notification handler receives the same PaymentFailed event again
        Then a notification will be sent with success
        And exactly 1 email will be sent (idempotent)
        And I will have exactly 1 notification stored for order "o1"

    Scenario: Order updates disabled, so the payment-failure notification is suppressed
        Given user "u1" has order updates disabled
        When the notification handler receives a PaymentFailed event for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then the notification will be suppressed
        And no email will be sent
        And no notification will be recorded for order "o1"

    Scenario: Preferences storage unavailable, so the payment-failure notification is still sent
        Given the notification preferences storage is failing
        When the notification handler receives a PaymentFailed event for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then a notification will be sent with success
        And an email will be sent to the user's email address
        And I will have a stored notification with a generated id, type "payment_failed", order "o1", user "u1", subject mentioning the order and a sent_at timestamp
        And the stored notification body will mention "insufficient funds"

    Scenario: User has no email address, so the payment-failure notification is skipped
        Given user "u1" has no email address on file
        When the notification handler receives a PaymentFailed event for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then the notification will be skipped (no email, no audit row)
        And a warning will be logged
