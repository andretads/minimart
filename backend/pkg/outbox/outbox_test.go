package outbox

import (
	"context"
	"errors"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// fakeStore is an in-memory outbox.Store. It records the calls it receives so
// tests can assert the dispatcher's locking and cleanup behaviour.
type fakeStore struct {
	mu      sync.Mutex
	records map[string]Record
	order   []string

	lockErr     error
	clearAllErr error

	lockCalls     []lockCall
	clearedByID   []string
	clearCtxErr   error
	clearAllCalls int
	callOrder     []string
	staleBefore   []time.Time
	deletedBefore []time.Time
}

type lockCall struct {
	lockID   string
	lockedOn time.Time
	state    RecordState
	limit    int
}

func newFakeStore(records ...Record) *fakeStore {
	s := &fakeStore{records: map[string]Record{}}
	for _, r := range records {
		s.records[r.ID] = r
		s.order = append(s.order, r.ID)
	}
	return s
}

func (s *fakeStore) AddRecordTx(_ context.Context, record Record, _ pgx.Tx) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[record.ID]; !ok {
		s.order = append(s.order, record.ID)
	}
	s.records[record.ID] = record
	return nil
}

func (s *fakeStore) GetRecordsByLockID(_ context.Context, lockID string) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Record
	for _, id := range s.order {
		if r := s.records[id]; r.LockID != nil && *r.LockID == lockID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *fakeStore) UpdateRecordLockByState(_ context.Context, lockID string, lockedOn time.Time, state RecordState, limit int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lockCalls = append(s.lockCalls, lockCall{lockID: lockID, lockedOn: lockedOn, state: state, limit: limit})
	s.callOrder = append(s.callOrder, "lock")
	if s.lockErr != nil {
		return s.lockErr
	}
	claimed := 0
	for _, id := range s.order {
		if limit > 0 && claimed >= limit {
			break
		}
		if r := s.records[id]; r.State == state && r.LockID == nil && !s.hasAbandonedPredecessor(r) {
			r.LockID = &lockID
			r.LockedOn = &lockedOn
			s.records[id] = r
			claimed++
		}
	}
	return nil
}

func (s *fakeStore) hasAbandonedPredecessor(r Record) bool {
	for _, id := range s.order {
		other := s.records[id]
		if other.State != MaxAttemptsReached || other.Message.Topic != r.Message.Topic || other.Message.Key != r.Message.Key {
			continue
		}
		if other.CreatedOn.Before(r.CreatedOn) || (other.CreatedOn.Equal(r.CreatedOn) && other.ID < r.ID) {
			return true
		}
	}
	return false
}

func (s *fakeStore) UpdateRecordByID(_ context.Context, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.ID] = record
	return nil
}

func (s *fakeStore) ClearLocksWithDurationBeforeDate(_ context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.staleBefore = append(s.staleBefore, before)
	for _, id := range s.order {
		if r := s.records[id]; r.LockID != nil && r.LockedOn != nil && r.LockedOn.Before(before) {
			r.LockID = nil
			r.LockedOn = nil
			s.records[id] = r
		}
	}
	return nil
}

func (s *fakeStore) ClearLocksByLockID(ctx context.Context, lockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearCtxErr = ctx.Err()
	s.clearedByID = append(s.clearedByID, lockID)
	for _, id := range s.order {
		if r := s.records[id]; r.LockID != nil && *r.LockID == lockID {
			r.LockID = nil
			r.LockedOn = nil
			s.records[id] = r
		}
	}
	return nil
}

func (s *fakeStore) ClearAllLocks(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearAllCalls++
	s.callOrder = append(s.callOrder, "clearAll")
	if s.clearAllErr != nil {
		return s.clearAllErr
	}
	for _, id := range s.order {
		if r := s.records[id]; r.LockID != nil {
			r.LockID = nil
			r.LockedOn = nil
			s.records[id] = r
		}
	}
	return nil
}

func (s *fakeStore) RemoveDeliveredRecordsBeforeDatetime(_ context.Context, expiryTime time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletedBefore = append(s.deletedBefore, expiryTime)
	return nil
}

func (s *fakeStore) GetStats(context.Context) (Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var stats Stats
	for _, id := range s.order {
		r := s.records[id]
		age := max(time.Since(r.CreatedOn), 0)
		switch r.State {
		case PendingDelivery:
			stats.Pending++
			stats.OldestPending = max(stats.OldestPending, age)
		case Delivered:
			stats.Delivered++
			stats.OldestDelivered = max(stats.OldestDelivered, age)
		case MaxAttemptsReached:
			stats.Abandoned++
			stats.OldestAbandoned = max(stats.OldestAbandoned, age)
		}
	}
	return stats, nil
}

func (s *fakeStore) ListAbandonedRecords(_ context.Context, limit int) ([]AbandonedRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.abandonedRecords("", limit), nil
}

func (s *fakeStore) ListAbandonedRecordsByTopic(_ context.Context, topic string, limit int) ([]AbandonedRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.abandonedRecords(topic, limit), nil
}

func (s *fakeStore) abandonedRecords(topic string, limit int) []AbandonedRecord {
	var out []AbandonedRecord
	for _, id := range s.order {
		r := s.records[id]
		if r.State != MaxAttemptsReached || (topic != "" && r.Message.Topic != topic) {
			continue
		}
		if limit > 0 && len(out) >= limit {
			break
		}
		out = append(out, AbandonedRecord{
			ID:        r.ID,
			Topic:     r.Message.Topic,
			Key:       r.Message.Key,
			CreatedOn: r.CreatedOn,
		})
	}
	return out
}

func (s *fakeStore) RequeueAbandonedRecords(_ context.Context, ids []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var requeued []string
	for _, id := range ids {
		r, ok := s.records[id]
		if !ok || r.State != MaxAttemptsReached {
			continue
		}
		r.State = PendingDelivery
		r.NumberOfAttempts = 0
		r.LockID = nil
		r.LockedOn = nil
		r.ProcessedOn = nil
		r.Error = nil
		s.records[id] = r
		requeued = append(requeued, id)
	}
	return requeued, nil
}

func (s *fakeStore) Records() map[string]Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]Record, len(s.records))
	maps.Copy(out, s.records)
	return out
}

func (s *fakeStore) Order() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.order))
	copy(out, s.order)
	return out
}

func (s *fakeStore) LockCalls() []lockCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]lockCall, len(s.lockCalls))
	copy(out, s.lockCalls)
	return out
}

func (s *fakeStore) ClearedByID() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.clearedByID))
	copy(out, s.clearedByID)
	return out
}

func (s *fakeStore) ClearCtxErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clearCtxErr
}

func (s *fakeStore) ClearAllCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clearAllCalls
}

func (s *fakeStore) CallOrder() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.callOrder))
	copy(out, s.callOrder)
	return out
}

func (s *fakeStore) StaleBefore() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]time.Time, len(s.staleBefore))
	copy(out, s.staleBefore)
	return out
}

func (s *fakeStore) DeletedBefore() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]time.Time, len(s.deletedBefore))
	copy(out, s.deletedBefore)
	return out
}

// fakeBroker records every Send attempt (successful or not) and can be made to
// fail.
type fakeBroker struct {
	mu      sync.Mutex
	sent    []Message
	err     error
	failFor map[messageIdentity]error
}

func (b *fakeBroker) Send(_ context.Context, msg Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, msg)
	if err, ok := b.failFor[messageIdentity{topic: msg.Topic, key: msg.Key}]; ok {
		return err
	}
	return b.err
}

func (b *fakeBroker) Sent() []Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Message, len(b.sent))
	copy(out, b.sent)
	return out
}

type fakeRelayLock struct {
	mu sync.Mutex

	acquired      bool
	acquireErr    error
	releaseErr    error
	acquireCalls  int
	releaseCalls  int
	releaseCtxErr error
}

func (l *fakeRelayLock) TryAcquire(context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.acquireCalls++
	if l.acquireErr != nil {
		return false, l.acquireErr
	}
	return l.acquired, nil
}

func (l *fakeRelayLock) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.releaseCalls++
	l.releaseCtxErr = ctx.Err()
	return l.releaseErr
}

var base = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

func TestPublisherSend(t *testing.T) {
	store := newFakeStore()
	p := NewPublisher(store)
	p.now = func() time.Time { return base }

	msg := Message{Key: "k", Body: []byte("payload"), Topic: "orders"}
	require.NoError(t, p.Send(t.Context(), msg, nil))

	require.Len(t, store.Order(), 1)
	rec := store.Records()[store.Order()[0]]
	assert.NotEmpty(t, rec.ID)
	assert.Equal(t, msg, rec.Message)
	assert.Equal(t, PendingDelivery, rec.State)
	assert.Equal(t, base, rec.CreatedOn)
}

func TestProcessRecordsDelivers(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", Message: Message{Key: "k", Topic: "t"}, State: PendingDelivery})
	broker := &fakeBroker{}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, nil)
	p.now = func() time.Time { return base }

	require.NoError(t, p.ProcessRecords(t.Context()))

	require.Len(t, broker.Sent(), 1)
	assert.Equal(t, "k", broker.Sent()[0].Key)
	require.Len(t, store.LockCalls(), 1)
	assert.Equal(t, "m1", store.LockCalls()[0].lockID)
	assert.Equal(t, PendingDelivery, store.LockCalls()[0].state)
	assert.Equal(t, base, store.LockCalls()[0].lockedOn)
	rec := store.Records()["r1"]
	assert.Equal(t, Delivered, rec.State)
	assert.Equal(t, 1, rec.NumberOfAttempts)
	assert.Nil(t, rec.LockID)
	assert.Nil(t, rec.LockedOn)
	require.NotNil(t, rec.ProcessedOn)
	assert.Equal(t, base, *rec.ProcessedOn)
	assert.Contains(t, store.ClearedByID(), "m1", "the dispatcher always releases its locks")
}

func TestProcessRecordsNoPending(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", State: Delivered})
	broker := &fakeBroker{}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, nil)

	require.NoError(t, p.ProcessRecords(t.Context()))
	assert.Empty(t, broker.Sent())
}

func TestProcessRecordsSendFailureSkipsSamePartitionAndRecordsError(t *testing.T) {
	store := newFakeStore(
		Record{ID: "r1", State: PendingDelivery},
		Record{ID: "r2", State: PendingDelivery},
	)
	broker := &fakeBroker{err: errors.New("broker down")}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, nil)
	p.now = func() time.Time { return base }

	err := p.ProcessRecords(t.Context())
	require.Error(t, err)
	assert.Len(t, broker.Sent(), 1, "both rows share the same topic and key, so ordering blocks the second")

	rec := store.Records()["r1"]
	assert.Equal(t, PendingDelivery, rec.State)
	assert.Equal(t, 1, rec.NumberOfAttempts)
	require.NotNil(t, rec.Error)
	assert.Contains(t, *rec.Error, "broker down")
	assert.Nil(t, rec.LockID)
}

func TestProcessRecordsMaxAttemptsReached(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery, NumberOfAttempts: 2})
	broker := &fakeBroker{err: errors.New("boom")}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{MaxSendAttemptsEnabled: true, MaxSendAttempts: 3}, nil)
	p.now = func() time.Time { return base }

	require.Error(t, p.ProcessRecords(t.Context()))
	assert.Equal(t, MaxAttemptsReached, store.Records()["r1"].State)
	assert.Nil(t, store.Records()["r1"].LockID)
}

func TestProcessRecordsAbandonedRowBlocksSuccessorUntilReplay(t *testing.T) {
	first := Message{Key: "a", Topic: "orders"}
	second := Message{Key: "a", Topic: "orders"}
	store := newFakeStore(
		Record{ID: "r1", Message: first, State: PendingDelivery, NumberOfAttempts: 2, CreatedOn: base},
		Record{ID: "r2", Message: second, State: PendingDelivery, CreatedOn: base.Add(time.Second)},
	)
	broker := &fakeBroker{failFor: map[messageIdentity]error{
		{topic: "orders", key: "a"}: errors.New("broker down"),
	}}
	policy := RetryPolicy{MaxSendAttemptsEnabled: true, MaxSendAttempts: 3}
	p := newRecordProcessor(store, broker, "m1", 10, policy, nil)
	p.now = func() time.Time { return base }

	require.Error(t, p.ProcessRecords(t.Context()))
	assert.Equal(t, MaxAttemptsReached, store.Records()["r1"].State)
	assert.Equal(t, PendingDelivery, store.Records()["r2"].State)
	require.Len(t, broker.Sent(), 1)

	broker.failFor = nil
	require.NoError(t, p.ProcessRecords(t.Context()))
	require.Len(t, broker.Sent(), 1, "the abandoned row must keep blocking its successor")

	result, err := NewReplayer(store).Run(t.Context(), ReplayOptions{})
	require.NoError(t, err)
	assert.Equal(t, ReplayResult{Selected: 1, Requeued: 1}, result)
	assert.Equal(t, base, store.Records()["r1"].CreatedOn)

	require.NoError(t, p.ProcessRecords(t.Context()))
	sent := broker.Sent()
	require.Len(t, sent, 3)
	assert.Equal(t, first, sent[1])
	assert.Equal(t, second, sent[2])
	assert.Equal(t, Delivered, store.Records()["r1"].State)
	assert.Equal(t, Delivered, store.Records()["r2"].State)
}

func TestProcessRecordsClearsDeadLocksOnRelayTakeover(t *testing.T) {
	lockedOn := base.Add(-time.Minute)
	dead := "worker-1/dead"
	store := newFakeStore(
		Record{ID: "r1", Message: Message{Key: "k", Topic: "orders"}, State: PendingDelivery, CreatedOn: base, LockID: &dead, LockedOn: &lockedOn},
		Record{ID: "r2", Message: Message{Key: "k", Topic: "orders"}, State: PendingDelivery, CreatedOn: base.Add(time.Second)},
	)
	broker := &fakeBroker{}
	lock := &fakeRelayLock{acquired: true}
	p := newRecordProcessor(store, broker, "worker-2/new", 10, RetryPolicy{}, lock)
	p.now = func() time.Time { return base }

	require.NoError(t, p.ProcessRecords(t.Context()))

	assert.Equal(t, []string{"clearAll", "lock"}, store.CallOrder(), "locks are reclaimed before the takeover claims rows")
	assert.Nil(t, store.Records()["r1"].LockID)
	require.Len(t, broker.Sent(), 2)
	assert.Equal(t, Delivered, store.Records()["r1"].State)
	assert.Equal(t, Delivered, store.Records()["r2"].State)
}

func TestProcessRecordsClearAllLocksFailureReleasesRelayLock(t *testing.T) {
	clearErr := errors.New("db down")
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery})
	store.clearAllErr = clearErr
	broker := &fakeBroker{}
	lock := &fakeRelayLock{acquired: true}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, lock)

	err := p.ProcessRecords(t.Context())
	require.ErrorIs(t, err, clearErr)
	assert.Empty(t, broker.Sent())
	assert.Empty(t, store.LockCalls())
	assert.Equal(t, 1, store.ClearAllCalls())
	assert.Equal(t, 1, lock.releaseCalls)
}

func TestProcessRecordsLockFailure(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery})
	store.lockErr = errors.New("db down")
	p := newRecordProcessor(store, &fakeBroker{}, "m1", 10, RetryPolicy{}, nil)

	require.Error(t, p.ProcessRecords(t.Context()))
	assert.Nil(t, store.Records()["r1"].LockID)
}

func TestProcessRecordsReleasesLocksAfterCancellation(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery})
	p := newRecordProcessor(store, &fakeBroker{}, "m1", 10, RetryPolicy{}, nil)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_ = p.ProcessRecords(ctx)

	assert.Contains(t, store.ClearedByID(), "m1", "the dispatcher always releases its locks")
	assert.NoError(t, store.ClearCtxErr(), "locks are released even when the run context is cancelled")
}

func TestRecordUnlocker(t *testing.T) {
	old := base.Add(-2 * time.Minute)
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery, LockID: new("crashed"), LockedOn: &old})
	u := newRecordUnlocker(store, time.Minute)
	u.now = func() time.Time { return base }

	require.NoError(t, u.UnlockExpiredMessages(t.Context()))
	assert.Nil(t, store.Records()["r1"].LockID)
	require.Len(t, store.StaleBefore(), 1)
	assert.Equal(t, base.Add(-time.Minute), store.StaleBefore()[0])
}

func TestRecordCleaner(t *testing.T) {
	store := newFakeStore()
	c := newRecordCleaner(store, 24*time.Hour)
	c.now = func() time.Time { return base }

	require.NoError(t, c.RemoveExpiredMessages(t.Context()))
	require.Len(t, store.DeletedBefore(), 1)
	assert.Equal(t, base.Add(-24*time.Hour), store.DeletedBefore()[0])
}

func TestDispatcherRunProcessesAndStopsOnCancel(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery})
	broker := &fakeBroker{}
	d := NewDispatcher(store, broker, DispatcherSettings{
		ProcessInterval:           5 * time.Millisecond,
		LockCheckerInterval:       time.Hour,
		CleanupWorkerInterval:     time.Hour,
		MaxLockTimeDuration:       time.Minute,
		MessagesRetentionDuration: time.Hour,
	}, "m1", nil)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		d.Run(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool { return len(broker.Sent()) >= 1 }, time.Second, 5*time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatcher did not stop when its context was cancelled")
	}
}

func TestDispatchersSharingAMachineIDUseDistinctLockIDs(t *testing.T) {
	store := newFakeStore()
	broker := &fakeBroker{}
	settings := DispatcherSettings{
		ProcessInterval:           time.Millisecond,
		LockCheckerInterval:       time.Hour,
		CleanupWorkerInterval:     time.Hour,
		MaxLockTimeDuration:       time.Minute,
		MessagesRetentionDuration: time.Hour,
	}

	first := NewDispatcher(store, broker, settings, "worker-1", nil)
	second := NewDispatcher(store, broker, settings, "worker-1", nil)

	firstProcessor, ok := first.recordProcessor.(*recordProcessor)
	require.True(t, ok)
	secondProcessor, ok := second.recordProcessor.(*recordProcessor)
	require.True(t, ok)

	assert.NotEqual(t, firstProcessor.machineID, secondProcessor.machineID)
	assert.Contains(t, firstProcessor.machineID, "worker-1/")
	assert.Contains(t, secondProcessor.machineID, "worker-1/")
}

func TestProcessRecordsFailureOnlyBlocksSamePartition(t *testing.T) {
	store := newFakeStore(
		Record{ID: "r1", Message: Message{Key: "a", Topic: "orders"}, State: PendingDelivery},
		Record{ID: "r2", Message: Message{Key: "a", Topic: "orders"}, State: PendingDelivery},
		Record{ID: "r3", Message: Message{Key: "b", Topic: "orders"}, State: PendingDelivery},
		Record{ID: "r4", Message: Message{Key: "a", Topic: "payments"}, State: PendingDelivery},
	)
	broker := &fakeBroker{failFor: map[messageIdentity]error{
		{topic: "orders", key: "a"}: errors.New("broker down"),
	}}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, nil)
	p.now = func() time.Time { return base }

	require.Error(t, p.ProcessRecords(t.Context()))

	sent := broker.Sent()
	require.Len(t, sent, 3, "only later rows in the failed partition are skipped")
	assert.Equal(t, Message{Key: "a", Topic: "orders"}, sent[0])
	assert.Equal(t, Message{Key: "b", Topic: "orders"}, sent[1])
	assert.Equal(t, Message{Key: "a", Topic: "payments"}, sent[2])

	records := store.Records()
	assert.Equal(t, PendingDelivery, records["r1"].State)
	assert.Equal(t, 1, records["r1"].NumberOfAttempts)
	require.NotNil(t, records["r1"].Error)
	assert.Equal(t, PendingDelivery, records["r2"].State)
	assert.Zero(t, records["r2"].NumberOfAttempts)
	assert.Equal(t, Delivered, records["r3"].State)
	assert.Equal(t, Delivered, records["r4"].State)
}

func TestProcessRecordsHonorsBatchSize(t *testing.T) {
	store := newFakeStore(
		Record{ID: "r1", Message: Message{Key: "a", Topic: "orders"}, State: PendingDelivery},
		Record{ID: "r2", Message: Message{Key: "b", Topic: "orders"}, State: PendingDelivery},
		Record{ID: "r3", Message: Message{Key: "c", Topic: "orders"}, State: PendingDelivery},
	)
	broker := &fakeBroker{}
	p := newRecordProcessor(store, broker, "m1", 2, RetryPolicy{}, nil)

	require.NoError(t, p.ProcessRecords(t.Context()))
	require.Len(t, store.LockCalls(), 1)
	assert.Equal(t, 2, store.LockCalls()[0].limit)
	assert.Len(t, broker.Sent(), 2)
}

func TestProcessRecordsPublishesAllTopicsInSharedTable(t *testing.T) {
	expected := []Message{
		{Topic: "order-placed", Key: "order-1"},
		{Topic: "payment-succeeded", Key: "payment-1"},
		{Topic: "shipment-created", Key: "shipment-1"},
		{Topic: "review-created", Key: "review-1"},
		{Topic: "review-moderated", Key: "review-1"},
	}
	store := newFakeStore(
		Record{ID: "r1", Message: expected[0], State: PendingDelivery, CreatedOn: base},
		Record{ID: "r2", Message: expected[1], State: PendingDelivery, CreatedOn: base.Add(time.Second)},
		Record{ID: "r3", Message: expected[2], State: PendingDelivery, CreatedOn: base.Add(2 * time.Second)},
		Record{ID: "r4", Message: expected[3], State: PendingDelivery, CreatedOn: base.Add(3 * time.Second)},
		Record{ID: "r5", Message: expected[4], State: PendingDelivery, CreatedOn: base.Add(4 * time.Second)},
	)
	broker := &fakeBroker{}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, nil)

	require.NoError(t, p.ProcessRecords(t.Context()))
	assert.Equal(t, expected, broker.Sent())

	records := store.Records()
	for _, id := range []string{"r1", "r2", "r3", "r4", "r5"} {
		assert.Equal(t, Delivered, records[id].State)
	}
}

func TestProcessRecordsSkipsWithoutRelayLock(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery})
	broker := &fakeBroker{}
	lock := &fakeRelayLock{}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, lock)

	require.NoError(t, p.ProcessRecords(t.Context()))
	assert.Empty(t, broker.Sent())
	assert.Empty(t, store.LockCalls())
	assert.Zero(t, store.ClearAllCalls())
	assert.Equal(t, 1, lock.acquireCalls)
	assert.Zero(t, lock.releaseCalls)
}

func TestProcessRecordsReturnsRelayLockError(t *testing.T) {
	acquireErr := errors.New("advisory lock unavailable")
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery})
	broker := &fakeBroker{}
	lock := &fakeRelayLock{acquireErr: acquireErr}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, lock)

	err := p.ProcessRecords(t.Context())
	require.ErrorIs(t, err, acquireErr)
	assert.Empty(t, broker.Sent())
	assert.Empty(t, store.LockCalls())
	assert.Zero(t, lock.releaseCalls)
}

func TestProcessRecordsReleasesRelayLockAfterCancellation(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery})
	broker := &fakeBroker{}
	lock := &fakeRelayLock{acquired: true}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, lock)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := p.ProcessRecords(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, broker.Sent())
	assert.Equal(t, 1, lock.releaseCalls)
	require.NoError(t, lock.releaseCtxErr)
}

func TestProcessRecordsReleasesRelayLockAfterLockFailure(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery})
	store.lockErr = errors.New("claim failed")
	broker := &fakeBroker{}
	lock := &fakeRelayLock{acquired: true}
	p := newRecordProcessor(store, broker, "m1", 10, RetryPolicy{}, lock)

	require.Error(t, p.ProcessRecords(t.Context()))
	assert.Empty(t, broker.Sent())
	assert.Equal(t, 1, lock.releaseCalls)
}

func TestReplayerRequeuesAbandonedRows(t *testing.T) {
	store := newFakeStore(
		Record{ID: "r1", Message: Message{Key: "k1", Topic: "orders"}, State: MaxAttemptsReached, CreatedOn: base},
		Record{ID: "r2", Message: Message{Key: "k2", Topic: "payments"}, State: MaxAttemptsReached, CreatedOn: base},
		Record{ID: "r3", State: PendingDelivery},
	)
	replayer := NewReplayer(store)

	result, err := replayer.Run(t.Context(), ReplayOptions{Limit: 1})
	require.NoError(t, err)
	assert.Equal(t, ReplayResult{Selected: 1, Requeued: 1}, result)
	records := store.Records()
	assert.Equal(t, PendingDelivery, records["r1"].State)
	assert.Zero(t, records["r1"].NumberOfAttempts)
	assert.Nil(t, records["r1"].Error)
	assert.Equal(t, MaxAttemptsReached, records["r2"].State)

	result, err = replayer.Run(t.Context(), ReplayOptions{Topic: "payments", Limit: 0, DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, ReplayResult{Selected: 1, Requeued: 0}, result)
	assert.Equal(t, MaxAttemptsReached, store.Records()["r2"].State)
}

type statsStore struct {
	*fakeStore
	stats Stats
}

func (s *statsStore) GetStats(context.Context) (Stats, error) {
	return s.stats, nil
}

type fakeStatsRecorder struct {
	counts map[string]int64
	ages   map[string]time.Duration
}

func newFakeStatsRecorder() *fakeStatsRecorder {
	return &fakeStatsRecorder{
		counts: make(map[string]int64),
		ages:   make(map[string]time.Duration),
	}
}

func (r *fakeStatsRecorder) SetOutboxRecords(state string, count int64) {
	r.counts[state] = count
}

func (r *fakeStatsRecorder) SetOutboxOldestRecordAge(state string, age time.Duration) {
	r.ages[state] = age
}

func TestMetricsCollectorCollectsByState(t *testing.T) {
	stats := Stats{
		Pending:         7,
		Locked:          2,
		Delivered:       100,
		Abandoned:       1,
		OldestPending:   5 * time.Minute,
		OldestLocked:    time.Minute,
		OldestDelivered: time.Hour,
		OldestAbandoned: 30 * time.Minute,
	}
	store := &statsStore{fakeStore: newFakeStore(), stats: stats}
	recorder := newFakeStatsRecorder()
	collector := NewMetricsCollector(store, recorder, time.Hour)

	require.NoError(t, collector.collect(t.Context()))
	assert.Equal(t, map[string]int64{
		StateLabelPending:   7,
		StateLabelLocked:    2,
		StateLabelDelivered: 100,
		StateLabelAbandoned: 1,
	}, recorder.counts)
	assert.Equal(t, map[string]time.Duration{
		StateLabelPending:   5 * time.Minute,
		StateLabelLocked:    time.Minute,
		StateLabelDelivered: time.Hour,
		StateLabelAbandoned: 30 * time.Minute,
	}, recorder.ages)
}
