Feature: DeliverShipment moves a shipped shipment to delivered

    All possible scenarios are:
        - Error path: Shipment does not exist
        - Error path: Invalid transition is rejected
        - Happy path: Shipment got delivered

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Shipment does not exist
        When I deliver shipment "missing"
        Then I will get this error "envio não encontrado"

    Scenario: Invalid transition is rejected
        Given I have a shipment "s1" for user "u1" and order "o1" with status "processing"
        When I deliver shipment "s1"
        Then I will get this error "transição de estado inválida"

    Scenario: Shipment got delivered
        Given I have a shipment "s1" for user "u1" and order "o1" with status "shipped"
        When I deliver shipment "s1"
        Then the shipment "s1" will have status "delivered"
