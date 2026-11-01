Feature: GetPreferences for a user

    All possible scenarios are:
        - Error path: Empty user_id is rejected
        - Error path: Preferences storage failure is propagated
        - Happy path: A user with no saved preferences gets the defaults
        - Happy path: Saved preferences are returned unchanged

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty user_id is rejected
        When I get the notification preferences for user ""
        Then I will get this error "id do usuário da notificação não pode ser vazio"

    Scenario: Preferences storage failure is propagated
        Given the notification preferences storage is failing
        When I get the notification preferences for user "u1"
        Then I will get this error "mongo down"

    Scenario: A user with no saved preferences gets the defaults
        When I get the notification preferences for user "u1"
        Then email notifications will be enabled
        And SMS notifications will be disabled
        And push notifications will be disabled
        And marketing notifications will be disabled
        And the preference update timestamp will be absent

    Scenario: Saved preferences are returned unchanged
        Given user "u1" has preferences with email "off", SMS "on", push "off" and marketing "on"
        When I get the notification preferences for user "u1"
        Then email notifications will be disabled
        And SMS notifications will be enabled
        And push notifications will be disabled
        And marketing notifications will be enabled
        And the preference update timestamp will be present
