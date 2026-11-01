Feature: UpdatePreferences for a user

    All possible scenarios are:
        - Error path: Empty user_id is rejected
        - Error path: Preferences storage failure is propagated
        - Happy path: All four switches are replaced
        - Happy path: A later update replaces the previous switches

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty user_id is rejected
        When I update the notification preferences for user "" to email "off", SMS "off", push "on" and marketing "on"
        Then I will get this error "id do usuário da notificação não pode ser vazio"

    Scenario: Preferences storage failure is propagated
        Given the notification preferences storage is failing
        When I update the notification preferences for user "u1" to email "off", SMS "off", push "on" and marketing "on"
        Then I will get this error "mongo down"

    Scenario: All four switches are replaced
        When I update the notification preferences for user "u1" to email "off", SMS "off", push "on" and marketing "on"
        Then email notifications will be disabled
        And SMS notifications will be disabled
        And push notifications will be enabled
        And marketing notifications will be enabled
        And the preference update timestamp will be present

    Scenario: A later update replaces the previous switches
        When I update the notification preferences for user "u1" to email "on", SMS "on", push "on" and marketing "on"
        And I update the notification preferences for user "u1" to email "off", SMS "off", push "off" and marketing "off"
        Then email notifications will be disabled
        And SMS notifications will be disabled
        And push notifications will be disabled
        And marketing notifications will be disabled
