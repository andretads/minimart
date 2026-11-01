package application

import (
	"context"
	"errors"
	"minimart/internal/notification/domain"
	"minimart/pkg/logger"
	"minimart/pkg/metrics"
)

type IngestRequest struct {
	Notification domain.Notification
	Err          error
}

// IngestNotification records a notification and schedules one delivery per
// allowed channel. Notifications that arrive twice under the same dedupe key
// keep their first deliveries.
func (s *Service) IngestNotification(ctx context.Context, n domain.Notification, constructionErr error) (domain.Notification, []domain.Delivery, error) {
	if constructionErr != nil {
		return s.ingestErr(constructionErr)
	}

	n, err := normalizeNotification(n)
	if err != nil {
		return s.ingestErr(err)
	}

	category := domain.CategoryForType(n.Type)
	channels, err := s.resolveChannels(ctx, n, category)
	if err != nil {
		return s.ingestErr(err)
	}
	if len(channels) == 0 {
		n.Status = domain.NotificationStatusSuppressed
		s.m.IncNotificationSuppressed(string(n.Type))
		return n, nil, nil
	}

	n.Status = domain.NotificationStatusPending
	persisted, deliveries, err := s.persistDeliveries(ctx, n, channels)
	if err != nil {
		s.m.IncNotificationSent(metrics.StatusError)
		return persisted, deliveries, err
	}

	s.m.IncNotificationSent(metrics.StatusSuccess)
	return persisted, deliveries, nil
}

// ingestErr records a failed ingest and returns the zero values every early
// error path shares.
func (s *Service) ingestErr(err error) (domain.Notification, []domain.Delivery, error) {
	s.m.IncNotificationSent(metrics.StatusError)
	return domain.Notification{}, nil, err
}

// normalizeNotification validates the constructed notification and fills in
// the defaults a sender may omit.
func normalizeNotification(n domain.Notification) (domain.Notification, error) {
	switch {
	case n.ID == "":
		return n, domain.ErrEmptyID
	case n.UserID == "":
		return n, domain.ErrEmptyUserID
	case n.OrderID == "" && n.DedupeKey == "":
		return n, domain.ErrEmptyOrderID
	}

	if n.RecipientID == "" {
		n.RecipientID = n.UserID
	}
	if n.DedupeKey == "" {
		n.DedupeKey = n.OrderID + "/" + string(n.Type)
	}
	if n.TemplateID == "" {
		n.TemplateID = domain.TemplateIDForType(n.Type)
	}
	if n.Priority == "" {
		n.Priority = domain.PriorityForType(n.Type)
	}
	if n.Status == "" {
		n.Status = domain.NotificationStatusAccepted
	}
	if n.Payload == nil {
		n.Payload = map[string]any{}
	}
	return n, nil
}

// resolveChannels loads the recipient's preferences and contacts and picks the
// channels the notification may use. Critical notifications use every channel
// with a contact, transactional ones fall back to all contacts when the
// preferences store is unavailable, and the rest honor the user's
// preferences.
func (s *Service) resolveChannels(ctx context.Context, n domain.Notification, category domain.Category) ([]domain.Channel, error) {
	critical := category == domain.CategoryCritical

	prefs, prefErr := s.GetPreferences(ctx, n.UserID)
	if prefErr != nil && !errors.Is(prefErr, domain.ErrPreferencesNotFound) {
		if category != domain.CategoryTransactional && !critical {
			return nil, prefErr
		}
		logger.FromContext(ctx).Warn("notification preferences unavailable; sending anyway",
			"error", prefErr, "type", string(n.Type), "user_id", n.UserID)
	}
	if errors.Is(prefErr, domain.ErrPreferencesNotFound) {
		prefs = domain.DefaultPreferences(n.UserID)
	}

	email, err := s.lookupContact(ctx, n.UserID, s.recipients.EmailForUser)
	if err != nil {
		return nil, err
	}
	phone, err := s.lookupContact(ctx, n.UserID, s.recipients.PhoneForUser)
	if err != nil {
		return nil, err
	}
	push, err := s.lookupContact(ctx, n.UserID, s.recipients.PushTokenForUser)
	if err != nil {
		return nil, err
	}

	contacts := map[domain.Channel]string{
		domain.ChannelEmail: email,
		domain.ChannelSMS:   phone,
		domain.ChannelPush:  push,
	}

	channels := make([]domain.Channel, 0)
	for _, ch := range domain.AllChannels {
		if contacts[ch] == "" {
			continue
		}
		if critical ||
			(prefErr != nil && category == domain.CategoryTransactional) ||
			prefs.Allows(ch, category) {
			channels = append(channels, ch)
		}
	}
	return channels, nil
}

// lookupContact resolves one contact channel, treating a missing contact as
// absent instead of failing the ingest.
func (s *Service) lookupContact(ctx context.Context, userID string, lookup func(context.Context, string) (string, error)) (string, error) {
	contact, err := lookup(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrRecipientNotFound) {
		return "", err
	}
	return contact, nil
}

// persistDeliveries saves the notification and creates one delivery per
// channel. A notification already stored under the same dedupe key reuses its
// existing deliveries instead of creating new ones.
func (s *Service) persistDeliveries(ctx context.Context, n domain.Notification, channels []domain.Channel) (domain.Notification, []domain.Delivery, error) {
	if err := s.repo.Save(ctx, n); err != nil {
		return domain.Notification{}, nil, err
	}
	persisted, err := s.repo.GetByDedupeKey(ctx, n.DedupeKey)
	if err != nil {
		return domain.Notification{}, nil, err
	}

	existing, err := s.deliveries.ListByNotification(ctx, persisted.ID)
	if err != nil {
		return persisted, nil, err
	}
	if len(existing) > 0 {
		return persisted, existing, nil
	}

	deliveries := make([]domain.Delivery, 0, len(channels))
	for _, ch := range channels {
		d, err := domain.NewDelivery(s.newDeliveryID(), persisted.ID, persisted.RecipientID, ch, providerFor(ch))
		if err != nil {
			return persisted, deliveries, err
		}
		if err := s.deliveries.SaveDelivery(ctx, d); err != nil {
			return persisted, deliveries, err
		}
		deliveries = append(deliveries, d)
	}
	return persisted, deliveries, nil
}

func (s *Service) newDeliveryID() string { return newID() }

func providerFor(ch domain.Channel) string {
	switch ch {
	case domain.ChannelEmail:
		return "resend"
	case domain.ChannelSMS:
		return "log"
	case domain.ChannelPush:
		return "log"
	}
	return "log"
}
