Feature: CreateQuote prices a shipping quote against the cart subtotal

    A quote snapshots the destination and the shipping cost for one method.
    The rate decides the price: the subtotal the customer is shopping with
    reaches the rate's free shipping threshold and the quote is free, below
    the threshold the quote costs the rate's base cost. A threshold of zero
    disables free shipping.

    The subtotal the customer sends is advisory. The cart checkout re-prices
    the quote with the subtotal it computed from its own items, so the amount
    charged never depends on the number the customer claimed.

    Creating a quote requires an Idempotency-Key: a retry with the same key
    and the same parameters replays the quote that already exists, so a
    dropped response never leaves the customer with two quotes. The same key
    with different parameters is a conflict.

    All possible scenarios are:
        - Error path: Blank method_id is rejected
        - Error path: The method does not exist
        - Error path: The method is inactive
        - Error path: The address does not belong to the customer
        - Error path: No zone matches the address
        - Error path: No rate matches the zone and method
        - Error path: The request carries no idempotency key
        - Error path: The idempotency key is reused with other parameters
        - Happy path: A subtotal below the threshold costs the base cost
        - Happy path: A subtotal at the threshold is free
        - Happy path: A subtotal above the threshold is free
        - Happy path: A rate without a free threshold always costs the base cost
        - Happy path: The same idempotency key replays the original quote
        - Edge path: The quote snapshots the subtotal and the threshold

    Background: Clear the storage
        Given I have empty shipping storage
        And I have a shipping method "sm1" named "Standard" that is active
        And I have a shipping zone "sz1" for "BR" that is active
        And I have a shipping rate "sr1" for zone "sz1" and method "sm1" costing 599 cents with a free threshold of 5000 cents
        And I have an address "addr1" for customer "cust1"

    Scenario: Blank method_id is rejected
        When I create a quote for customer "cust1" with method "" and address "addr1"
        Then I will get this error "cotação de frete inválida"

    Scenario: The method does not exist
        When I create a quote for customer "cust1" with method "missing" and address "addr1"
        Then I will get this error "método de frete não encontrado"

    Scenario: The method is inactive
        Given I have a shipping method "sm2" named "Express" that is inactive
        When I create a quote for customer "cust1" with method "sm2" and address "addr1"
        Then I will get this error "método de frete inativo"

    Scenario: The address does not belong to the customer
        When I create a quote for customer "cust2" with method "sm1" and address "addr1"
        Then I will get this error "endereço não encontrado"

    Scenario: No zone matches the address
        Given I have a shipping zone "sz2" for "US" that is active
        And I have an address "addr2" for customer "cust1" in "US"
        When I create a quote for customer "cust1" with method "sm1" and address "addr2"
        Then I will get this error "nenhuma tarifa de frete corresponde ao destino"

    Scenario: No rate matches the zone and method
        Given I have a shipping method "sm3" named "Freight" that is active
        When I create a quote for customer "cust1" with method "sm3" and address "addr1"
        Then I will get this error "nenhuma tarifa de frete corresponde ao destino"

    Scenario: The request carries no idempotency key
        When I create a quote for customer "cust1" with method "sm1" and address "addr1" without an idempotency key
        Then I will get this error "cabeçalho Idempotency-Key é obrigatório"

    Scenario: The idempotency key is reused with other parameters
        Given I have a quote for customer "cust1" with method "sm1" and address "addr1" under the idempotency key "key-1"
        When I create a quote for customer "cust1" with method "sm1" and address "addr1" for a subtotal of 2400 cents under the idempotency key "key-1"
        Then I will get this error "chave de idempotência reutilizada com parâmetros diferentes"

    Scenario: A subtotal below the threshold costs the base cost
        When I create a quote for customer "cust1" with method "sm1" and address "addr1" for a subtotal of 4999 cents
        Then the quote costs 599 cents

    Scenario: A subtotal at the threshold is free
        When I create a quote for customer "cust1" with method "sm1" and address "addr1" for a subtotal of 5000 cents
        Then the quote costs 0 cents

    Scenario: A subtotal above the threshold is free
        When I create a quote for customer "cust1" with method "sm1" and address "addr1" for a subtotal of 12000 cents
        Then the quote costs 0 cents

    Scenario: A rate without a free threshold always costs the base cost
        Given I have a shipping rate "sr2" for zone "sz1" and method "sm1" costing 299 cents with a free threshold of 0 cents
        When I create a quote for customer "cust1" with method "sm1" and address "addr1" for a subtotal of 12000 cents
        Then the quote costs 299 cents

    Scenario: The same idempotency key replays the original quote
        Given I have a quote for customer "cust1" with method "sm1" and address "addr1" under the idempotency key "key-1"
        When I create a quote for customer "cust1" with method "sm1" and address "addr1" under the idempotency key "key-1"
        Then I get the same quote "key-1" created

    Scenario: The quote snapshots the subtotal and the threshold
        When I create a quote for customer "cust1" with method "sm1" and address "addr1" for a subtotal of 1200 cents
        Then the quote is priced for a subtotal of 1200 cents
        And the quote snapshots a free threshold of 5000 cents
