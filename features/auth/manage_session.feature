Feature: Manage the browser authentication session

    Login and refresh return the access-token expiry so the SPA can refresh
    proactively. Refresh renews both tokens (sliding session); logout is public
    and always clears cookies.

    All possible scenarios are:
        - Happy path: Login returns the access-token expiry
        - Happy path: Refresh renews both cookies and returns the new expiry
        - Edge path: Remember-me keeps the longer refresh lifetime after refresh
        - Error path: Refresh without a valid refresh token returns 401
        - Edge path: Logout clears cookies without a valid access token

    Background: I have an account with username "john" and password "secret123"

    Scenario: Login returns the access-token expiry
        When I log in with username "john" and password "secret123"
        Then the response includes access_token_expires_at and expires_in
        And the response sets mm_at and mm_rt cookies

    Scenario: Refresh renews both cookies and returns the new expiry
        Given I am logged in
        When I refresh the session
        Then the response includes access_token_expires_at and expires_in
        And the response sets a new mm_at cookie
        And the response sets a new mm_rt cookie

    Scenario: Remember-me keeps the longer refresh lifetime after refresh
        When I log in with username "john", password "secret123", and remember_me true
        And I refresh the session
        Then the renewed mm_rt cookie keeps the remember-me lifetime

    Scenario: Refresh without a valid refresh token returns 401
        When I refresh the session without an mm_rt cookie
        Then I will get this error "invalid refresh token"

    Scenario: Logout clears cookies without a valid access token
        When I log out without an mm_at cookie
        Then the response clears mm_at and mm_rt cookies
