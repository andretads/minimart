Feature: ListRates returns shipping rates

    All possible scenarios are:
        - Happy path: Rates are returned

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Rates are returned
        Given I have a shipping rate "sr1" for zone "sz1" and method "sm1" costing 599 cents with a free threshold of 5000 cents
        When I list shipping rates for zone "sz1" and method "sm1"
        Then I will get 1 shipping rate
