package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Client struct {
	s3     *s3.Client
	bucket string
	pubURL string
}

func NewR2Client(accountID, accessKeyID, secretAccessKey, bucket, publicURL string) (*R2Client, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{URL: endpoint}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("load R2 config: %w", err)
	}

	return &R2Client{
		s3:     s3.NewFromConfig(cfg),
		bucket: bucket,
		pubURL: publicURL,
	}, nil
}

type UploadResult struct {
	Key       string `json:"key"`
	PublicURL string `json:"public_url"`
	Size      int    `json:"size"`
}

// Upload 上传文件到 R2，返回公开访问 URL
func (r *R2Client) Upload(ctx context.Context, fileType, ext string, data []byte) (*UploadResult, error) {
	id, err := randomID()
	if err != nil {
		return nil, fmt.Errorf("generate file id: %w", err)
	}
	key := path.Join(
		fileType,
		time.Now().Format("2006-01-02"),
		fmt.Sprintf("%s.%s", id, ext),
	)

	contentType := contentTypeFor(ext)

	_, err = r.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("upload to R2: %w", err)
	}

	pubURL := ""
	if r.pubURL != "" {
		pubURL = r.pubURL + "/" + key
	}

	return &UploadResult{
		Key:       key,
		PublicURL: pubURL,
		Size:      len(data),
	}, nil
}

// Delete 删除 R2 上的对象，key 为存储路径（如 music/2024-01-01/abc.mp3）
func (r *R2Client) Delete(ctx context.Context, key string) error {
	_, err := r.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	return err
}

// KeyFromURL 从公开 URL 反推 key，URL 不匹配时返回空字符串
func (r *R2Client) KeyFromURL(publicURL string) string {
	if r.pubURL == "" || publicURL == "" {
		return ""
	}
	prefix := r.pubURL + "/"
	if len(publicURL) > len(prefix) && publicURL[:len(prefix)] == prefix {
		return publicURL[len(prefix):]
	}
	return ""
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func contentTypeFor(ext string) string {
	switch ext {
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "ogg":
		return "audio/ogg"
	case "txt":
		return "text/plain; charset=utf-8"
	case "json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
