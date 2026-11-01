package application

import (
	"context"
	"errors"
	"minimart/internal/payment/domain"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepo is a hand-written test double for the payment Repository port. Save
// mimics the database's ON CONFLICT DO NOTHING behavior: inserting an existing
// idempotency key reports ErrIdempotencyConflict.
type fakeRepo struct {
	byID      map[string]domain.Payment
	byKey     map[string]string
	saveErr   error
	updateErr error
	refundErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[string]domain.Payment{}, byKey: map[string]string{}}
}

func (f *fakeRepo) Save(_ context.Context, p domain.Payment) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if _, ok := f.byKey[p.IdempotencyKey]; ok {
		return domain.ErrIdempotencyConflict
	}
	f.byID[p.ID] = p
	f.byKey[p.IdempotencyKey] = p.ID
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (domain.Payment, error) {
	p, ok := f.byID[id]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return p, nil
}

func (f *fakeRepo) GetByIdempotencyKey(_ context.Context, key string) (domain.Payment, error) {
	id, ok := f.byKey[key]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return f.byID[id], nil
}

func (f *fakeRepo) ListReconcilable(_ context.Context, _ time.Time, _ int) ([]domain.Payment, error) {
	var out []domain.Payment
	for _, p := range f.byID {
		if p.Status == domain.StatusPending || p.Status == domain.StatusUnknown {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeRepo) Reconcile(_ context.Context, p domain.Payment) error {
	existing, ok := f.byID[p.ID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	if existing.Status != domain.StatusPending && existing.Status != domain.StatusUnknown {
		return domain.ErrPaymentNotPending
	}
	f.byID[p.ID] = p
	return nil
}

func (f *fakeRepo) Refund(_ context.Context, p domain.Payment) error {
	if f.refundErr != nil {
		return f.refundErr
	}
	existing, ok := f.byID[p.ID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	switch existing.Status {
	case domain.StatusPending, domain.StatusSucceeded, domain.StatusUnknown:
	default:
		return domain.ErrPaymentNotRefundable
	}
	f.byID[p.ID] = p
	return nil
}

func (f *fakeRepo) ListSucceededByOrderID(_ context.Context, orderID string) ([]domain.Payment, error) {
	var out []domain.Payment
	for _, p := range f.byID {
		if p.OrderID == orderID && p.Status == domain.StatusSucceeded {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, p domain.Payment) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	existing, ok := f.byID[p.ID]
	if !ok {
		return domain.ErrPaymentNotFound
	}
	if existing.Status != domain.StatusPending {
		return domain.ErrPaymentNotPending
	}
	f.byID[p.ID] = p
	return nil
}

// fakeIdem is an in-memory idempotency.Store with injectable failures.
type fakeIdem struct {
	records     map[string]idempotency.Record
	getErr      error
	reserveErr  error
	completeErr error
	releaseErr  error
	released    []string
}

func newFakeIdem() *fakeIdem {
	return &fakeIdem{records: map[string]idempotency.Record{}}
}

func (f *fakeIdem) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	if f.getErr != nil {
		return idempotency.Record{}, false, f.getErr
	}
	r, ok := f.records[key]
	return r, ok, nil
}

func (f *fakeIdem) Reserve(_ context.Context, key, fp string, _ time.Duration) (bool, error) {
	if f.reserveErr != nil {
		return false, f.reserveErr
	}
	if _, ok := f.records[key]; ok {
		return false, nil
	}
	f.records[key] = idempotency.Record{Fingerprint: fp}
	return true, nil
}

func (f *fakeIdem) Complete(_ context.Context, key, fp, id string, _ time.Duration) error {
	if f.completeErr != nil {
		return f.completeErr
	}
	f.records[key] = idempotency.Record{Fingerprint: fp, ResourceID: id}
	return nil
}

func (f *fakeIdem) Release(_ context.Context, key, _ string) error {
	f.released = append(f.released, key)
	if f.releaseErr != nil {
		return f.releaseErr
	}
	delete(f.records, key)
	return nil
}

// stubProvider is a scriptable domain.Provider.
type stubProvider struct {
	name       string
	auth       domain.Result
	authErr    error
	capture    domain.Result
	captureErr error
	refund     domain.Result
	refundErr  error
	authorized int
	captured   int
	refunded   int
}

func (p *stubProvider) Name() string {
	if p.name == "" {
		return "fake"
	}
	return p.name
}

func (p *stubProvider) Authorize(context.Context, domain.AuthorizationRequest) (domain.Result, error) {
	p.authorized++
	return p.auth, p.authErr
}

func (p *stubProvider) Capture(context.Context, string, int64) (domain.Result, error) {
	p.captured++
	return p.capture, p.captureErr
}

func (p *stubProvider) GetStatus(context.Context, string) (domain.Result, error) {
	return domain.Result{Outcome: domain.OutcomeApproved}, nil
}

func (p *stubProvider) Refund(_ context.Context, providerRef string, _ int64) (domain.Result, error) {
	p.refunded++
	if p.refundErr != nil {
		return domain.Result{}, p.refundErr
	}
	if p.refund != (domain.Result{}) {
		return p.refund, nil
	}
	return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: providerRef}, nil
}

// countingMetrics records payment and idempotency signals while no-op'ing the rest.
type countingMetrics struct {
	metrics.Noop
	payments    []string
	amounts     []float64
	idempotency [][2]string
}

func (c *countingMetrics) IncPayment(state string) { c.payments = append(c.payments, state) }

func (c *countingMetrics) ObservePaymentAmount(cents float64) { c.amounts = append(c.amounts, cents) }

func (c *countingMetrics) IncIdempotency(scope, result string) {
	c.idempotency = append(c.idempotency, [2]string{scope, result})
}

func approvedProvider() *stubProvider {
	return &stubProvider{
		auth:    domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"},
		capture: domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"},
	}
}

func testRequest() CreatePaymentRequest {
	return CreatePaymentRequest{OrderID: "ord-1", UserID: "user-1", IdempotencyKey: "idem-1", AmountCents: 1500, Currency: "USD", Method: "card"}
}

func newService(repo *fakeRepo, idem *fakeIdem, provider *stubProvider, m metrics.Metrics) *Service {
	return NewService(repo, nil, idem, provider, m, 24*time.Hour)
}

func TestCreatePaymentSucceeds(t *testing.T) {
	repo, idem, provider, m := newFakeRepo(), newFakeIdem(), approvedProvider(), &countingMetrics{}
	svc := newService(repo, idem, provider, m)

	got, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)

	assert.Equal(t, domain.StatusSucceeded, got.Status)
	assert.Equal(t, int64(1500), got.AmountCents)
	assert.Equal(t, "fake", got.Provider)
	require.NotNil(t, got.ProviderRef)
	assert.Equal(t, "ref-1", *got.ProviderRef)
	assert.Equal(t, 1, provider.authorized)
	assert.Equal(t, 1, provider.captured)

	// Two events accumulated: initiated + succeeded.
	require.Len(t, got.Events(), 2)
	_, ok := got.Events()[1].(domain.PaymentSucceededEvent)
	assert.True(t, ok)

	// The idempotency key is completed with the payment id.
	rec, found, err := idem.Get(t.Context(), "idem-1")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, got.ID, rec.ResourceID)

	assert.Equal(t, []string{"succeeded"}, m.payments)
	assert.Equal(t, []float64{1500}, m.amounts)
}

func TestCreatePaymentOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		auth       domain.Result
		wantStatus domain.Status
	}{
		{"declined", domain.Result{Outcome: domain.OutcomeDeclined, Reason: "no funds"}, domain.StatusFailed},
		{"timeout is unknown", domain.Result{Outcome: domain.OutcomeTimeout, Reason: "slow"}, domain.StatusUnknown},
		{"technical error is unknown", domain.Result{Outcome: domain.OutcomeTechnicalError, Reason: "boom"}, domain.StatusUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider := &stubProvider{auth: tt.auth}
			svc := newService(newFakeRepo(), newFakeIdem(), provider, metrics.Noop{})

			got, err := svc.CreatePayment(t.Context(), testRequest())
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, 0, provider.captured, "capture only runs after an approval")
			require.NotNil(t, got.FailureReason)
		})
	}
}

func TestCreatePaymentProviderErrorMapsToUnknown(t *testing.T) {
	provider := &stubProvider{authErr: errors.New("connection reset")}
	svc := newService(newFakeRepo(), newFakeIdem(), provider, metrics.Noop{})

	got, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusUnknown, got.Status)
}

func TestCreatePaymentValidation(t *testing.T) {
	svc := newService(newFakeRepo(), newFakeIdem(), approvedProvider(), metrics.Noop{})

	req := testRequest()
	req.AmountCents = 0
	_, err := svc.CreatePayment(t.Context(), req)
	assert.ErrorIs(t, err, domain.ErrPaymentAmountInvalid)
}

func TestCreatePaymentRequiresIdempotencyKey(t *testing.T) {
	svc := newService(newFakeRepo(), newFakeIdem(), approvedProvider(), metrics.Noop{})

	req := testRequest()
	req.IdempotencyKey = ""
	_, err := svc.CreatePayment(t.Context(), req)
	assert.ErrorIs(t, err, ErrIdempotencyKeyRequired)
}

func TestCreatePaymentReplaysSameKey(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	svc := newService(repo, idem, provider, metrics.Noop{})

	first, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)

	second, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID, "a replay returns the same payment")
	assert.Equal(t, 1, provider.authorized, "the provider is not called again")
}

func TestCreatePaymentConflictsOnDifferentParams(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	svc := newService(repo, idem, provider, metrics.Noop{})

	_, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)

	req := testRequest()
	req.AmountCents = 9999 // same key, different amount
	_, err = svc.CreatePayment(t.Context(), req)
	assert.ErrorIs(t, err, domain.ErrIdempotencyConflict)
}

func TestCreatePaymentInProgress(t *testing.T) {
	idem := newFakeIdem()
	// Simulate another in-flight request holding the reservation (empty resource).
	fp := requestFingerprint("ord-1", "user-1", "USD", "card", 1500)
	idem.records["idem-1"] = idempotency.Record{Fingerprint: fp}

	svc := newService(newFakeRepo(), idem, approvedProvider(), metrics.Noop{})
	_, err := svc.CreatePayment(t.Context(), testRequest())
	assert.ErrorIs(t, err, ErrPaymentInProgress)
}

func TestCreatePaymentFailsOpenWhenRedisDown(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()
	idem.getErr = errors.New("redis down")
	idem.reserveErr = errors.New("redis down")
	idem.completeErr = errors.New("redis down")
	svc := newService(repo, idem, provider, metrics.Noop{})

	// Proceeds without the cache; the database records the payment.
	first, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusSucceeded, first.Status)

	// A retry still replays from the database despite Redis being down.
	second, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, 1, provider.authorized)
}

func TestCreatePaymentReplaysWhenDatabaseWonTheRace(t *testing.T) {
	repo, idem, provider := newFakeRepo(), newFakeIdem(), approvedProvider()

	// A concurrent request already created the payment, but the cache is empty.
	winner, err := domain.NewPayment("winner", "ord-1", "user-1", "idem-1", 1500, "USD", "card", "fake")
	require.NoError(t, err)
	winner, err = winner.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-w"})
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), winner))

	svc := newService(repo, idem, provider, metrics.Noop{})
	got, err := svc.CreatePayment(t.Context(), testRequest())
	require.NoError(t, err)
	assert.Equal(t, "winner", got.ID)
	assert.Equal(t, 0, provider.authorized, "the losing request does not charge")
}

func TestCreatePaymentReleasesReservationOnRepositoryFailure(t *testing.T) {
	repo, idem := newFakeRepo(), newFakeIdem()
	repo.saveErr = errors.New("disk on fire")
	svc := newService(repo, idem, approvedProvider(), metrics.Noop{})

	_, err := svc.CreatePayment(t.Context(), testRequest())
	require.Error(t, err)
	assert.Equal(t, []string{"idem-1"}, idem.released, "a failed request frees its reservation")
}
