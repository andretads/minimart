Feature: ListMethods returns active shipping methods

    All possible scenarios are:
        - Happy path: Active methods are returned
        - Happy path: Inactive methods are excluded

    Background: Clear the storage
        Given I have empty shipping storage

    Scenario: Active methods are returned
        Given I have a shipping method "sm1" named "Standard" that is active
        When I list shipping methods
        Then I will get 1 shipping method

    Scenario: Inactive methods are excluded
        Given I have a shipping method "sm1" named "Standard" that is active
        And I have a shipping method "sm2" named "Express" that is inactive
        When I list shipping methods
        Then I will get 1 shipping method
