Feature: CreateZone with a new shipping zone

    All possible scenarios are:
        - Error path: Blank name is rejected
        - Error path: Blank country is rejected
        - Happy path: Zone got created with all fields

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Blank name is rejected
        When I create a shipping zone with name "" and country "BR"
        Then I will get this error "zona de frete não encontrada"

    Scenario: Blank country is rejected
        When I create a shipping zone with name "Brazil" and country ""
        Then I will get this error "zona de frete não encontrada"

    Scenario: Zone got created with all fields
        When I create a shipping zone with name "Brazil" and country "BR" and state "SP" and city "Sao Paulo" and postal start "01000-000" and postal end "09999-999"
        Then I will have a stored shipping zone with a generated id, name "Brazil", country "BR", state "SP", city "Sao Paulo", postal start "01000-000", postal end "09999-999" and a created_at timestamp
