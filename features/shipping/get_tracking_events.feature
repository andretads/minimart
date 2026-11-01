Feature: GetTrackingEvents of a shipment

    The tracking history of a shipment is append-only: the shipping domain
    records an event per transition, and this read returns them oldest first so
    the customer can follow the parcel. The shipment must belong to the
    authenticated user.

    All possible scenarios are:
        - Error path: The caller is not authenticated
        - Error path: Shipment does not exist
        - Error path: The shipment belongs to another user
        - Happy path: The tracking history is returned

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: The caller is not authenticated
        When I get the tracking events of shipment "missing" without a user
        Then I will get this error "não autorizado"

    Scenario: Shipment does not exist
        When I get the tracking events of shipment "missing"
        Then I will get this error "envio não encontrado"

    Scenario: The shipment belongs to another user
        Given I have a shipment "s1" for user "u1" and order "o1"
        When I get the tracking events of shipment "s1" as user "u2"
        Then I will get this error "incompatibilidade de propriedade"

    Scenario: The tracking history is returned
        Given I have a shipment "s1" for user "u1" and order "o1"
        And the shipment "s1" has a tracking event "shipped" described as "Objeto enviado" in "Sao Paulo"
        When I get the tracking events of shipment "s1" as user "u1"
        Then I will get 1 tracking event for shipment "s1"
        And the tracking event "shipped" is described as "Objeto enviado" in "Sao Paulo"
