//go:build integration

package test

import (
	"context"
	"minimart/internal/notification/domain"
	"sync"
)

type fakeEmailSender struct {
	mu   sync.Mutex
	sent []domain.Email
}

func newFakeEmailSender() *fakeEmailSender {
	return &fakeEmailSender{sent: make([]domain.Email, 0)}
}

func (f *fakeEmailSender) Send(ctx context.Context, e domain.Email) (domain.Receipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, e)
	return domain.Receipt{ProviderMessageID: "fake", Delivered: false}, nil
}

type fakeRecipientResolver struct {
	mu     sync.Mutex
	emails map[string]string
}

func newFakeRecipientResolver() *fakeRecipientResolver {
	return &fakeRecipientResolver{emails: make(map[string]string)}
}

func (f *fakeRecipientResolver) EmailForUser(ctx context.Context, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if email, ok := f.emails[userID]; ok {
		return email, nil
	}
	return "", domain.ErrRecipientNotFound
}

func (f *fakeRecipientResolver) PhoneForUser(ctx context.Context, userID string) (string, error) {
	return "", domain.ErrRecipientNotFound
}

func (f *fakeRecipientResolver) PushTokenForUser(ctx context.Context, userID string) (string, error) {
	return "", domain.ErrRecipientNotFound
}

func (f *fakeRecipientResolver) SetEmail(userID, email string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.emails[userID] = email
}
