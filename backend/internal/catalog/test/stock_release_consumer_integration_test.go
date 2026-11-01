//go:build integration

package test

import (
	"context"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/internal/catalog/infrastructure/postgres"
	catalogevent "minimart/internal/catalog/presentation/message/event"
	"minimart/internal/catalog/testkit"
	"minimart/internal/category/presentation/module/client"
	"minimart/pkg/kafka"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"minimart/pkg/moderation"
	"minimart/pkg/outbox"
	outboxpg "minimart/pkg/outbox/postgres"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// fakeCategoryProvider is a test double for CategoryProvider.
type fakeCategoryProvider struct{}

func (f *fakeCategoryProvider) GetCategory(ctx context.Context, id string) (client.CategoryDTO, error) {
	return client.CategoryDTO{ID: id, Name: "Test Category"}, nil
}

// TestCatalogConsumer_OrderStockRelease drives the real catalog consumer against
// real Kafka and Postgres: an order-stock-release message restores the stock the
// order was holding.
func TestCatalogConsumer_OrderStockRelease(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	pool, err := pgxpool.New(ctx, testDSN)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// Seed the test category
	require.NoError(t, seedTestCategory(ctx, pool))

	repo := postgres.NewCatalogRepository(pool, pool, outbox.NewPublisher(outboxpg.NewStore(pool)))
	id := uniqueCatalogID("it-consumer-stock")
	seedIntegrationProduct(t, repo, id, 5)
	require.NoError(t, repo.ReserveStock(ctx, "o1", []domain.ReservationItem{{ProductID: id, Quantity: 2}}))
	require.Equal(t, 3, stockOf(t, repo, id))

	// Create a fake category provider for the test
	fakeCatProvider := &fakeCategoryProvider{}
	h := catalogevent.NewHandler(application.NewService(repo, testkit.NewFakeS3Client(), fakeCatProvider, time.Hour, 5*1024*1024, metrics.Noop{}, moderation.NewNoopProvider()))

	// A per-run topic and group keep this test isolated from other consumers.
	topic := kafka.Topic("order-stock-release-it-" + id)
	group := "catalog-stock-it-" + id

	consumer := kafka.NewConsumer(testKafkaBrokers, nil, metrics.Noop{})
	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Subscribe(ctx, topic, group, h.OrderStockRelease)
	}()

	producer, err := kafka.NewProducer(testKafkaBrokers, metrics.Noop{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = producer.Close() })

	require.NoError(t, producer.PublishMessage(ctx, topic, "o1", message.OrderStockRelease{OrderID: "o1"}))

	require.Eventually(t, func() bool {
		p, err := repo.GetByID(t.Context(), id)
		return err == nil && p.Stock == 5
	}, 10*time.Second, 100*time.Millisecond, "stock should be restored after the release event is consumed")

	cancel()
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("consumer did not stop after cancel")
	}
}
