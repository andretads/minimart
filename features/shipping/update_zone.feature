Feature: UpdateZone of a shipping zone

    All possible scenarios are:
        - Error path: Zone does not exist
        - Error path: Blank name is rejected
        - Error path: Blank country is rejected
        - Happy path: Zone got updated

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Zone does not exist
        When I update shipping zone "missing" with name "Brazil" and country "BR"
        Then I will get this error "zona de frete não encontrada"

    Scenario: Blank name is rejected
        Given I have a shipping zone "sz1" for "BR" that is active
        When I update shipping zone "sz1" with name "" and country "BR"
        Then I will get this error "zona de frete não encontrada"

    Scenario: Blank country is rejected
        Given I have a shipping zone "sz1" for "BR" that is active
        When I update shipping zone "sz1" with name "Brazil" and country ""
        Then I will get this error "zona de frete não encontrada"

    Scenario: Zone got updated
        Given I have a shipping zone "sz1" for "BR" that is active
        When I update shipping zone "sz1" with name "Brazil" and country "BR" and state "RJ" and city "Rio" and postal start "20000-000" and postal end "29999-999"
        Then the shipping zone "sz1" will have name "Brazil", country "BR", state "RJ", city "Rio", postal start "20000-000" and postal end "29999-999"
