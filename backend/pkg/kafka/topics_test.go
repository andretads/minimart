package kafka

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// composePath resolves backend/docker-compose.yml from this file's location, so
// the test works no matter which package's working directory it runs in.
func composePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve the test file's location")
	return filepath.Join(filepath.Dir(file), "..", "..", "docker-compose.yml")
}

// composeDLQTopics returns the source topics the compose stack provisions a
// dead-letter topic for. The redpanda-init service loops over them.
func composeDLQTopics(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(composePath(t))
	require.NoError(t, err)

	const marker = "for topic in "
	for line := range strings.SplitSeq(string(raw), "\n") {
		_, rest, found := strings.Cut(line, marker)
		if !found {
			continue
		}
		end := strings.Index(rest, ";")
		require.Positive(t, end, "the topic list must be terminated by ';'")
		return strings.Fields(rest[:end])
	}
	t.Fatal("docker-compose.yml has no redpanda-init topic list")
	return nil
}

// TestComposeProvisionsDLQForEveryTopic guards the operational contract that
// every source topic has a dead-letter topic with a longer retention. Without
// it, a handler that exhausts its retries on a topic the compose stack did not
// provision loses the message instead of parking it for triage.
func TestComposeProvisionsDLQForEveryTopic(t *testing.T) {
	provisioned := composeDLQTopics(t)

	var expected []string
	for _, topic := range AllTopics() {
		expected = append(expected, topic.String())
	}

	assert.ElementsMatch(t, expected, provisioned,
		"docker-compose.yml must provision a .dlq topic for every source topic")
}

// TestAllTopicsAreUnique guards against a topic constant being listed twice,
// which would silently provision the same DLQ twice.
func TestAllTopicsAreUnique(t *testing.T) {
	topics := AllTopics()
	seen := make(map[Topic]bool, len(topics))
	for _, topic := range topics {
		assert.False(t, seen[topic], "topic %s is listed twice", topic)
		seen[topic] = true
	}
	assert.Len(t, topics, len(seen))
}

// TestDLQTopicSuffix pins the DLQ naming convention: the suffix is part of the
// operational contract that the compose stack and the replay tool both rely on.
func TestDLQTopicSuffix(t *testing.T) {
	for _, topic := range AllTopics() {
		assert.Equal(t, topic.String()+".dlq", DLQTopic(topic).String())
	}
}
