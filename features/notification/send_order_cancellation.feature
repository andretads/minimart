Feature: SendOrderCancellation in reaction to an OrderCancelled event

    All possible scenarios are:
        - Error path: Malformed event payload
        - Error path: Storage failure
        - Happy path: Cancellation email got sent with no problems
        - Happy path: Redelivered event does not duplicate the notification (idempotent send)
        - Happy path: Order updates disabled, so the cancellation is suppressed (no email)
        - Edge path: Preferences storage unavailable, so the cancellation is still sent
        - Edge path: User has no email address, so the cancellation is skipped

    Background: Prepare the notification service
        Given the notification service is ready

    Scenario: Malformed event payload
        When the notification handler receives the malformed payload "not json"
        Then I will get an error
        And no notification will be sent

    Scenario: Storage failure
        Given the notification storage is failing
        When the notification handler receives an OrderCancelled event for order "o1", user "u1", total 1250 cents and 2 items
        Then I will get an error
        And no notification will be sent

    Scenario: Cancellation email got sent with no problems
        When the notification handler receives an OrderCancelled event for order "o1", user "u1", total 1250 cents and 2 items
        Then a notification will be sent with success
        And an email will be sent to the user's email address
        And I will have a stored notification with a generated id, type "order_cancelled", order "o1", user "u1", subject mentioning the order and a sent_at timestamp

    Scenario: Redelivered event does not duplicate the notification
        Given the notification handler receives an OrderCancelled event for order "o1", user "u1", total 1250 cents and 2 items
        When the notification handler receives the same OrderCancelled event again
        Then a notification will be sent with success
        And exactly 1 email will be sent (idempotent)
        And I will have exactly 1 notification stored for order "o1"

    Scenario: Order updates disabled, so the cancellation is suppressed
        Given user "u1" has order updates disabled
        When the notification handler receives an OrderCancelled event for order "o1", user "u1", total 1250 cents and 2 items
        Then the notification will be suppressed
        And no email will be sent
        And no notification will be recorded for order "o1"

    Scenario: Preferences storage unavailable, so the cancellation is still sent
        Given the notification preferences storage is failing
        When the notification handler receives an OrderCancelled event for order "o1", user "u1", total 1250 cents and 2 items
        Then a notification will be sent with success
        And an email will be sent to the user's email address
        And I will have a stored notification with a generated id, type "order_cancelled", order "o1", user "u1", subject mentioning the order and a sent_at timestamp

    Scenario: User has no email address, so the cancellation is skipped
        Given user "u1" has no email address on file
        When the notification handler receives an OrderCancelled event for order "o1", user "u1", total 1250 cents and 2 items
        Then the notification will be skipped (no email, no audit row)
        And a warning will be logged
