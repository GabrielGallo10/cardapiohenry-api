package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Storage struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
	keyPrefix     string
}

func NewR2Storage(cfg R2) (*R2Storage, error) {
	if cfg.AccountID == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.Bucket == "" {
		return nil, errors.New("configuração do R2 incompleta")
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	publicBaseURL := strings.TrimRight(cfg.PublicBaseURL, "/")
	if publicBaseURL == "" {
		publicBaseURL = fmt.Sprintf("https://%s.r2.dev", cfg.Bucket)
	}

	return &R2Storage{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: publicBaseURL,
		keyPrefix:     strings.Trim(cfg.ObjectKeyPrefix, "/"),
	}, nil
}

func (s *R2Storage) UploadImage(ctx context.Context, reader io.Reader, filename, contentType string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}

	key := fmt.Sprintf("%s/%s%s", s.keyPrefix, time.Now().UTC().Format("20060102-150405"), ext)
	key = strings.TrimPrefix(key, "/")

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", "", err
	}

	return s.publicBaseURL + "/" + key, key, nil
}
