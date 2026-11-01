Feature: ProcessShipment moves a pending shipment to processing

    All possible scenarios are:
        - Error path: Shipment does not exist
        - Error path: Invalid transition is rejected
        - Happy path: Shipment got processed

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Shipment does not exist
        When I process shipment "missing"
        Then I will get this error "envio não encontrado"

    Scenario: Invalid transition is rejected
        Given I have a shipment "s1" for user "u1" and order "o1" with status "shipped"
        When I process shipment "s1"
        Then I will get this error "transição de estado inválida"

    Scenario: Shipment got processed
        Given I have a shipment "s1" for user "u1" and order "o1" with status "pending"
        When I process shipment "s1"
        Then the shipment "s1" will have status "processing"
