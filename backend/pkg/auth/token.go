package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	claimsContextKey contextKey = "auth_claims"
)

type Claims struct {
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	TokenType  string `json:"token_type"`
	RememberMe bool   `json:"remember_me,omitempty"`
	jwt.RegisteredClaims
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

var (
	ErrInvalidToken = errors.New("auth: invalid token")
	ErrExpiredToken = errors.New("auth: expired token")
)

type TokenService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	issuer     string
}

func NewTokenService(secret string, accessTTL, refreshTTL time.Duration) (*TokenService, error) {
	if secret == "" {
		return nil, errors.New("auth: JWT secret is required")
	}
	return &TokenService{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		issuer:     "minimart",
	}, nil
}

func (s *TokenService) GenerateAccessToken(userID, username string) (string, error) {
	token, _, err := s.GenerateAccessTokenWithExpiry(userID, username)
	return token, err
}

// GenerateAccessTokenWithExpiry issues an access token and returns its exact
// expiry so handlers can report it to clients without reparsing the JWT.
func (s *TokenService) GenerateAccessTokenWithExpiry(userID, username string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.accessTTL)
	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: "access",
		Issuer:    s.issuer,
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		NotBefore: jwt.NewNumericDate(now),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func (s *TokenService) GenerateRefreshToken(userID, username string) (string, error) {
	return s.generateRefreshToken(userID, username, s.refreshTTL, false)
}

func (s *TokenService) GenerateRefreshTokenWithTTL(userID, username string, ttl time.Duration) (string, error) {
	return s.generateRefreshToken(userID, username, ttl, false)
}

// GenerateRefreshTokenForSession issues a refresh token carrying the
// remember-me marker so a later refresh can renew the session with the same
// policy.
func (s *TokenService) GenerateRefreshTokenForSession(userID, username string, ttl time.Duration, rememberMe bool) (string, error) {
	return s.generateRefreshToken(userID, username, ttl, rememberMe)
}

func (s *TokenService) generateRefreshToken(userID, username string, ttl time.Duration, rememberMe bool) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:     userID,
		Username:   username,
		TokenType:  "refresh",
		RememberMe: rememberMe,
		Issuer:     s.issuer,
		Subject:    userID,
		IssuedAt:   jwt.NewNumericDate(now),
		ExpiresAt:  jwt.NewNumericDate(now.Add(ttl)),
		NotBefore:  jwt.NewNumericDate(now),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, "access")
}

func (s *TokenService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return s.validateToken(tokenString, "refresh")
}

func (s *TokenService) AccessTTL() time.Duration {
	return s.accessTTL
}

func (s *TokenService) RefreshTTL() time.Duration {
	return s.refreshTTL
}

func (s *TokenService) validateToken(tokenString, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected algorithm: %v", token.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}), jwt.WithExpirationRequired(), jwt.WithIssuer(s.issuer))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != expectedType {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.UserID, true
}

func UsernameFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.Username, true
}

func AccessTokenExpiresAtFromContext(ctx context.Context) (time.Time, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims.ExpiresAt == nil {
		return time.Time{}, false
	}
	return claims.ExpiresAt.Time, true
}
