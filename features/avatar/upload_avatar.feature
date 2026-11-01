Feature: UploadAvatar uploads or replaces an avatar image

    All possible scenarios are:
        - Error path: Empty account ID is rejected
        - Error path: Image is required
        - Error path: Invalid image format is rejected
        - Error path: Invalid image size is rejected
        - Error path: Image too large is rejected
        - Error path: Upload failure propagates
        - Happy path: Image uploaded and metadata saved
        - Happy path: Existing avatar overwritten with version bump
        - Happy path: Object key follows the avatars/{account_id}/avatar convention
        - Happy path: Supported image types are jpeg, png, webp, gif

    Background: Clear the storage
        Given I have empty storage

    Scenario: Empty account ID is rejected
        When I upload an avatar for account "" with image type "image/jpeg" and size 100
        Then I will get this error "avatar: id da conta é obrigatório"

    Scenario: Image is required
        When I upload an avatar for account "acct-1" without an image
        Then I will get this error "avatar: imagem é obrigatória"

    Scenario: Invalid image format is rejected
        When I upload an avatar for account "acct-1" with content_type "text/plain" and size 100
        Then I will get this error "avatar: formato de imagem inválido, deve ser uma imagem"

    Scenario: Invalid image size is rejected
        When I upload an avatar for account "acct-1" with content_type "image/jpeg" and size 0
        Then I will get this error "avatar: tamanho de imagem inválido"

    Scenario: Image too large is rejected
        When I upload an avatar for account "acct-1" with content_type "image/jpeg" and size 6291456
        Then I will get this error "avatar: imagem excede o tamanho máximo permitido"

    Scenario: Upload failure propagates
        Given the S3 upload fails
        When I upload an avatar for account "acct-1" with content_type "image/jpeg" and size 100
        Then I will get an error containing "upload avatar"

    Scenario: Image uploaded and metadata saved
        When I upload an avatar for account "acct-1" with content_type "image/jpeg" and size 100
        Then I will have an avatar view with account_id "acct-1", object_key "avatars/acct-1/avatar", bucket "test-bucket", content_type "image/jpeg", size 100, version 1 and a presigned URL

    Scenario: Existing avatar overwritten with version bump
        Given I have an avatar for account "acct-1" with content_type "image/jpeg" and size 100
        When I upload an avatar for account "acct-1" with content_type "image/png" and size 200
        Then I will have an avatar view with account_id "acct-1", content_type "image/png", size 200, version 2

    Scenario: Object key follows the avatars/{account_id}/avatar convention
        When I upload an avatar for account "acct-2" with content_type "image/webp" and size 50
        Then the stored object key will be "avatars/acct-2/avatar"

    Scenario: Supported image types
        When I upload an avatar for account "acct-3" with content_type "image/gif" and size 50
        Then I will have an avatar view with content_type "image/gif"