package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientPresignUsesPublicEndpoint(t *testing.T) {
	cfg := Config{
		Endpoint:       "http://internal:9000",
		PublicEndpoint: "http://localhost:9000",
		Bucket:         "test-bucket",
		Region:         "us-east-1",
		AccessKey:      "test",
		SecretKey:      "test",
		UsePathStyle:   true,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	url, err := client.PresignGet(t.Context(), "test-key", 1)
	require.NoError(t, err)

	// Presigned URL should use the public endpoint (localhost:9000), not internal
	assert.Contains(t, url, "http://localhost:9000/test-bucket/test-key")
	assert.NotContains(t, url, "internal:9000")
}

func TestClientPresignDefaultsToEndpointWhenPublicEmpty(t *testing.T) {
	cfg := Config{
		Endpoint:     "http://internal:9000",
		Bucket:       "test-bucket",
		Region:       "us-east-1",
		AccessKey:    "test",
		SecretKey:    "test",
		UsePathStyle: true,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err)

	url, err := client.PresignGet(t.Context(), "test-key", 1)
	require.NoError(t, err)

	// Presigned URL should fall back to the internal endpoint when public is not set
	assert.Contains(t, url, "http://internal:9000/test-bucket/test-key")
}
