package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
)

// Config holds S3/MinIO configuration
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PublicURL string // Public URL for accessing uploaded files
}

// MediaUploader provides interface for uploading media to S3 storage
type MediaUploader interface {
	UploadMedia(ctx context.Context, channelID string, messageID int, filename string, contentType string, data []byte) (string, error)
	DeleteMedia(ctx context.Context, objectKey string) error
	GetPublicURL(objectKey string) string
	EnsureBucket(ctx context.Context) error
}

// Client wraps MinIO client with media upload functionality
type Client struct {
	client    *minio.Client
	bucket    string
	publicURL string
	logger    *zerolog.Logger
}

// NewClient creates a new S3/MinIO client
func NewClient(cfg *Config, logger *zerolog.Logger) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &Client{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
		logger:    logger,
	}, nil
}

// EnsureBucket creates bucket if it doesn't exist and sets public read policy
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = c.client.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		c.logger.Info().Str("bucket", c.bucket).Msg("created S3 bucket")

		// Set public read policy for the bucket
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {"AWS": ["*"]},
					"Action": ["s3:GetObject"],
					"Resource": ["arn:aws:s3:::%s/*"]
				}
			]
		}`, c.bucket)

		err = c.client.SetBucketPolicy(ctx, c.bucket, policy)
		if err != nil {
			c.logger.Warn().Err(err).Msg("failed to set public bucket policy, files may not be publicly accessible")
		} else {
			c.logger.Info().Str("bucket", c.bucket).Msg("set public read policy on bucket")
		}
	}

	return nil
}

// UploadMedia uploads media file to S3 and returns public URL
// Path structure: channels/{channel_id}/{YYYY}/{MM}/{DD}/{message_id}_{filename}
func (c *Client) UploadMedia(ctx context.Context, channelID string, messageID int, filename string, contentType string, data []byte) (string, error) {
	now := time.Now().UTC()
	objectKey := fmt.Sprintf(
		"channels/%s/%d/%02d/%02d/%d_%s",
		channelID,
		now.Year(),
		now.Month(),
		now.Day(),
		messageID,
		filename,
	)

	reader := bytes.NewReader(data)
	_, err := c.client.PutObject(ctx, c.bucket, objectKey, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload media to S3: %w", err)
	}

	publicURL := c.GetPublicURL(objectKey)
	c.logger.Debug().
		Str("channel_id", channelID).
		Int("message_id", messageID).
		Str("object_key", objectKey).
		Str("url", publicURL).
		Msg("uploaded media to S3")

	return publicURL, nil
}

// UploadMediaFromReader uploads media from io.Reader to S3 and returns public URL
func (c *Client) UploadMediaFromReader(ctx context.Context, channelID string, messageID int, filename string, contentType string, reader io.Reader, size int64) (string, error) {
	now := time.Now().UTC()
	objectKey := fmt.Sprintf(
		"channels/%s/%d/%02d/%02d/%d_%s",
		channelID,
		now.Year(),
		now.Month(),
		now.Day(),
		messageID,
		filename,
	)

	_, err := c.client.PutObject(ctx, c.bucket, objectKey, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload media to S3: %w", err)
	}

	publicURL := c.GetPublicURL(objectKey)
	c.logger.Debug().
		Str("channel_id", channelID).
		Int("message_id", messageID).
		Str("object_key", objectKey).
		Str("url", publicURL).
		Msg("uploaded media to S3")

	return publicURL, nil
}

// DeleteMedia deletes media file from S3
func (c *Client) DeleteMedia(ctx context.Context, objectKey string) error {
	err := c.client.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete media from S3: %w", err)
	}

	c.logger.Debug().Str("object_key", objectKey).Msg("deleted media from S3")
	return nil
}

// GetPublicURL returns public URL for the given object key
func (c *Client) GetPublicURL(objectKey string) string {
	return fmt.Sprintf("%s/%s/%s", c.publicURL, c.bucket, objectKey)
}

// GetObjectKey extracts object key from full URL
func (c *Client) GetObjectKey(url string) string {
	prefix := fmt.Sprintf("%s/%s/", c.publicURL, c.bucket)
	if len(url) > len(prefix) {
		return url[len(prefix):]
	}
	return path.Base(url)
}
