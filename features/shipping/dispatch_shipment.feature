Feature: DispatchShipment moves a processing shipment to shipped

    All possible scenarios are:
        - Error path: Shipment does not exist
        - Error path: Invalid transition is rejected
        - Error path: Missing tracking code is rejected
        - Error path: Missing carrier is rejected
        - Happy path: Shipment got dispatched

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Shipment does not exist
        When I dispatch shipment "missing" with tracking code "BR123" and carrier "Correios"
        Then I will get this error "envio não encontrado"

    Scenario: Invalid transition is rejected
        Given I have a shipment "s1" for user "u1" and order "o1" with status "pending"
        When I dispatch shipment "s1" with tracking code "BR123" and carrier "Correios"
        Then I will get this error "transição de estado inválida"

    Scenario: Missing tracking code is rejected
        Given I have a shipment "s1" for user "u1" and order "o1" with status "processing"
        When I dispatch shipment "s1" with tracking code "" and carrier "Correios"
        Then I will get this error "transição de estado inválida"

    Scenario: Missing carrier is rejected
        Given I have a shipment "s1" for user "u1" and order "o1" with status "processing"
        When I dispatch shipment "s1" with tracking code "BR123" and carrier ""
        Then I will get this error "transição de estado inválida"

    Scenario: Shipment got dispatched
        Given I have a shipment "s1" for user "u1" and order "o1" with status "processing"
        When I dispatch shipment "s1" with tracking code "BR123" and carrier "Correios"
        Then the shipment "s1" will have status "shipped" and tracking code "BR123" and carrier "Correios"
