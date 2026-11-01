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

	lockErr error

	lockCalls     []lockCall
	clearedByID   []string
	staleBefore   []time.Time
	deletedBefore []time.Time
}

type lockCall struct {
	lockID   string
	lockedOn time.Time
	state    RecordState
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

func (s *fakeStore) UpdateRecordLockByState(_ context.Context, lockID string, lockedOn time.Time, state RecordState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lockCalls = append(s.lockCalls, lockCall{lockID: lockID, lockedOn: lockedOn, state: state})
	if s.lockErr != nil {
		return s.lockErr
	}
	for _, id := range s.order {
		if r := s.records[id]; r.State == state && r.LockID == nil {
			r.LockID = &lockID
			r.LockedOn = &lockedOn
			s.records[id] = r
		}
	}
	return nil
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

func (s *fakeStore) ClearLocksByLockID(_ context.Context, lockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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

func (s *fakeStore) RemoveRecordsBeforeDatetime(_ context.Context, expiryTime time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletedBefore = append(s.deletedBefore, expiryTime)
	return nil
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
	mu   sync.Mutex
	sent []Message
	err  error
}

func (b *fakeBroker) Send(_ context.Context, msg Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, msg)
	return b.err
}

func (b *fakeBroker) Sent() []Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Message, len(b.sent))
	copy(out, b.sent)
	return out
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
	p := newRecordProcessor(store, broker, "m1", RetryPolicy{})
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
	p := newRecordProcessor(store, broker, "m1", RetryPolicy{})

	require.NoError(t, p.ProcessRecords(t.Context()))
	assert.Empty(t, broker.Sent())
}

func TestProcessRecordsSendFailureStopsAndRecordsError(t *testing.T) {
	store := newFakeStore(
		Record{ID: "r1", State: PendingDelivery},
		Record{ID: "r2", State: PendingDelivery},
	)
	broker := &fakeBroker{err: errors.New("broker down")}
	p := newRecordProcessor(store, broker, "m1", RetryPolicy{})
	p.now = func() time.Time { return base }

	err := p.ProcessRecords(t.Context())
	require.Error(t, err)
	assert.Len(t, broker.Sent(), 1, "processing stops at the first failure to preserve order")

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
	p := newRecordProcessor(store, broker, "m1", RetryPolicy{MaxSendAttemptsEnabled: true, MaxSendAttempts: 3})
	p.now = func() time.Time { return base }

	require.Error(t, p.ProcessRecords(t.Context()))
	assert.Equal(t, MaxAttemptsReached, store.Records()["r1"].State)
	assert.Nil(t, store.Records()["r1"].LockID)
}

func TestProcessRecordsLockFailure(t *testing.T) {
	store := newFakeStore(Record{ID: "r1", State: PendingDelivery})
	store.lockErr = errors.New("db down")
	p := newRecordProcessor(store, &fakeBroker{}, "m1", RetryPolicy{})

	require.Error(t, p.ProcessRecords(t.Context()))
	assert.Nil(t, store.Records()["r1"].LockID)
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
	}, "m1")

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

	first := NewDispatcher(store, broker, settings, "worker-1")
	second := NewDispatcher(store, broker, settings, "worker-1")

	firstProcessor, ok := first.recordProcessor.(*recordProcessor)
	require.True(t, ok)
	secondProcessor, ok := second.recordProcessor.(*recordProcessor)
	require.True(t, ok)

	assert.NotEqual(t, firstProcessor.machineID, secondProcessor.machineID)
	assert.Contains(t, firstProcessor.machineID, "worker-1/")
	assert.Contains(t, secondProcessor.machineID, "worker-1/")
}
