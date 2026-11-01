package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmail_Validation(t *testing.T) {
	tests := []struct {
		name    string
		to      string
		subject string
		text    string
		wantErr error
	}{
		{"empty recipient", "", "Subject", "Body", ErrEmptyRecipient},
		{"whitespace-only recipient", "   ", "Subject", "Body", ErrEmptyRecipient},
		{"valid recipient", "user@example.com", "Subject", "Body", nil},
		{"valid recipient with whitespace", "  user@example.com  ", "Subject", "Body", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := NewEmail(tt.to, tt.subject, tt.text)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "user@example.com", e.To)
		})
	}
}

func TestNewEmail_HTMLGeneration(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantHTML string
	}{
		{"plain text", "Hello World", "<pre>Hello World</pre>"},
		{"special characters", "<script>alert(1)</script>", "<pre>&lt;script&gt;alert(1)&lt;/script&gt;</pre>"},
		{"newlines", "Line 1\nLine 2", "<pre>Line 1<br>Line 2</pre>"},
		{"mixed content", "Hi\n<script>", "<pre>Hi<br>&lt;script&gt;</pre>"},
		{"multiple newlines", "A\n\nB", "<pre>A<br><br>B</pre>"},
		{"empty text", "", "<pre></pre>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := NewEmail("user@example.com", "Subject", tt.text)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHTML, e.HTML)
		})
	}
}

func TestNewEmail_FieldPopulation(t *testing.T) {
	e, err := NewEmail("user@example.com", "Test Subject", "Test Body")
	require.NoError(t, err)

	assert.Equal(t, "user@example.com", e.To)
	assert.Equal(t, "Test Subject", e.Subject)
	assert.Equal(t, "Test Body", e.Text)
	assert.Equal(t, "<pre>Test Body</pre>", e.HTML)
	assert.Empty(t, e.IdempotencyKey)
}

func TestNewEmail_TrimWhitespace(t *testing.T) {
	e, err := NewEmail("  user@example.com  ", "  Subject  ", "  Body  ")
	require.NoError(t, err)

	assert.Equal(t, "user@example.com", e.To)
	assert.Equal(t, "  Subject  ", e.Subject)
	assert.Equal(t, "  Body  ", e.Text)
}
