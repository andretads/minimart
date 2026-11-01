Feature: GetPaymentByKey

    A client that loses the create response can recover the same payment with
    the Idempotency-Key it used for the original intent.

    Background: Clear the storage
        Given I have empty payment storage and an empty idempotency cache

    Scenario: A client recovers the same payment by its idempotency key
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        When I get the payment by key "k1" for user "u1"
        Then I will get the same payment as before

    Scenario: Another user cannot read a payment by idempotency key
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        When I get the payment by key "k1" for user "u2"
        Then I will get this error "pagamento não encontrado"

    Scenario: Unknown idempotency key is not found
        When I get the payment by key "missing" for user "u1"
        Then I will get this error "pagamento não encontrado"
