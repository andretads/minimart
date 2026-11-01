Feature: GetMethod of a shipping method

    All possible scenarios are:
        - Error path: Method does not exist
        - Happy path: The method is returned

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Method does not exist
        When I get the shipping method "missing"
        Then I will get this error "método de frete não encontrado"

    Scenario: The method is returned
        Given I have a shipping method "sm1" named "Standard" that is active
        When I get the shipping method "sm1"
        Then I will get the shipping method "sm1" named "Standard"
