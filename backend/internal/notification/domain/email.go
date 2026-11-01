package domain

import (
	"fmt"
	"html"
	"strings"
)

type Email struct {
	To             string
	Subject        string
	Text           string
	HTML           string
	IdempotencyKey string
}

func NewEmail(to, subject, text string) (Email, error) {
	if strings.TrimSpace(to) == "" {
		return Email{}, ErrEmptyRecipient
	}
	htmlBody := html.EscapeString(text)
	htmlBody = strings.ReplaceAll(htmlBody, "\n", "<br>")
	return Email{
		To:      strings.TrimSpace(to),
		Subject: subject,
		Text:    text,
		HTML:    fmt.Sprintf("<pre>%s</pre>", htmlBody),
	}, nil
}
