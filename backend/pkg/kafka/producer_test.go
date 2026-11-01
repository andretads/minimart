package kafka

import (
	"errors"
	"minimart/pkg/message"
	"minimart/pkg/metrics"
	"testing"

	wmessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWatermillPublisher captures published watermill messages.
type fakeWatermillPublisher struct {
	published []*wmessage.Message
	err       error
}

func (f *fakeWatermillPublisher) Publish(_ string, msgs ...*wmessage.Message) error {
	f.published = append(f.published, msgs...)
	return f.err
}

func (f *fakeWatermillPublisher) Close() error { return nil }

// countingMetrics records publish results while no-op'ing the rest.
type countingMetrics struct {
	metrics.Noop
	published []string
}

func (c *countingMetrics) IncKafkaPublished(topic, status string) {
	c.published = append(c.published, topic+":"+status)
}

func TestPartitionByKey(t *testing.T) {
	msg := wmessage.NewMessage("id", []byte("x"))
	msg.Metadata.Set("key", "order-1")

	key, err := partitionByKey("order-placed", msg)
	require.NoError(t, err)
	assert.Equal(t, "order-1", key)
}

func TestPublishMarshalsStruct(t *testing.T) {
	pub := &fakeWatermillPublisher{}
	p := &Producer{pub: pub, m: metrics.Noop{}}

	err := p.publish(t.Context(), Message{Topic: OrderPlacedTopic, Key: "order-1", Payload: message.OrderPlaced{
		OrderID: "order-1", UserID: "u1", TotalCents: 1250, ItemCount: 2,
	}})
	require.NoError(t, err)

	require.Len(t, pub.published, 1)
	got := pub.published[0]
	assert.JSONEq(t, `{"order_id":"order-1","user_id":"u1","total_cents":1250,"item_count":2}`, string(got.Payload))
	assert.Equal(t, "order-1", got.Metadata.Get("key"))
}

func TestPublishPassesBytesThrough(t *testing.T) {
	pub := &fakeWatermillPublisher{}
	p := &Producer{pub: pub, m: metrics.Noop{}}

	require.NoError(t, p.publish(t.Context(), Message{Topic: OrderPlacedTopic, Key: "k", Payload: []byte(`{"raw":true}`)}))
	require.Len(t, pub.published, 1)
	assert.JSONEq(t, `{"raw":true}`, string(pub.published[0].Payload))
}

func TestPublishMarshalError(t *testing.T) {
	p := &Producer{pub: &fakeWatermillPublisher{}, m: metrics.Noop{}}

	err := p.publish(t.Context(), Message{Topic: OrderPlacedTopic, Key: "k", Payload: make(chan int)})
	assert.ErrorContains(t, err, "marshal event")
}

func TestPublishSetsHeadersAndPreservesKey(t *testing.T) {
	pub := &fakeWatermillPublisher{}
	p := &Producer{pub: pub, m: metrics.Noop{}}

	err := p.Publish(t.Context(), Message{
		Topic:   DLQTopic(OrderPlacedTopic),
		Key:     "order-1",
		Payload: []byte(`{"raw":true}`),
		Headers: map[string]string{
			DLQHeaderOriginalTopic: OrderPlacedTopic.String(),
			"key":                  "header-must-not-win",
		},
	})
	require.NoError(t, err)
	require.Len(t, pub.published, 1)
	got := pub.published[0]
	assert.Equal(t, "order-1", got.Metadata.Get("key"))
	assert.Equal(t, OrderPlacedTopic.String(), got.Metadata.Get(DLQHeaderOriginalTopic))
}

func TestPublishUnderlyingError(t *testing.T) {
	pub := &fakeWatermillPublisher{err: errors.New("broker down")}
	p := &Producer{pub: pub, m: metrics.Noop{}}

	err := p.publish(t.Context(), Message{Topic: OrderPlacedTopic, Key: "k", Payload: message.OrderPlaced{}})
	assert.ErrorContains(t, err, "publish to order-placed")
}

func TestPublishMessageRecordsMetrics(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := &countingMetrics{}
		p := &Producer{pub: &fakeWatermillPublisher{}, m: m}

		require.NoError(t, p.PublishMessage(t.Context(), OrderPlacedTopic, "k", message.OrderPlaced{}))
		assert.Equal(t, []string{"order-placed:success"}, m.published)
	})

	t.Run("error", func(t *testing.T) {
		m := &countingMetrics{}
		p := &Producer{pub: &fakeWatermillPublisher{err: errors.New("boom")}, m: m}

		require.Error(t, p.PublishMessage(t.Context(), OrderPlacedTopic, "k", message.OrderPlaced{}))
		assert.Equal(t, []string{"order-placed:error"}, m.published)
	})
}
