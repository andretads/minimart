Feature: GetShipment of a shipment

    All possible scenarios are:
        - Error path: Shipment does not exist
        - Happy path: The shipment is returned

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Shipment does not exist
        When I get the shipment "missing"
        Then I will get this error "envio não encontrado"

    Scenario: The shipment is returned
        Given I have a shipment "s1" for user "u1" and order "o1"
        When I get the shipment "s1"
        Then I will get the shipment "s1" for order "o1"
