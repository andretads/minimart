Feature: GetAvatar retrieves an avatar by account ID

    All possible scenarios are:
        - Error path: Empty account ID is rejected
        - Error path: Avatar not found
        - Happy path: Avatar returned with presigned URL
        - Error path: Presign failure propagates

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty account ID is rejected
        When I get the avatar for account ""
        Then I will get this error "avatar: id da conta é obrigatório"

    Scenario: Avatar not found
        When I get the avatar for account "missing-account"
        Then I will get this error "avatar: avatar não encontrado"

    Scenario: Avatar returned with presigned URL
        Given I have an avatar for account "acct-1" with content_type "image/jpeg" and size 100
        When I get the avatar for account "acct-1"
        Then I will have an avatar view with account_id "acct-1", content_type "image/jpeg", size 100 and a presigned URL

    Scenario: Presign failure propagates
        Given I have an avatar for account "acct-1"
        And presigning fails
        When I get the avatar for account "acct-1"
        Then I will get an error