Feature: ListZones returns shipping zones

    All possible scenarios are:
        - Happy path: Zones are returned

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Zones are returned
        Given I have a shipping zone "sz1" for "BR" that is active
        When I list shipping zones
        Then I will get 1 shipping zone
