Feature: GetPayment by id

    All possible scenarios are:
        - Error path: Payment does not exist
        - Happy path: Payment got returned with no problems

    Background: Clear the storage
        Given I have empty payment storage and an empty idempotency cache

    Scenario: Payment does not exist
        Given I have no payments stored
        When I get the payment with id "missing"
        Then I will get this error "pagamento não encontrado"

    Scenario: Payment got returned with no problems
        Given a payment was created for order "o1" of 1500 cents in "USD" via "card" with key "k1"
        When I get the payment with its id
        Then I will get a payment for order "o1" with status "succeeded"
