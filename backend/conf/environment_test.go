package conf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const defaultDatabaseURL = "postgres://minimart:minimart@localhost:5432/minimart?sslmode=disable" //nolint:gosec // test fixture DSN, not a real credential

func TestDatabaseURL(t *testing.T) {
	tests := []struct {
		name string
		set  string
		want string
	}{
		{name: "default when unset", set: "", want: defaultDatabaseURL},
		{name: "env overrides default", set: "postgres://user:pw@db:5432/x", want: "postgres://user:pw@db:5432/x"}, //nolint:gosec // test fixture DSN, not a real credential
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tt.set)
			assert.Equal(t, tt.want, DatabaseURL())
		})
	}
}

func TestDatabaseReplicaURL(t *testing.T) {
	tests := []struct {
		name    string
		primary string
		replica string
		want    string
	}{
		{name: "falls back to primary when replica unset", primary: "", replica: "", want: defaultDatabaseURL},
		{name: "falls back to explicit primary", primary: "postgres://primary/db", replica: "", want: "postgres://primary/db"},
		{name: "uses replica when set", primary: "postgres://primary/db", replica: "postgres://replica/db", want: "postgres://replica/db"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tt.primary)
			t.Setenv("DATABASE_REPLICA_URL", tt.replica)
			assert.Equal(t, tt.want, DatabaseReplicaURL())
		})
	}
}

func TestMongoURL(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("MONGO_URL", "")
		assert.Equal(t, "mongodb://localhost:27017", MongoURL())
	})
	t.Run("override", func(t *testing.T) {
		t.Setenv("MONGO_URL", "mongodb://mongo:27017")
		assert.Equal(t, "mongodb://mongo:27017", MongoURL())
	})
}

func TestMongoDatabase(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("MONGO_DATABASE", "")
		assert.Equal(t, "minimart", MongoDatabase())
	})
	t.Run("override", func(t *testing.T) {
		t.Setenv("MONGO_DATABASE", "minimart_test")
		assert.Equal(t, "minimart_test", MongoDatabase())
	})
}

func TestServerHTTPAddr(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("SERVER_HTTP_ADDR", "")
		assert.Equal(t, ":8080", ServerHTTPAddr())
	})
	t.Run("override", func(t *testing.T) {
		t.Setenv("SERVER_HTTP_ADDR", ":9090")
		assert.Equal(t, ":9090", ServerHTTPAddr())
	})
}

func TestWorkerHTTPAddr(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("WORKER_HTTP_ADDR", "")
		assert.Equal(t, ":8081", WorkerHTTPAddr())
	})
	t.Run("override", func(t *testing.T) {
		t.Setenv("WORKER_HTTP_ADDR", ":9091")
		assert.Equal(t, ":9091", WorkerHTTPAddr())
	})
}

func TestEnv(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("ENV", "")
		assert.Equal(t, "development", Env())
	})
	t.Run("override", func(t *testing.T) {
		t.Setenv("ENV", "production")
		assert.Equal(t, "production", Env())
	})
}

func TestKafkaBrokers(t *testing.T) {
	tests := []struct {
		name string
		set  string
		want []string
	}{
		{name: "default single broker", set: "", want: []string{"localhost:9092"}},
		{name: "comma separated", set: "b1:9092,b2:9092,b3:9092", want: []string{"b1:9092", "b2:9092", "b3:9092"}},
		{name: "trims spaces", set: " b1:9092 , b2:9092 ", want: []string{"b1:9092", "b2:9092"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KAFKA_BROKERS", tt.set)
			assert.Equal(t, tt.want, KafkaBrokers())
		})
	}
}

func TestKafkaConsumerSettings(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("KAFKA_CONSUMER_MAX_ATTEMPTS", "")
		t.Setenv("KAFKA_CONSUMER_RETRY_BASE_DELAY", "")
		t.Setenv("KAFKA_CONSUMER_RETRY_MAX_DELAY", "")
		t.Setenv("KAFKA_CONSUMER_RESTART_MIN_BACKOFF", "")
		t.Setenv("KAFKA_CONSUMER_RESTART_MAX_BACKOFF", "")

		assert.Equal(t, 5, KafkaConsumerMaxAttempts())
		assert.Equal(t, 500*time.Millisecond, KafkaConsumerRetryBaseDelay())
		assert.Equal(t, 30*time.Second, KafkaConsumerRetryMaxDelay())
		assert.Equal(t, time.Second, KafkaConsumerRestartMinBackoff())
		assert.Equal(t, 30*time.Second, KafkaConsumerRestartMaxBackoff())
	})
	t.Run("overrides", func(t *testing.T) {
		t.Setenv("KAFKA_CONSUMER_MAX_ATTEMPTS", "3")
		t.Setenv("KAFKA_CONSUMER_RETRY_BASE_DELAY", "100ms")
		t.Setenv("KAFKA_CONSUMER_RETRY_MAX_DELAY", "2s")
		t.Setenv("KAFKA_CONSUMER_RESTART_MIN_BACKOFF", "250ms")
		t.Setenv("KAFKA_CONSUMER_RESTART_MAX_BACKOFF", "5s")

		assert.Equal(t, 3, KafkaConsumerMaxAttempts())
		assert.Equal(t, 100*time.Millisecond, KafkaConsumerRetryBaseDelay())
		assert.Equal(t, 2*time.Second, KafkaConsumerRetryMaxDelay())
		assert.Equal(t, 250*time.Millisecond, KafkaConsumerRestartMinBackoff())
		assert.Equal(t, 5*time.Second, KafkaConsumerRestartMaxBackoff())
	})
}

func TestAdminUsernames(t *testing.T) {
	tests := []struct {
		name string
		set  string
		want []string
	}{
		{name: "empty returns nil", set: "", want: nil},
		{name: "single username", set: "admin", want: []string{"admin"}},
		{name: "comma separated", set: "admin,user1,user2", want: []string{"admin", "user1", "user2"}},
		{name: "trims spaces", set: " admin , user1 ", want: []string{"admin", "user1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ADMIN_USERNAMES", tt.set)
			assert.Equal(t, tt.want, AdminUsernames())
		})
	}
}

func TestRBACPermissionCacheTTL(t *testing.T) {
	tests := []struct {
		name string
		set  string
		want string
	}{
		{name: "default", set: "", want: "5m0s"},
		{name: "override", set: "10m", want: "10m0s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("RBAC_PERMISSIONS_CACHE_TTL", tt.set)
			assert.Equal(t, tt.want, RBACPermissionCacheTTL().String())
		})
	}
}
