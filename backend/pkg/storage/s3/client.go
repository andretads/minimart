package s3

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	client          *s3.Client
	presignClient   *s3.Client
	bucket          string
	endpoint        string
	presignEndpoint string
}

func NewClient(cfg Config) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	// Internal client for uploads/deletes (uses internal endpoint)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	// Presign client for generating browser-reachable URLs
	// Defaults to Endpoint if PublicEndpoint is not set
	presignEndpoint := cfg.Endpoint
	if cfg.PublicEndpoint != "" {
		presignEndpoint = cfg.PublicEndpoint
	}
	presignClient := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if presignEndpoint != "" {
			o.BaseEndpoint = aws.String(presignEndpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	c := &Client{
		client:          client,
		presignClient:   presignClient,
		bucket:          cfg.Bucket,
		endpoint:        cfg.Endpoint,
		presignEndpoint: presignEndpoint,
	}

	slog.Info("s3 client configured",
		"bucket", c.bucket,
		"endpoint", c.endpoint,
		"presign_endpoint", c.presignEndpoint,
		"region", cfg.Region,
		"path_style", cfg.UsePathStyle,
	)

	return c, nil
}

func (c *Client) Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return fmt.Errorf("s3 upload bucket=%q key=%q endpoint=%q: %w", c.bucket, key, c.endpoint, err)
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete bucket=%q key=%q endpoint=%q: %w", c.bucket, key, c.endpoint, err)
	}
	return nil
}

func (c *Client) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presigner := s3.NewPresignClient(c.presignClient)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("s3 presign bucket=%q key=%q endpoint=%q: %w", c.bucket, key, c.presignEndpoint, err)
	}
	return req.URL, nil
}

func (c *Client) Bucket() string {
	return c.bucket
}
