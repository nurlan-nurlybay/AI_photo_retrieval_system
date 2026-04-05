package storage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	Client     *s3.Client
	BucketName string
}

func NewS3Client(cfg aws.Config, bucketName string) *S3Client {
	return &S3Client{
		Client:     s3.NewFromConfig(cfg),
		BucketName: bucketName,
	}
}

func (s *S3Client) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.Client)
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))
	
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (s *S3Client) Put(ctx context.Context, key string, r *bytes.Reader) (string, error) {
	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		return "", fmt.Errorf("s3 put error: %w", err)
	}
	// Return a virtual-hosted style URL (assuming public access, or just as a stable identifier)
	publicURL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.BucketName, key)
	return publicURL, nil
}

func (s *S3Client) Delete(ctx context.Context, key string) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 delete error: %w", err)
	}
	return nil
}
