Feature: SendOrderCancellation in reaction to an OrderCancelled event

    All possible scenarios are:
        - Error path: Empty order id is rejected
        - Error path: Empty user_id is rejected
        - Error path: Non-positive total is rejected
        - Error path: Notification storage failure is propagated
        - Happy path: The cancellation is recorded and queued for delivery
        - Happy path: Redelivered event does not duplicate the notification (idempotent)
        - Happy path: Email disabled and no other contact suppresses the cancellation
        - Edge path: Preferences storage unavailable, so the cancellation is still sent
        - Edge path: User has no contact on file, so the cancellation is suppressed

    Background: Prepare the notification service
        Given the notification service is ready

    Scenario: Empty order id is rejected
        When I send the order cancellation for order "", user "u1" and total 1250 cents
        Then I will get this error "id do pedido da notificação não pode ser vazio"
        And no notification will be recorded for order ""

    Scenario: Empty user_id is rejected
        When I send the order cancellation for order "o1", user "" and total 1250 cents
        Then I will get this error "id do usuário da notificação não pode ser vazio"
        And no notification will be recorded for order "o1"

    Scenario: Non-positive total is rejected
        When I send the order cancellation for order "o1", user "u1" and total 0 cents
        Then I will get this error "total da notificação deve ser maior que zero"
        And no notification will be recorded for order "o1"

    Scenario: Notification storage failure is propagated
        Given the notification storage is failing
        When I send the order cancellation for order "o1", user "u1" and total 1250 cents
        Then I will get this error "disk on fire"
        And no notification will be recorded for order "o1"

    Scenario: The cancellation is recorded and queued for delivery
        Given user "u1" can be reached by email with email notifications enabled
        When I send the order cancellation for order "o1", user "u1" and total 1250 cents
        Then a notification will be recorded with type "order_cancelled", order "o1", user "u1" and status "PENDING"
        And the notification will have dedupe key "o1/order_cancelled"
        And one "PENDING" email delivery will be queued
        And the subject will mention the order "o1"
        And the notification will have a sent_at timestamp

    Scenario: Redelivered event does not duplicate the notification
        Given user "u1" can be reached by email with email notifications enabled
        And I send the order cancellation for order "o1", user "u1" and total 1250 cents
        When I send the order cancellation for order "o1", user "u1" and total 1250 cents again
        Then I will have exactly 1 notification stored for order "o1"
        And exactly 1 email delivery will be queued (idempotent)

    Scenario: Email disabled and no other contact suppresses the cancellation
        Given user "u1" can be reached by email
        And user "u1" has email, SMS and push notifications disabled
        When I send the order cancellation for order "o1", user "u1" and total 1250 cents
        Then the notification will be suppressed
        And no notification will be recorded for order "o1"
        And no delivery will be queued

    Scenario: Preferences storage unavailable, so the cancellation is still sent
        Given the notification preferences storage is failing
        And user "u1" can be reached by email
        When I send the order cancellation for order "o1", user "u1" and total 1250 cents
        Then a warning will be logged
        And a notification will be recorded with type "order_cancelled", order "o1", user "u1" and status "PENDING"
        And one "PENDING" email delivery will be queued

    Scenario: User has no contact on file, so the cancellation is suppressed
        Given user "u1" has no contact on file
        When I send the order cancellation for order "o1", user "u1" and total 1250 cents
        Then the notification will be suppressed
        And no notification will be recorded for order "o1"
        And no delivery will be queued
