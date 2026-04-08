package config

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
)

const maxUploadImageBytes = 10 << 20

type R2Storage struct {
	client    *s3.Client
	bucket    string
	keyPrefix string
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

	// R2: match Cloudflare's aws-sdk-go-v2 example (BaseEndpoint only). Newer SDKs
	// default to optional request checksums that R2 may reject; use WhenRequired.
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = cfg.UsePathStyle
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})

	return &R2Storage{
		client:    client,
		bucket:    cfg.Bucket,
		keyPrefix: strings.Trim(cfg.ObjectKeyPrefix, "/"),
	}, nil
}

// UploadImage guarda no R2 com chave derivada do SHA-256 dos bytes.
// Se já existir objeto com o mesmo conteúdo (mesma chave), não faz upload de novo.
func (s *R2Storage) UploadImage(ctx context.Context, reader io.Reader, contentLength int64, filename, contentType string) (objectKey string, err error) {
	_ = contentLength
	_ = filename
	limited := io.LimitReader(reader, maxUploadImageBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New("arquivo vazio")
	}
	if len(data) > maxUploadImageBytes {
		return "", errors.New("arquivo demasiado grande (máx. 10 MiB)")
	}

	sum := sha256.Sum256(data)
	hashHex := hex.EncodeToString(sum[:])
	prefix := strings.Trim(s.keyPrefix, "/")
	var key string
	if prefix == "" {
		key = "by-hash/" + hashHex
	} else {
		key = prefix + "/by-hash/" + hashHex
	}

	exists, err := s.objectExists(ctx, key)
	if err != nil {
		return "", err
	}
	if exists {
		return key, nil
	}

	n := int64(len(data))
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:             aws.String(s.bucket),
		Key:                aws.String(key),
		Body:               bytes.NewReader(data),
		ContentType:        aws.String(contentType),
		ContentLength:      aws.Int64(n),
		CacheControl:       aws.String("public, max-age=31536000, immutable"),
		ContentDisposition: aws.String("inline"),
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *R2Storage) objectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}
	var re *awshttp.ResponseError
	if errors.As(err, &re) && re.Response != nil && re.Response.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

// OwnedObjectKey indica se a chave pertence ao prefixo configurado (evita apagar fora do bucket lógico).
func (s *R2Storage) OwnedObjectKey(key string) bool {
	key = strings.Trim(strings.TrimPrefix(key, "/"), " ")
	if key == "" || strings.Contains(key, "..") {
		return false
	}
	p := strings.Trim(s.keyPrefix, "/")
	if p == "" {
		return true
	}
	return key == p || strings.HasPrefix(key, p+"/")
}

// DeleteObject remove um objeto no R2 (só chaves sob o prefixo configurado).
func (s *R2Storage) DeleteObject(ctx context.Context, key string) error {
	key = strings.Trim(strings.TrimPrefix(key, "/"), " ")
	if !s.OwnedObjectKey(key) {
		return errors.New("chave fora do prefixo permitido")
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// GetObject obtém o objeto no R2 (usado pelo proxy público GET /storage/…).
func (s *R2Storage) GetObject(ctx context.Context, key string) (*s3.GetObjectOutput, error) {
	return s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
}
