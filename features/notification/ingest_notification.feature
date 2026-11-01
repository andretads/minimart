Feature: IngestNotification records a notification and schedules deliveries

    All possible scenarios are:
        - Error path: A construction error is propagated
        - Error path: Empty notification id is rejected
        - Error path: Empty user_id is rejected
        - Error path: Missing order id and dedupe key is rejected
        - Happy path: Defaults are filled in for a valid notification
        - Happy path: One delivery is queued per eligible channel
        - Happy path: A notification with no eligible channel is suppressed and not stored
        - Happy path: A duplicate dedupe key reuses the stored notification and its deliveries
        - Edge path: A transactional notification is sent when preferences are unavailable

    Background: Prepare the notification service
        Given the notification service is ready

    Scenario: A construction error is propagated
        When I ingest a notification with the construction error "bad notification"
        Then I will get this error "bad notification"

    Scenario: Empty notification id is rejected
        When I ingest a notification record with id "", order "o1" and user "u1"
        Then I will get this error "id da notificação não pode ser vazio"

    Scenario: Empty user_id is rejected
        When I ingest a notification record with id "n1", order "o1" and user ""
        Then I will get this error "id do usuário da notificação não pode ser vazio"

    Scenario: Missing order id and dedupe key is rejected
        When I ingest a notification record with id "n1", user "u1", no order id and no dedupe key
        Then I will get this error "id do pedido da notificação não pode ser vazio"

    Scenario: Defaults are filled in for a valid notification
        When I ingest a notification record with id "n1", type "order_confirmation", order "o1" and user "u1"
        Then the notification will have recipient "u1", dedupe key "o1/order_confirmation", template "order_confirmation_v1", priority "normal" and status "PENDING"

    Scenario: One delivery is queued per eligible channel
        Given user "u1" can be reached by email and SMS with email and SMS notifications enabled
        When I ingest a notification record with id "n1", type "order_confirmation", order "o1" and user "u1"
        Then one "PENDING" email delivery will be queued
        And one "PENDING" SMS delivery will be queued

    Scenario: A notification with no eligible channel is suppressed and not stored
        Given user "u1" can be reached by email
        And user "u1" has email, SMS and push notifications disabled
        When I ingest a notification record with id "n1", type "order_confirmation", order "o1" and user "u1"
        Then the notification will be suppressed
        And no notification will be recorded for order "o1"
        And no delivery will be queued

    Scenario: A duplicate dedupe key reuses the stored notification and its deliveries
        Given user "u1" can be reached by email with email notifications enabled
        And I ingest a notification record with id "n1", type "order_confirmation", order "o1" and user "u1"
        When I ingest a notification record with id "n2", type "order_confirmation", order "o1" and user "u1"
        Then the returned notification will have id "n1"
        And I will have exactly 1 notification stored for order "o1"
        And exactly 1 email delivery will be queued (idempotent)

    Scenario: A transactional notification is sent when preferences are unavailable
        Given the notification preferences storage is failing
        And user "u1" can be reached by email
        When I ingest a notification record with id "n1", type "order_confirmation", order "o1" and user "u1"
        Then a warning will be logged
        And a notification will be recorded with type "order_confirmation", order "o1", user "u1" and status "PENDING"
        And one "PENDING" email delivery will be queued
