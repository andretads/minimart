Feature: CreateRate with a new shipping rate

    All possible scenarios are:
        - Error path: Blank zone id is rejected
        - Error path: Blank method id is rejected
        - Error path: Negative base cost is rejected
        - Error path: Negative free threshold is rejected
        - Happy path: Rate got created with all fields

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Blank zone id is rejected
        When I create a shipping rate with zone "" and method "sm1" and base cost 599 cents and free threshold 5000 cents
        Then I will get this error "tarifa de frete não encontrada"

    Scenario: Blank method id is rejected
        When I create a shipping rate with zone "sz1" and method "" and base cost 599 cents and free threshold 5000 cents
        Then I will get this error "tarifa de frete não encontrada"

    Scenario: Negative base cost is rejected
        When I create a shipping rate with zone "sz1" and method "sm1" and base cost -1 cents and free threshold 5000 cents
        Then I will get this error "transição de estado inválida"

    Scenario: Negative free threshold is rejected
        When I create a shipping rate with zone "sz1" and method "sm1" and base cost 599 cents and free threshold -1 cents
        Then I will get this error "transição de estado inválida"

    Scenario: Rate got created with all fields
        Given I have a shipping zone "sz1" for "BR" that is active
        And I have a shipping method "sm1" named "Standard" that is active
        When I create a shipping rate with zone "sz1" and method "sm1" and base cost 599 cents and free threshold 5000 cents and active true
        Then I will have a stored shipping rate with a generated id, zone "sz1", method "sm1", base cost 599 cents, free threshold 5000 cents, active true and a created_at timestamp
