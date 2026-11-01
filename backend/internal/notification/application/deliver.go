package application

import (
	"context"
	"errors"
	"minimart/internal/notification/domain"
	"minimart/pkg/kafka"
	"minimart/pkg/logger"
	"minimart/pkg/metrics"
	"time"
)

func (s *Service) DeliverNotification(ctx context.Context, deliveryID string) error {
	d, err := s.deliveries.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return err
	}
	if d.IsTerminal() {
		return nil
	}

	n, err := s.repo.GetByID(ctx, d.NotificationID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	d.Status = domain.DeliveryStatusProcessing
	d.AttemptCount++
	d.LastAttemptAt = &now
	d.UpdatedAt = now
	if uErr := s.deliveries.UpdateDeliveryStatus(ctx, d.ID, domain.DeliveryStatusProcessing); uErr != nil {
		return uErr
	}

	rendered, rErr := domain.RenderTemplate(n.TemplateID, d.Channel, n.Subject, n.Body)
	if rErr != nil {
		return s.failDelivery(ctx, d, n, d.AttemptCount, rErr)
	}

	contact, err := s.contactFor(ctx, n.RecipientID, d.Channel)
	if err != nil {
		return s.failDelivery(ctx, d, n, d.AttemptCount, err)
	}

	var receipt domain.Receipt
	var sendErr error
	switch d.Channel {
	case domain.ChannelEmail:
		msg := domain.Email{To: contact, Subject: rendered.Subject, Text: rendered.Text, HTML: rendered.HTML, IdempotencyKey: n.ID + "/" + string(d.Channel)}
		receipt, sendErr = s.sender.Send(ctx, msg)
	case domain.ChannelSMS:
		msg := domain.SMS{To: contact, Text: rendered.Text, IdempotencyKey: n.ID + "/" + string(d.Channel)}
		receipt, sendErr = s.smsSender.Send(ctx, msg)
	case domain.ChannelPush:
		msg := domain.Push{Token: contact, Title: rendered.Title, Body: rendered.Body, IdempotencyKey: n.ID + "/" + string(d.Channel)}
		receipt, sendErr = s.pushSender.Send(ctx, msg)
	default:
		sendErr = domain.ErrInvalidChannel
	}
	if sendErr != nil {
		var perr domain.ProviderError
		if errors.As(sendErr, &perr) && perr.IsTransient() {
			if d.AttemptCount >= s.retry.MaxAttempts {
				return s.failDelivery(ctx, d, n, d.AttemptCount, sendErr)
			}
			delay := s.retryDelay(d.AttemptCount, perr.RetryAfter)
			next := time.Now().UTC().Add(delay)
			if err := s.deliveries.MarkRetry(ctx, d.ID, d.AttemptCount, next, sendErr); err != nil {
				return err
			}
			s.updateNotificationStatus(ctx, n.ID)
			logger.FromContext(ctx).Warn("delivery scheduled for retry",
				"delivery_id", d.ID, "channel", string(d.Channel), "attempt", d.AttemptCount, "delay", delay.String())
			return nil
		}
		return s.failDelivery(ctx, d, n, d.AttemptCount, sendErr)
	}

	deliveredAt := receipt.DeliveredAt
	if receipt.Delivered && deliveredAt == nil {
		now2 := time.Now().UTC()
		deliveredAt = &now2
	}
	if err := s.deliveries.MarkSent(ctx, d.ID, d.Provider, receipt.ProviderMessageID, d.AttemptCount, receipt.Delivered, deliveredAt); err != nil {
		return err
	}
	s.updateNotificationStatus(ctx, n.ID)
	logger.FromContext(ctx).Info("delivery sent",
		"delivery_id", d.ID, "notification_id", n.ID, "channel", string(d.Channel), "provider", d.Provider)
	return nil
}

func (s *Service) contactFor(ctx context.Context, recipientID string, ch domain.Channel) (string, error) {
	switch ch {
	case domain.ChannelEmail:
		return s.recipients.EmailForUser(ctx, recipientID)
	case domain.ChannelSMS:
		return s.recipients.PhoneForUser(ctx, recipientID)
	case domain.ChannelPush:
		return s.recipients.PushTokenForUser(ctx, recipientID)
	}
	return "", domain.ErrInvalidChannel
}

func (s *Service) failDelivery(ctx context.Context, d domain.Delivery, n domain.Notification, attempts int, err error) error {
	if fErr := s.deliveries.MarkFailed(ctx, d.ID, attempts, err); fErr != nil {
		return fErr
	}
	s.updateNotificationStatus(ctx, n.ID)

	logger.FromContext(ctx).Error("delivery failed",
		"delivery_id", d.ID, "notification_id", n.ID, "channel", string(d.Channel), "error", err)
	s.m.IncNotificationSent(metrics.StatusError)
	return kafka.Permanent(err)
}

func (s *Service) retryDelay(attempt int, retryAfter *time.Duration) time.Duration {
	if retryAfter != nil && *retryAfter > 0 {
		if *retryAfter > s.retry.MaxDelay {
			return s.retry.MaxDelay
		}
		return *retryAfter
	}
	d := s.retry.BaseDelay
	for i := 1; i < attempt && d < s.retry.MaxDelay; i++ {
		d *= 2
	}
	if d > s.retry.MaxDelay {
		d = s.retry.MaxDelay
	}
	return d
}

func (s *Service) updateNotificationStatus(ctx context.Context, notificationID string) {
	deliveries, err := s.deliveries.ListByNotification(ctx, notificationID)
	if err != nil {
		logger.FromContext(ctx).Warn("failed to list deliveries for notification status update",
			"error", err, "notification_id", notificationID)
		return
	}
	var status domain.NotificationStatus
	allSent := len(deliveries) > 0
	allDelivered := len(deliveries) > 0
	anyRetry := false
	allFailed := len(deliveries) > 0
	for _, d := range deliveries {
		switch d.Status {
		case domain.DeliveryStatusPending, domain.DeliveryStatusProcessing:
			allSent = false
			allDelivered = false
			allFailed = false
		case domain.DeliveryStatusSent:
			allDelivered = false
			allFailed = false
		case domain.DeliveryStatusDelivered:
			allSent = false
			allFailed = false
		case domain.DeliveryStatusRetryPending:
			anyRetry = true
			allSent = false
			allDelivered = false
			allFailed = false
		case domain.DeliveryStatusFailed:
			allSent = false
			allDelivered = false
		}
	}
	switch {
	case allDelivered:
		status = domain.NotificationStatusDelivered
	case allSent:
		status = domain.NotificationStatusSent
	case anyRetry:
		status = domain.NotificationStatusRetryPending
	case allFailed:
		status = domain.NotificationStatusFailed
	default:
		status = domain.NotificationStatusProcessing
	}
	sentAt := time.Now().UTC()
	var deliveredAt *time.Time
	if status == domain.NotificationStatusDelivered {
		deliveredAt = &sentAt
	}
	if err := s.repo.UpdateStatus(ctx, notificationID, status, &sentAt, deliveredAt); err != nil {
		logger.FromContext(ctx).Warn("failed to update notification status",
			"error", err, "notification_id", notificationID, "status", string(status))
	}
}

func channelTopic(ch domain.Channel) kafka.Topic {
	switch ch {
	case domain.ChannelEmail:
		return kafka.NotificationEmailTopic
	case domain.ChannelSMS:
		return kafka.NotificationSMSTopic
	case domain.ChannelPush:
		return kafka.NotificationPushTopic
	}
	return kafka.Topic("notification-unknown")
}
