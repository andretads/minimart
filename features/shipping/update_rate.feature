Feature: UpdateRate of a shipping rate

    All possible scenarios are:
        - Error path: Rate does not exist
        - Error path: Negative base cost is rejected
        - Error path: Negative free threshold is rejected
        - Happy path: Rate got updated

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Rate does not exist
        When I update shipping rate "missing" with base cost 299 cents and free threshold 0 cents and active true
        Then I will get this error "tarifa de frete não encontrada"

    Scenario: Negative base cost is rejected
        Given I have a shipping rate "sr1" for zone "sz1" and method "sm1" costing 599 cents with a free threshold of 5000 cents
        When I update shipping rate "sr1" with base cost -1 cents and free threshold 5000 cents and active true
        Then I will get this error "transição de estado inválida"

    Scenario: Negative free threshold is rejected
        Given I have a shipping rate "sr1" for zone "sz1" and method "sm1" costing 599 cents with a free threshold of 5000 cents
        When I update shipping rate "sr1" with base cost 599 cents and free threshold -1 cents and active true
        Then I will get this error "transição de estado inválida"

    Scenario: Rate got updated
        Given I have a shipping rate "sr1" for zone "sz1" and method "sm1" costing 599 cents with a free threshold of 5000 cents
        When I update shipping rate "sr1" with base cost 299 cents and free threshold 0 cents and active false
        Then the shipping rate "sr1" will have base cost 299 cents, free threshold 0 cents and active false
