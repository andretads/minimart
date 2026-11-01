Feature: UpdatePreferences for a user

    All possible scenarios are:
        - Error path: Empty user_id is rejected
        - Error path: Preferences storage failure is propagated
        - Error path: An incomplete update is rejected
        - Happy path: All four switches are replaced
        - Happy path: A later update replaces the previous switches

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty user_id is rejected
        When I update the notification preferences for user "" to order updates "off", promotions "off", security alerts "on" and newsletter "on"
        Then I will get this error "id do usuário da notificação não pode ser vazio"

    Scenario: Preferences storage failure is propagated
        Given the notification preferences storage is failing
        When I update the notification preferences for user "u1" to order updates "off", promotions "off", security alerts "on" and newsletter "on"
        Then I will get this error "mongo down"

    Scenario: An incomplete update is rejected
        When I update the notification preferences for user "u1" with only order updates "off"
        Then I will get this error "todos os campos de preferência são obrigatórios"

    Scenario: All four switches are replaced
        When I update the notification preferences for user "u1" to order updates "off", promotions "off", security alerts "on" and newsletter "on"
        Then order updates will be disabled
        And promotions will be disabled
        And security alerts will be enabled
        And the newsletter will be enabled
        And the preference update timestamp will be present

    Scenario: A later update replaces the previous switches
        When I update the notification preferences for user "u1" to order updates "on", promotions "on", security alerts "on" and newsletter "on"
        And I update the notification preferences for user "u1" to order updates "off", promotions "off", security alerts "off" and newsletter "off"
        Then order updates will be disabled
        And promotions will be disabled
        And security alerts will be disabled
        And the newsletter will be disabled
