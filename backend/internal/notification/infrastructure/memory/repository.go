package memory

import (
	"context"
	"minimart/internal/notification/domain"
	"slices"
	"sync"
	"time"
)

type deliveryKey struct {
	notificationID string
	channel        domain.Channel
}

type Repository struct {
	mu                        sync.RWMutex
	byID                      map[string]domain.Notification
	notificationIDByDedupeKey map[string]string
	byUser                    map[string][]string
	prefs                     map[string]domain.Preferences
	deliveries                map[string]domain.Delivery
	deliveryPairs             map[deliveryKey]string
	byNotif                   map[string][]string
}

var (
	_ domain.NotificationRepository = (*Repository)(nil)
	_ domain.DeliveryRepository     = (*Repository)(nil)
	_ domain.PreferencesRepository  = (*Repository)(nil)
)

func New() *Repository {
	return &Repository{
		byID:                      make(map[string]domain.Notification),
		notificationIDByDedupeKey: make(map[string]string),
		byUser:                    make(map[string][]string),
		prefs:                     make(map[string]domain.Preferences),
		deliveries:                make(map[string]domain.Delivery),
		deliveryPairs:             make(map[deliveryKey]string),
		byNotif:                   make(map[string][]string),
	}
}

func (r *Repository) Save(ctx context.Context, n domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := n.DedupeKey
	if key == "" {
		key = n.OrderID + "/" + string(n.Type)
		n.DedupeKey = key
	}
	if _, ok := r.notificationIDByDedupeKey[key]; ok {
		return nil
	}
	if n.ID == "" {
		return domain.ErrEmptyID
	}
	r.notificationIDByDedupeKey[key] = n.ID
	r.byID[n.ID] = n
	r.byUser[n.UserID] = append(r.byUser[n.UserID], n.ID)
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.byID[id]
	if !ok {
		return domain.Notification{}, domain.ErrNotificationNotFound
	}
	return n, nil
}

func (r *Repository) GetByIDForUser(ctx context.Context, id, userID string) (domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.byID[id]
	if !ok || n.UserID != userID {
		return domain.Notification{}, domain.ErrNotificationNotFound
	}
	return n, nil
}

func (r *Repository) GetByOrderIDAndType(ctx context.Context, orderID string, nType domain.Type) (domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, n := range r.byID {
		if n.OrderID == orderID && n.Type == nType {
			return n, nil
		}
	}
	return domain.Notification{}, domain.ErrNotificationNotFound
}

func (r *Repository) GetByDedupeKey(ctx context.Context, key string) (domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if id, ok := r.notificationIDByDedupeKey[key]; ok {
		if n, ok2 := r.byID[id]; ok2 {
			return n, nil
		}
	}
	return domain.Notification{}, domain.ErrNotificationNotFound
}

func (r *Repository) ListByUserID(ctx context.Context, userID string, page, pageSize int) ([]domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byUser[userID]
	out := make([]domain.Notification, 0, len(ids))
	for i := range slices.Backward(ids) {
		if n, ok := r.byID[ids[i]]; ok {
			out = append(out, n)
		}
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := min((page-1)*pageSize, len(out))
	end := min(start+pageSize, len(out))
	return append([]domain.Notification(nil), out[start:end]...), nil
}

func (r *Repository) CountByUserID(ctx context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byUser[userID]), nil
}

func (r *Repository) MarkRead(ctx context.Context, id, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.byID[id]
	if !ok || n.UserID != userID {
		return domain.ErrNotificationNotFound
	}
	now := time.Now().UTC()
	n.ReadAt = &now
	r.byID[id] = n
	return nil
}

func (r *Repository) MarkUnread(ctx context.Context, id, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.byID[id]
	if !ok || n.UserID != userID {
		return domain.ErrNotificationNotFound
	}
	n.ReadAt = nil
	r.byID[id] = n
	return nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	for _, id := range r.byUser[userID] {
		if n, ok := r.byID[id]; ok {
			n.ReadAt = &now
			r.byID[id] = n
		}
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.byID[id]
	if !ok || n.UserID != userID {
		return domain.ErrNotificationNotFound
	}
	delete(r.byID, id)
	delete(r.notificationIDByDedupeKey, n.DedupeKey)

	filtered := r.byUser[userID][:0]
	for _, v := range r.byUser[userID] {
		if v != id {
			filtered = append(filtered, v)
		}
	}
	r.byUser[userID] = filtered
	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status domain.NotificationStatus, sentAt, deliveredAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.byID[id]
	if !ok {
		return domain.ErrNotificationNotFound
	}
	n.Status = status
	if sentAt != nil {
		n.SentAt = *sentAt
	}
	if deliveredAt != nil {
		n.DeliveredAt = deliveredAt
	}
	r.byID[id] = n
	return nil
}

func (r *Repository) SaveDelivery(ctx context.Context, d domain.Delivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := deliveryKey{notificationID: d.NotificationID, channel: d.Channel}
	if id, ok := r.deliveryPairs[key]; ok {
		r.deliveries[id] = d
		return nil
	}
	if d.ID == "" {
		return domain.ErrEmptyID
	}
	r.deliveryPairs[key] = d.ID
	r.deliveries[d.ID] = d
	r.byNotif[d.NotificationID] = append(r.byNotif[d.NotificationID], d.ID)
	return nil
}

func (r *Repository) GetDeliveryByID(ctx context.Context, id string) (domain.Delivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.deliveries[id]
	if !ok {
		return domain.Delivery{}, domain.ErrDeliveryNotFound
	}
	return d, nil
}

func (r *Repository) GetByNotificationAndChannel(ctx context.Context, notificationID string, channel domain.Channel) (domain.Delivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := deliveryKey{notificationID: notificationID, channel: channel}
	if id, ok := r.deliveryPairs[key]; ok {
		if d, ok2 := r.deliveries[id]; ok2 {
			return d, nil
		}
	}
	return domain.Delivery{}, domain.ErrDeliveryNotFound
}

func (r *Repository) ListByNotification(ctx context.Context, notificationID string) ([]domain.Delivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Delivery, 0)
	for _, id := range r.byNotif[notificationID] {
		if d, ok := r.deliveries[id]; ok {
			out = append(out, d)
		}
	}
	return out, nil
}

func (r *Repository) ClaimDue(ctx context.Context, workerID string, limit int) ([]domain.Delivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	out := make([]domain.Delivery, 0)
	for _, d := range r.deliveries {
		if len(out) >= limit {
			break
		}
		if d.Status != domain.DeliveryStatusPending && d.Status != domain.DeliveryStatusRetryPending {
			continue
		}
		if d.NextAttemptAt != nil && d.NextAttemptAt.After(now) {
			continue
		}
		if d.LockedUntil != nil && d.LockedUntil.After(now) {
			continue
		}
		d.Status = domain.DeliveryStatusProcessing
		d.LockedBy = workerID
		until := now.Add(30 * time.Second)
		d.LockedUntil = &until
		r.deliveries[d.ID] = d
		out = append(out, d)
	}
	return out, nil
}

func (r *Repository) MarkSent(ctx context.Context, id, provider, providerMessageID string, attemptCount int, delivered bool, deliveredAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[id]
	if !ok {
		return domain.ErrDeliveryNotFound
	}
	now := time.Now().UTC()
	d.AttemptCount = attemptCount
	d.Status = domain.DeliveryStatusSent
	if delivered {
		d.Status = domain.DeliveryStatusDelivered
		d.DeliveredAt = deliveredAt
		if d.DeliveredAt == nil {
			d.DeliveredAt = &now
		}
	}
	d.Provider = provider
	if provider != "" {
		d.Provider = provider
	}
	d.ProviderMessageID = providerMessageID
	d.LastAttemptAt = &now
	d.UpdatedAt = now
	r.deliveries[id] = d
	return nil
}

func (r *Repository) MarkRetry(ctx context.Context, id string, attemptCount int, nextAttemptAt time.Time, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[id]
	if !ok {
		return domain.ErrDeliveryNotFound
	}
	now := time.Now().UTC()
	d.Status = domain.DeliveryStatusRetryPending
	d.LastAttemptAt = &now
	d.NextAttemptAt = &nextAttemptAt
	d.UpdatedAt = now
	if err != nil {
		d.Error = err.Error()
	}
	r.deliveries[id] = d
	return nil
}

func (r *Repository) MarkFailed(ctx context.Context, id string, attemptCount int, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[id]
	if !ok {
		return domain.ErrDeliveryNotFound
	}
	now := time.Now().UTC()
	d.Status = domain.DeliveryStatusFailed
	d.LastAttemptAt = &now
	d.UpdatedAt = now
	if err != nil {
		d.Error = err.Error()
	}
	r.deliveries[id] = d
	return nil
}

func (r *Repository) ReleaseLease(ctx context.Context, id, workerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[id]
	if !ok {
		return domain.ErrDeliveryNotFound
	}
	if d.LockedBy != workerID {
		return nil
	}
	d.LockedBy = ""
	d.LockedUntil = nil
	r.deliveries[id] = d
	return nil
}

func (r *Repository) UpdateDeliveryStatus(ctx context.Context, id string, status domain.DeliveryStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[id]
	if !ok {
		return domain.ErrDeliveryNotFound
	}
	d.Status = status
	d.UpdatedAt = time.Now().UTC()
	r.deliveries[id] = d
	return nil
}

func (r *Repository) GetByUserID(ctx context.Context, userID string) (domain.Preferences, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.prefs[userID]
	if !ok {
		return domain.Preferences{}, domain.ErrPreferencesNotFound
	}
	return p, nil
}

func (r *Repository) Upsert(ctx context.Context, p domain.Preferences) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prefs[p.UserID] = p
	return nil
}
