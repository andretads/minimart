// Package tests spins up the real dependencies integration tests need.
//
// It exists so integration tests are self-contained: they start their own
// Postgres with testcontainers-go and apply the same db/migrations the
// production binary runs, instead of depending on a pre-provisioned
// docker-compose database. Nothing here imports "testing"; TestMain drives it,
// which keeps the testing flag registration out of a non-test package.
package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// minioImage is a stable MinIO release. MinIO no longer publishes to Docker
// Hub or allows anonymous pulls from Quay, so this points at the community
// GHCR mirror. Update if needed.
const minioImage = "ghcr.io/coollabsio/minio:RELEASE.2025-10-15T17-29-55Z"

// S3 is a running MinIO container with a bucket already created.
type S3 struct {
	Container testcontainers.Container
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
}

// StartS3 starts an ephemeral MinIO container, waits until it accepts
// connections, creates the default bucket, and returns the container plus
// connection details. Call Close to stop it.
func StartS3(ctx context.Context) (*S3, error) {
	req := testcontainers.ContainerRequest{
		Image:        minioImage,
		ExposedPorts: []string{"9000/tcp"},
		Cmd:          []string{"server", "/data"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		WaitingFor: wait.ForHTTP("/minio/health/live").
			WithPort("9000/tcp").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start minio container: %w", err)
	}

	endpoint, err := container.PortEndpoint(ctx, "9000/tcp", "http")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("minio port endpoint: %w", err)
	}

	// Create the default bucket using the AWS SDK.
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", "")),
	)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	bucket := "minimart"
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("create bucket: %w", err)
	}

	return &S3{
		Container: container,
		Endpoint:  endpoint,
		Bucket:    bucket,
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
	}, nil
}

// Close stops the container.
func (s *S3) Close(ctx context.Context) error {
	return s.Container.Terminate(ctx)
}
