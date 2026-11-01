Feature: ChangePassword rotates an account's password

    Verifies the current password, requires the new password and its confirmation
    to match, and stores a fresh bcrypt hash. The account is resolved from the
    authenticated access token, so a caller can only change its own password.

    All possible scenarios are:
        - Error path: Blank current password is rejected
        - Error path: Incorrect current password is rejected
        - Error path: New password and confirmation do not match
        - Error path: New password too short (< 6) is rejected
        - Error path: New password too long (> 72) is rejected
        - Happy path: Password is rehashed and the new password authenticates
        - Edge path: Unknown account is rejected

    Background: Clear the storage
        Given I have empty auth storage

    Scenario: Incorrect current password is rejected
        Given I have an account with username "john" and password "secret123"
        When I change the password with current password "wrong" to "newsecret"
        Then I will get this error "auth: senha atual incorreta"

    Scenario: New password and confirmation do not match
        Given I have an account with username "john" and password "secret123"
        When I change the password with current password "secret123", new password "newsecret" and confirmation "different"
        Then I will get this error "auth: a nova senha e a confirmação não coincidem"

    Scenario: New password too short is rejected
        Given I have an account with username "john" and password "secret123"
        When I change the password with current password "secret123" to "abc"
        Then I will get this error "auth: a senha deve ter pelo menos 6 caracteres"

    Scenario: New password too long is rejected
        Given I have an account with username "john" and password "secret123"
        When I change the password with current password "secret123" to "a" repeated 73 times
        Then I will get this error "auth: a senha deve ter no máximo 72 caracteres"

    Scenario: Password is rehashed and the new password authenticates
        Given I have an account with username "john" and password "secret123"
        When I change the password with current password "secret123" to "newsecret"
        Then I will be able to authenticate with password "newsecret"
        And authenticating with password "secret123" will fail

    Scenario: Unknown account is rejected
        When I change the password for an unknown account
        Then I will get this error "auth: conta não encontrada"
