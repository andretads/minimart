package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishReplayPreservesPayloadAndKeyStripsTriageHeaders(t *testing.T) {
	pub := &fakeHeaderPublisher{}
	msg := testMessage(`{"event":"original"}`)
	msg.Metadata.Set(DLQHeaderOriginalTopic, OrderPlacedTopic.String())
	msg.Metadata.Set(DLQHeaderError, "boom")
	msg.Metadata.Set("trace-id", "abc")

	err := publishReplay(t.Context(), OrderPlacedTopic, DLQTopic(OrderPlacedTopic), "aggregate-1", msg, pub)

	require.NoError(t, err)
	require.Len(t, pub.published, 1)
	got := pub.published[0].msg
	assert.Equal(t, OrderPlacedTopic, got.Topic)
	assert.Equal(t, "aggregate-1", got.Key)
	assert.Equal(t, msg.Payload, got.Payload)
	assert.Equal(t, "abc", got.Headers["trace-id"])
	assert.Equal(t, DLQTopic(OrderPlacedTopic).String(), got.Headers[minimartHeaderPrefix+"replayed-from"])
	assert.NotEmpty(t, got.Headers[minimartHeaderPrefix+"replayed-at"])
	_, hasError := got.Headers[DLQHeaderError]
	assert.False(t, hasError)
	_, hasOriginalTopic := got.Headers[DLQHeaderOriginalTopic]
	assert.False(t, hasOriginalTopic)
}

func TestPublishReplayIncrementsReplayCount(t *testing.T) {
	pub := &fakeHeaderPublisher{}
	msg := testMessage(`{"event":"original"}`)
	msg.Metadata.Set(DLQHeaderReplayCount, "2")

	err := publishReplay(t.Context(), OrderPlacedTopic, DLQTopic(OrderPlacedTopic), "aggregate-1", msg, pub)

	require.NoError(t, err)
	require.Len(t, pub.published, 1)
	assert.Equal(t, "3", pub.published[0].msg.Headers[DLQHeaderReplayCount])
}

func TestReplayCountOf(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "missing", value: "", want: 0},
		{name: "invalid", value: "abc", want: 0},
		{name: "valid", value: "4", want: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := map[string]string{}
			if tt.value != "" {
				metadata[DLQHeaderReplayCount] = tt.value
			}
			assert.Equal(t, tt.want, replayCountOf(metadata))
		})
	}
}

func TestReplayRequiresPublisherAndTarget(t *testing.T) {
	_, err := Replay(t.Context(), nil, OrderPlacedTopic, OrderPlacedTopic, nil, ReplayOptions{})
	require.Error(t, err)

	_, err = Replay(t.Context(), nil, OrderPlacedTopic, "", &fakeHeaderPublisher{}, ReplayOptions{})
	require.Error(t, err)
}
