Feature: SendPaymentFailed in reaction to a PaymentFailed event

    All possible scenarios are:
        - Error path: Empty order id is rejected
        - Error path: Empty user_id is rejected
        - Error path: Non-positive amount is rejected
        - Error path: Notification storage failure is propagated
        - Happy path: The payment failure is recorded and queued for delivery
        - Happy path: Redelivered event does not duplicate the notification (idempotent)
        - Happy path: Email disabled and no other contact suppresses the notification
        - Edge path: Preferences storage unavailable, so the notification is still sent
        - Edge path: User has no contact on file, so the notification is suppressed

    Background: Prepare the notification service
        Given the notification service is ready

    Scenario: Empty order id is rejected
        When I send the payment failure for order "", user "u1", amount 1250 cents and reason "insufficient funds"
        Then I will get this error "id do pedido da notificação não pode ser vazio"
        And no notification will be recorded for order ""

    Scenario: Empty user_id is rejected
        When I send the payment failure for order "o1", user "", amount 1250 cents and reason "insufficient funds"
        Then I will get this error "id do usuário da notificação não pode ser vazio"
        And no notification will be recorded for order "o1"

    Scenario: Non-positive amount is rejected
        When I send the payment failure for order "o1", user "u1", amount 0 cents and reason "insufficient funds"
        Then I will get this error "total da notificação deve ser maior que zero"
        And no notification will be recorded for order "o1"

    Scenario: Notification storage failure is propagated
        Given the notification storage is failing
        When I send the payment failure for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then I will get this error "disk on fire"
        And no notification will be recorded for order "o1"

    Scenario: The payment failure is recorded and queued for delivery
        Given user "u1" can be reached by email with email notifications enabled
        When I send the payment failure for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then a notification will be recorded with type "payment_failed", order "o1", user "u1" and status "PENDING"
        And the notification will have dedupe key "o1/payment_failed"
        And one "PENDING" email delivery will be queued
        And the subject will mention the order "o1"
        And the recorded notification body will mention "insufficient funds"
        And the notification will have a sent_at timestamp

    Scenario: Redelivered event does not duplicate the notification
        Given user "u1" can be reached by email with email notifications enabled
        And I send the payment failure for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        When I send the payment failure for order "o1", user "u1", amount 1250 cents and reason "insufficient funds" again
        Then I will have exactly 1 notification stored for order "o1"
        And exactly 1 email delivery will be queued (idempotent)

    Scenario: Email disabled and no other contact suppresses the notification
        Given user "u1" can be reached by email
        And user "u1" has email, SMS and push notifications disabled
        When I send the payment failure for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then the notification will be suppressed
        And no notification will be recorded for order "o1"
        And no delivery will be queued

    Scenario: Preferences storage unavailable, so the notification is still sent
        Given the notification preferences storage is failing
        And user "u1" can be reached by email
        When I send the payment failure for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then a warning will be logged
        And a notification will be recorded with type "payment_failed", order "o1", user "u1" and status "PENDING"
        And one "PENDING" email delivery will be queued
        And the recorded notification body will mention "insufficient funds"

    Scenario: User has no contact on file, so the notification is suppressed
        Given user "u1" has no contact on file
        When I send the payment failure for order "o1", user "u1", amount 1250 cents and reason "insufficient funds"
        Then the notification will be suppressed
        And no notification will be recorded for order "o1"
        And no delivery will be queued
