package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	cookieEncryptionKeyLength = 32
	nonceSize                 = 12
)

var (
	ErrInvalidEncryptionKey = errors.New("auth: encryption key must be 32 bytes")
	ErrDecryptionFailed     = errors.New("auth: failed to decrypt cookie")
)

const (
	AccessTokenCookieName  = "mm_at"
	RefreshTokenCookieName = "mm_rt"
)

type CookieEncryption struct {
	aead cipher.AEAD
}

func NewCookieEncryption(key []byte) (*CookieEncryption, error) {
	if len(key) != cookieEncryptionKeyLength {
		return nil, ErrInvalidEncryptionKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return &CookieEncryption{aead: aead}, nil
}

func (c *CookieEncryption) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func (c *CookieEncryption) Decrypt(encrypted string) ([]byte, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	if len(ciphertext) < nonceSize {
		return nil, ErrDecryptionFailed
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	return plaintext, nil
}

type CookieConfig struct {
	Domain     string
	Secure     bool
	SameSite   http.SameSite
	Path       string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func DefaultCookieConfig(accessTTL, refreshTTL time.Duration) CookieConfig {
	return CookieConfig{
		Domain:     "",
		Secure:     false,
		SameSite:   http.SameSiteLaxMode,
		Path:       "/",
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
	}
}

func ProductionCookieConfig(accessTTL, refreshTTL time.Duration, domain string) CookieConfig {
	return CookieConfig{
		Domain:     domain,
		Secure:     true,
		SameSite:   http.SameSiteLaxMode,
		Path:       "/",
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
	}
}

func (c CookieConfig) SetCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // attributes are copied from CookieConfig
		Name:     name,
		Value:    value,
		Path:     c.Path,
		Domain:   c.Domain,
		MaxAge:   int(ttl.Seconds()),
		Secure:   c.Secure,
		HttpOnly: true,
		SameSite: c.SameSite,
	})
}

func (c CookieConfig) ClearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // attributes are copied from CookieConfig
		Name:     name,
		Value:    "",
		Path:     c.Path,
		Domain:   c.Domain,
		MaxAge:   -1,
		Secure:   c.Secure,
		HttpOnly: true,
		SameSite: c.SameSite,
	})
}
