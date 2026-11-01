Feature: GetZone of a shipping zone

    All possible scenarios are:
        - Error path: Zone does not exist
        - Happy path: The zone is returned

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Zone does not exist
        When I get the shipping zone "missing"
        Then I will get this error "zona de frete não encontrada"

    Scenario: The zone is returned
        Given I have a shipping zone "sz1" for "BR" that is active
        When I get the shipping zone "sz1"
        Then I will get the shipping zone "sz1" for "BR"
