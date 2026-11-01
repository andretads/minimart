package s3

// Config holds the configuration for the S3 client.
type Config struct {
	Endpoint       string
	Bucket         string
	Region         string
	AccessKey      string
	SecretKey      string
	UsePathStyle   bool
	PublicEndpoint string // Browser-facing endpoint for presigned URLs; defaults to Endpoint if empty
}
