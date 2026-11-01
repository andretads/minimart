Feature: RefundPayment for cancelled orders

    All possible scenarios are:
        - Error path: Payment does not exist
        - Error path: Payment is not succeeded
        - Error path: Order is not cancelled
        - Error path: Provider fails the refund
        - Edge path: Payment is already refunded
        - Edge path: Refund loses a race, so the payment was already refunded
        - Happy path: Succeeded payment on a cancelled order is refunded

    Background: Clear the storage
        Given I have empty payment storage and an empty idempotency cache

    Scenario: Payment does not exist
        Given I have no payments stored
        When I refund the payment with id "missing"
        Then I will get this error "pagamento não encontrado"

    Scenario: Payment is not succeeded
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        And the provider declined the payment
        When I refund the payment for order "o1"
        Then I will get this error "payment cannot be refunded"
        And the provider will not be asked to refund

    Scenario: Order is not cancelled
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        And the order "o1" has status "paid"
        When I refund the payment for order "o1"
        Then I will get this error "pedido não está cancelado"
        And the provider will not be asked to refund

    Scenario: Provider fails the refund
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        And the order "o1" is cancelled
        And the provider fails refunds
        When I refund the payment for order "o1"
        Then I will get this error "reembolso falhou no provedor de pagamentos"
        And I will have a stored payment for order "o1" with status "succeeded"

    Scenario: Payment is already refunded
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        And the order "o1" is cancelled
        And the payment was refunded
        When I refund the payment for order "o1"
        Then I will have a stored payment for order "o1" with status "refunded"
        And the provider will not be asked to refund

    Scenario: Refund loses a race, so the payment was already refunded
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        And the order "o1" is cancelled
        And the payment refund loses a race
        When the order-cancelled event for order "o1" is delivered again
        Then I will have a stored payment for order "o1" with status "refunded"

    Scenario: Succeeded payment on a cancelled order is refunded
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        And the order "o1" is cancelled
        When I refund the payment for order "o1"
        Then I will have a stored payment for order "o1" with status "refunded"
        And a single PaymentRefunded event will be published with reason "reembolso solicitado"
