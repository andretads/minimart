package auth

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenService(t *testing.T) {
	svc, err := NewTokenService("test-secret", 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("generates valid access token", func(t *testing.T) {
		token, err := svc.GenerateAccessToken("user-1", "testuser")
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := svc.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user-1", claims.UserID)
		assert.Equal(t, "testuser", claims.Username)
		assert.Equal(t, "access", claims.TokenType)
	})

	t.Run("generates valid refresh token", func(t *testing.T) {
		token, err := svc.GenerateRefreshToken("user-1", "testuser")
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := svc.ValidateRefreshToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user-1", claims.UserID)
		assert.Equal(t, "testuser", claims.Username)
		assert.Equal(t, "refresh", claims.TokenType)
	})

	t.Run("remember-me refresh token keeps the marker", func(t *testing.T) {
		token, err := svc.GenerateRefreshTokenForSession("user-1", "testuser", 30*24*time.Hour, true)
		require.NoError(t, err)

		claims, err := svc.ValidateRefreshToken(token)
		require.NoError(t, err)
		assert.True(t, claims.RememberMe)
	})

	t.Run("access token expiry is returned with the token", func(t *testing.T) {
		token, expiresAt, err := svc.GenerateAccessTokenWithExpiry("user-1", "testuser")
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.WithinDuration(t, time.Now().Add(15*time.Minute), expiresAt, time.Second)

		claims, err := svc.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, expiresAt.Unix(), claims.ExpiresAt.Unix())
	})

	t.Run("rejects refresh token as access token", func(t *testing.T) {
		refresh, _ := svc.GenerateRefreshToken("user-1", "testuser")
		_, err := svc.ValidateAccessToken(refresh)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("rejects access token as refresh token", func(t *testing.T) {
		access, _ := svc.GenerateAccessToken("user-1", "testuser")
		_, err := svc.ValidateRefreshToken(access)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("rejects token with wrong secret", func(t *testing.T) {
		other, _ := NewTokenService("wrong-secret", 15*time.Minute, 7*24*time.Hour)
		token, _ := other.GenerateAccessToken("user-1", "testuser")
		_, err := svc.ValidateAccessToken(token)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			short, _ := NewTokenService("test-secret", 1*time.Second, 7*24*time.Hour)
			token, _ := short.GenerateAccessToken("user-1", "testuser")
			// Advance fake time past expiry
			time.Sleep(2 * time.Second)
			_, err := svc.ValidateAccessToken(token)
			assert.ErrorIs(t, err, ErrExpiredToken)
		})
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		_, err := svc.ValidateAccessToken("not.a.token")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("context round-trip", func(t *testing.T) {
		claims, _ := svc.ValidateAccessToken(mustGenerateAccess(t, svc, "user-1", "testuser"))
		ctx := WithClaims(t.Context(), claims)

		got, ok := ClaimsFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, claims.UserID, got.UserID)

		uid, ok := UserIDFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "user-1", uid)

		uname, ok := UsernameFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "testuser", uname)
	})
}

func mustGenerateAccess(t *testing.T, svc *TokenService, userID, username string) string {
	t.Helper()
	token, err := svc.GenerateAccessToken(userID, username)
	require.NoError(t, err)
	return token
}
