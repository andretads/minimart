Feature: SignUp creates a new user account

    Creates an account with username and password, assigns the default "customer" role,
    and provisions a customer record. Customer provisioning and role assignment are
    best-effort: failures only log a warning, the account is still returned.

    All possible scenarios are:
        - Error path: Blank username is rejected
        - Error path: Blank password is rejected
        - Error path: Password too short (< 6) is rejected
        - Error path: Password too long (> 72) is rejected
        - Error path: Duplicate username is rejected
        - Happy path: Account created with generated ID and hashed password
        - Edge path: Customer provisioning failure does not fail signup
        - Edge path: Storage failure surfaces the error

    Background: Clear the storage
        Given I have empty auth storage

    Scenario: Blank username is rejected
        When I sign up with username "" and password "secret123"
        Then I will get this error "auth: nome de usuário é obrigatório"

    Scenario: Blank password is rejected
        When I sign up with username "john" and password ""
        Then I will get this error "auth: senha é obrigatória"

    Scenario: Password too short is rejected
        When I sign up with username "john" and password "abc"
        Then I will get this error "auth: a senha deve ter pelo menos 6 caracteres"

    Scenario: Password too long is rejected
        When I sign up with username "john" and password "a" repeated 73 times
        Then I will get this error "auth: a senha deve ter no máximo 72 caracteres"

    Scenario: Duplicate username is rejected
        Given I have an account with username "john"
        When I sign up with username "john" and password "secret123"
        Then I will get this error "auth: conta já existe"

    Scenario: Account created with generated ID and hashed password
        When I sign up with username "john" and password "secret123"
        Then I will have an account with a generated ID, username "john", and a password hash
        And the account has the "customer" role assigned

    Scenario: Customer provisioning failure does not fail signup
        Given the customer service is unavailable
        When I sign up with username "john" and password "secret123"
        Then I will have an account with username "john"
        And a warning is logged for customer provisioning failure

    Scenario: Storage failure surfaces the error
        Given the storage is failing
        When I sign up with username "john" and password "secret123"
        Then I will get this error "disk on fire"