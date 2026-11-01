Feature: ListShipments returns shipments

    All possible scenarios are:
        - Happy path: Shipments are returned newest first
        - Happy path: Empty list is returned

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Shipments are returned newest first
        Given I have a shipment "s1" for user "u1" and order "o1"
        When I list shipments for user "u1"
        Then I will get 1 shipment

    Scenario: Empty list is returned
        When I list shipments for user "u1"
        Then I will get 0 shipments
