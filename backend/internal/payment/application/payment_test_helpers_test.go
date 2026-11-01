package application

import (
	"context"
	"errors"
	"log/slog"
	"minimart/internal/order/presentation/module/client"
	"minimart/internal/payment/domain"
	"minimart/internal/payment/infrastructure/memory"
	"minimart/pkg/idempotency"
	"minimart/pkg/metrics"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testEnv wires the application service over an in-memory repository and
// scriptable ports. Each scenario gets a fresh env, which stands in for the
// feature's "empty payment storage and an empty idempotency cache" background.
type testEnv struct {
	svc       *Service
	repo      domain.Repository
	reader    *stubOrderReader
	idem      *stubIdem
	idemStore idempotency.Store
	provider  *stubProvider
	m         metrics.Metrics
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	idem := newStubIdem()
	e := &testEnv{
		repo:      memory.New(),
		reader:    &stubOrderReader{},
		idem:      idem,
		idemStore: idem,
		provider:  approvedProvider(),
		m:         metrics.Noop{},
	}
	e.wire()
	return e
}

func (e *testEnv) wire() {
	var reader OrderReader
	if e.reader != nil {
		reader = e.reader
	}
	e.svc = NewService(e.repo, reader, e.idemStore, e.provider, e.m, time.Hour)
}

// useOrder configures the order reader with a payable order.
func (e *testEnv) useOrder(o client.Order) {
	e.reader = &stubOrderReader{order: o}
	e.wire()
}

// noOrderReader simulates a request that is validated before any order lookup.
func (e *testEnv) noOrderReader() {
	e.reader = nil
	e.wire()
}

func (e *testEnv) orderNotFound() {
	e.reader = &stubOrderReader{err: client.ErrOrderNotFound}
	e.wire()
}

func defaultOrder() client.Order {
	return client.Order{ID: "o1", UserID: "u1", Status: "placed", TotalCents: 1500, Currency: "USD"}
}

func cancelledOrder() client.Order {
	o := defaultOrder()
	o.Status = "cancelled"
	return o
}

func validCreatePaymentRequest() CreatePaymentRequest {
	return CreatePaymentRequest{
		OrderID:        "o1",
		UserID:         "u1",
		IdempotencyKey: "k1",
		AmountCents:    1500,
		Currency:       "USD",
		Method:         "card",
	}
}

// storedPaymentWithStatus saves a payment directly in the given status, so a
// scenario can start from an already-founded payment without re-running the use
// case.
func storedPaymentWithStatus(t *testing.T, repo domain.Repository, id, orderID string, status domain.Status) domain.Payment {
	t.Helper()
	p, err := domain.NewPayment(id, orderID, "u1", "idem-"+id, 1500, "USD", "card", "fake")
	require.NoError(t, err)
	switch status {
	case domain.StatusPending:
	case domain.StatusSucceeded:
		p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-" + id})
	case domain.StatusFailed:
		p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeDeclined, Reason: "no funds"})
	case domain.StatusUnknown:
		p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeTimeout, Reason: "slow", ProviderRef: "ref-" + id})
	case domain.StatusRefunded:
		p, err = p.ApplyOutcome(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-" + id})
		require.NoError(t, err)
		p, err = p.Refund(domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-" + id})
	}
	require.NoError(t, err)
	require.NoError(t, repo.Save(t.Context(), p))
	return p
}

func storedPayment(t *testing.T, repo domain.Repository, id string) domain.Payment {
	t.Helper()
	p, err := repo.GetByID(t.Context(), id)
	require.NoError(t, err)
	return p
}

func reconcileOnce(t *testing.T, svc *Service) {
	t.Helper()
	r := &Reconciler{svc: svc, logger: slog.Default(), age: -time.Second, limit: 10}
	r.RunOnce(t.Context())
}

func countEvent[T domain.Event](events []domain.Event) int {
	count := 0
	for _, ev := range events {
		if _, ok := ev.(T); ok {
			count++
		}
	}
	return count
}

// stubOrderReader answers a fixed order, optionally flipping its status after a
// number of calls to simulate an order cancelled while the charge is in flight.
type stubOrderReader struct {
	mu         sync.Mutex
	order      client.Order
	err        error
	calls      int
	flipAfter  int
	flipStatus string
}

func (s *stubOrderReader) GetOrder(_ context.Context, _ string) (client.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return client.Order{}, s.err
	}
	o := s.order
	if s.flipAfter > 0 && s.calls > s.flipAfter {
		o.Status = s.flipStatus
	}
	return o, nil
}

// stubIdem is an in-memory idempotency.Store with injectable failures.
type stubIdem struct {
	mu          sync.Mutex
	records     map[string]idempotency.Record
	getErr      error
	reserveErr  error
	completeErr error
	releaseErr  error
	released    []string
}

func newStubIdem() *stubIdem {
	return &stubIdem{records: map[string]idempotency.Record{}}
}

func (f *stubIdem) Get(_ context.Context, key string) (idempotency.Record, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return idempotency.Record{}, false, f.getErr
	}
	r, ok := f.records[key]
	return r, ok, nil
}

func (f *stubIdem) Reserve(_ context.Context, key, fp string, _ time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reserveErr != nil {
		return false, f.reserveErr
	}
	if _, ok := f.records[key]; ok {
		return false, nil
	}
	f.records[key] = idempotency.Record{Fingerprint: fp}
	return true, nil
}

func (f *stubIdem) Complete(_ context.Context, key, fp, id string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.completeErr != nil {
		return f.completeErr
	}
	f.records[key] = idempotency.Record{Fingerprint: fp, ResourceID: id}
	return nil
}

func (f *stubIdem) Release(_ context.Context, key, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.released = append(f.released, key)
	if f.releaseErr != nil {
		return f.releaseErr
	}
	delete(f.records, key)
	return nil
}

// failingIdem simulates an unavailable idempotency cache.
type failingIdem struct{}

func (failingIdem) Get(context.Context, string) (idempotency.Record, bool, error) {
	return idempotency.Record{}, false, errors.New("idempotency store unavailable")
}

func (failingIdem) Reserve(context.Context, string, string, time.Duration) (bool, error) {
	return false, errors.New("idempotency store unavailable")
}

func (failingIdem) Complete(context.Context, string, string, string, time.Duration) error {
	return errors.New("idempotency store unavailable")
}

func (failingIdem) Release(context.Context, string, string) error {
	return errors.New("idempotency store unavailable")
}

// failingRepository injects a storage failure into the in-memory repository.
type failingRepository struct {
	domain.Repository
	saveErr   error
	updateErr error
	refundErr error
}

func (r *failingRepository) Save(ctx context.Context, p domain.Payment) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	return r.Repository.Save(ctx, p)
}

func (r *failingRepository) Update(ctx context.Context, p domain.Payment) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	return r.Repository.Update(ctx, p)
}

func (r *failingRepository) Refund(ctx context.Context, p domain.Payment) error {
	if r.refundErr != nil {
		return r.refundErr
	}
	return r.Repository.Refund(ctx, p)
}

// losingRefundRepository answers with a stale succeeded snapshot and rejects the
// refund, as when another worker refunded the payment first.
type losingRefundRepository struct {
	domain.Repository
	listed []domain.Payment
}

func (r *losingRefundRepository) ListSucceededByOrderID(context.Context, string) ([]domain.Payment, error) {
	return r.listed, nil
}

func (r *losingRefundRepository) Refund(context.Context, domain.Payment) error {
	return domain.ErrPaymentNotRefundable
}

// stubProvider is a scriptable domain.Provider.
type stubProvider struct {
	mu            sync.Mutex
	name          string
	auth          domain.Result
	authErr       error
	capture       domain.Result
	captureErr    error
	refund        domain.Result
	refundErr     error
	authorized    int
	captured      int
	refunded      int
	authorizeHook func()
}

func (p *stubProvider) Name() string {
	if p.name == "" {
		return "fake"
	}
	return p.name
}

func (p *stubProvider) Authorize(_ context.Context, _ domain.AuthorizationRequest) (domain.Result, error) {
	if p.authorizeHook != nil {
		p.authorizeHook()
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.authorized++
	return p.auth, p.authErr
}

func (p *stubProvider) Capture(_ context.Context, _ domain.CaptureRequest) (domain.Result, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.captured++
	return p.capture, p.captureErr
}

func (p *stubProvider) GetStatus(context.Context, string) (domain.Result, error) {
	return domain.Result{Outcome: domain.OutcomeApproved}, nil
}

func (p *stubProvider) Refund(_ context.Context, providerRef string, _ int64) (domain.Result, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.refunded++
	if p.refundErr != nil {
		return domain.Result{}, p.refundErr
	}
	if p.refund != (domain.Result{}) {
		return p.refund, nil
	}
	return domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: providerRef}, nil
}

func approvedProvider() *stubProvider {
	return &stubProvider{
		auth:    domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"},
		capture: domain.Result{Outcome: domain.OutcomeApproved, ProviderRef: "ref-1"},
	}
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
