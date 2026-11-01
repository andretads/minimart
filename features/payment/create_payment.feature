Feature: CreatePayment with idempotency

    All possible scenarios are:
        - Error path: Idempotency-Key header is missing
        - Error path: Amount is not positive
        - Error path: Currency is missing
        - Error path: Method is missing
        - Error path: User is missing
        - Error path: Order is not found
        - Error path: Payment user does not match order
        - Error path: Payment amount does not match order total
        - Error path: Payment currency does not match order
        - Error path: Same key reused with different parameters
        - Error path: Storage failure
        - Edge path: Provider times out, so the outcome is unknown
        - Edge path: Redis is unavailable, so idempotency falls back to the database
        - Edge path: A concurrent request is still in progress for the same key
        - Edge path: Order is cancelled while the charge is in flight, so the capture is refunded
        - Edge path: The in-flight refund fails, so the reconciler retries it
        - Happy path: Payment is approved and captured, and replays return the same payment
        - Happy path: Provider declines the payment

    Background: Clear the storage
        Given I have empty payment storage and an empty idempotency cache

    Scenario: Idempotency-Key header is missing
        When I create a payment with no idempotency key
        Then I will get this error "chave de idempotência é obrigatória"

    Scenario: Amount is not positive
        When I create a payment for order "o1" of 0 cents in "USD" via "card" with key "k1"
        Then I will get this error "valor do pagamento deve ser maior que zero"

    Scenario: Currency is missing
        When I create a payment for order "o1" of 1500 cents in "" via "card" with key "k1"
        Then I will get this error "moeda do pagamento é obrigatória"

    Scenario: Method is missing
        When I create a payment for order "o1" of 1500 cents in "USD" via "" with key "k1"
        Then I will get this error "método de pagamento é obrigatório"

    Scenario: User is missing
        When I create a payment for order "o1" for no user of 1500 cents in "USD" via "card" with key "k1"
        Then I will get this error "id do usuário do pagamento é obrigatório"

    Scenario: Same key reused with different parameters
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        When I create a payment for order "o1" of 2500 cents in "USD" via "card" with key "k1"
        Then I will get this error "chave de idempotência reutilizada com parâmetros diferentes"

    Scenario: Storage failure
        Given the payment storage is failing
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will get this error "disk on fire"
        And the idempotency reservation for key "k1" is released

    Scenario: Provider times out, so the outcome is unknown
        Given the provider times out for a payment of 7777 cents
        When I create a payment for order "o1" of 7777 cents in "USD" via "card" with key "k1"
        Then I will have a stored payment for order "o1" with status "unknown"
        And a single PaymentUnknown event will be published

    Scenario: Redis is unavailable, so idempotency falls back to the database
        Given the idempotency cache is unavailable
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will have a stored payment for order "o1" with status "succeeded"
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will get the same payment as before

    Scenario: A concurrent request is still in progress for the same key
        Given an idempotency reservation exists for key "k1" for the same request parameters
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will get this error "já existe um pagamento com esta chave de idempotência em processamento"

    Scenario: Payment is approved and captured, and replays return the same payment
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will have a stored payment for order "o1" with status "succeeded"
        And a single PaymentInitiated event will be published
        And a single PaymentSucceeded event will be published
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will get the same payment as before

    Scenario: Provider declines the payment
        Given the provider declines a payment of 9999 cents
        When I create a payment for order "o1" of 9999 cents in "USD" via "card" with key "k1"
        Then I will have a stored payment for order "o1" with status "failed"
        And a single PaymentFailed event will be published

    Scenario: Payment is rejected for a cancelled order
        Given I have a stored order with id "o1" for user "u1" with a total of 1500 cents
        And the order with id "o1" is already cancelled
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will get this error "pedido não pode mais ser pago"

    Scenario: Payment is rejected for a failed order
        Given I have a stored order with id "o1" for user "u1" with a total of 1500 cents
        And the order with id "o1" has status "payment_failed"
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will get this error "pedido não pode mais ser pago"

    Scenario: Order is cancelled while the charge is in flight
        Given I have a stored order with id "o1" for user "u1" with a total of 1500 cents
        And the order with id "o1" is cancelled after the payment started
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will have a stored payment for order "o1" with status "refunded"
        And a single PaymentRefunded event will be published with reason "pedido cancelado"

    Scenario: Order is cancelled while the charge is in flight and the refund fails
        Given I have a stored order with id "o1" for user "u1" with a total of 1500 cents
        And the order with id "o1" is cancelled after the payment started
        And the provider fails refunds
        When I create a payment for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        Then I will get this error "reembolso falhou no provedor de pagamentos"
        And I will have a stored payment for order "o1" with status "pending"
        And the stored payment keeps the charge reference for the reconciler
