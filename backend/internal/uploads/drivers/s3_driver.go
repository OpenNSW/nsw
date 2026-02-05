package drivers

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Driver implements StorageDriver for S3-compatible storage
type S3Driver struct {
	Client        *s3.Client
	PresignClient *s3.PresignClient
	Bucket        string
	PublicURL     string // Optional: Base URL if files are public
}

func NewS3Driver(client *s3.Client, bucket string, publicURL string) *S3Driver {
	return &S3Driver{
		Client:        client,
		PresignClient: s3.NewPresignClient(client),
		Bucket:        bucket,
		PublicURL:     publicURL,
	}
}

func (d *S3Driver) Save(ctx context.Context, key string, content io.Reader, contentType string) error {
	_, err := d.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(d.Bucket),
		Key:         aws.String(key),
		Body:        content,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}
	return nil
}

func (d *S3Driver) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	resp, err := d.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get from S3: %w", err)
	}

	contentType := "application/octet-stream"
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}

	return resp.Body, contentType, nil
}

func (d *S3Driver) Delete(ctx context.Context, key string) error {
	_, err := d.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(d.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}
	return nil
}

func (d *S3Driver) GenerateURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if d.PublicURL != "" {
		return fmt.Sprintf("%s/%s", d.PublicURL, key), nil
	}

	// Fallback to presigned URL
	if expires == 0 {
		expires = time.Hour
	}

	presignedReq, err := d.PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))

	if err != nil {
		return "", fmt.Errorf("failed to presign URL: %w", err)
	}
	return presignedReq.URL, nil
}
