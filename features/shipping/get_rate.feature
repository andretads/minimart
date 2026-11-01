Feature: GetRate of a shipping rate

    All possible scenarios are:
        - Error path: Rate does not exist
        - Happy path: The rate is returned

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Rate does not exist
        When I get the shipping rate "missing"
        Then I will get this error "tarifa de frete não encontrada"

    Scenario: The rate is returned
        Given I have a shipping rate "sr1" for zone "sz1" and method "sm1" costing 599 cents with a free threshold of 5000 cents
        When I get the shipping rate "sr1"
        Then I will get the shipping rate "sr1" costing 599 cents
