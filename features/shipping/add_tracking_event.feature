Feature: AddTrackingEvent records a tracking event for a shipment

    All possible scenarios are:
        - Error path: Shipment does not exist
        - Error path: Blank description is rejected
        - Happy path: Tracking event got created

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Shipment does not exist
        When I add a tracking event to shipment "missing" with status "shipped" and description "Objeto enviado" and location "Sao Paulo"
        Then I will get this error "envio não encontrado"

    Scenario: Blank description is rejected
        Given I have a shipment "s1" for user "u1" and order "o1"
        When I add a tracking event to shipment "s1" with status "shipped" and description "" and location "Sao Paulo"
        Then I will get this error "transição de estado inválida"

    Scenario: Tracking event got created
        Given I have a shipment "s1" for user "u1" and order "o1"
        When I add a tracking event to shipment "s1" with status "shipped" and description "Objeto enviado" and location "Sao Paulo"
        Then I will have a tracking event for shipment "s1" with status "shipped" and description "Objeto enviado" and location "Sao Paulo"
