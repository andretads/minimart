package application

import (
	"context"
	"minimart/internal/notification/domain"
	"sync"
)

type fakeEmailSender struct {
	mu      sync.Mutex
	sent    []domain.Email
	sendErr error
}

func newFakeEmailSender() *fakeEmailSender {
	return &fakeEmailSender{sent: make([]domain.Email, 0)}
}

func (f *fakeEmailSender) Send(ctx context.Context, e domain.Email) (domain.Receipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, e)
	if f.sendErr != nil {
		return domain.Receipt{}, f.sendErr
	}
	return domain.Receipt{ProviderMessageID: "fake", Delivered: false}, nil
}

func (f *fakeEmailSender) SetSendErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendErr = err
}

func (f *fakeEmailSender) GetSent() []domain.Email {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]domain.Email(nil), f.sent...)
}

type fakeRecipientResolver struct {
	mu     sync.Mutex
	emails map[string]string
	phones map[string]string
	err    error
}

func newFakeRecipientResolver() *fakeRecipientResolver {
	return &fakeRecipientResolver{emails: make(map[string]string), phones: make(map[string]string)}
}

func (f *fakeRecipientResolver) EmailForUser(ctx context.Context, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return "", f.err
	}
	if email, ok := f.emails[userID]; ok {
		return email, nil
	}
	return "", domain.ErrRecipientNotFound
}

func (f *fakeRecipientResolver) PhoneForUser(ctx context.Context, userID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return "", f.err
	}
	if phone, ok := f.phones[userID]; ok {
		return phone, nil
	}
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

func (f *fakeRecipientResolver) SetPhone(userID, phone string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.phones[userID] = phone
}

func (f *fakeRecipientResolver) SetErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}
