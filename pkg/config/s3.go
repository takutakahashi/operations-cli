package config

import (
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// s3URLPattern is a regexp pattern to match S3 URLs in the format s3://bucket/key or s3://bucket
var s3URLPattern = regexp.MustCompile(`^s3://([^/]+)(?:/(.*))?$`)

// isS3URL checks if a path is an S3 URL starting with s3://
func isS3URL(path string) bool {
	return s3URLPattern.MatchString(path)
}

// parseS3URL parses an S3 URL into bucket and key components
func parseS3URL(s3URL string) (bucket, key string, err error) {
	if !isS3URL(s3URL) {
		return "", "", fmt.Errorf("invalid S3 URL format: %s", s3URL)
	}

	// Remove the "s3://" prefix
	path := strings.TrimPrefix(s3URL, "s3://")

	// Split into bucket and key
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid S3 URL: bucket is empty")
	}

	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}

	return bucket, key, nil
}

// s3Client インターフェースは、S3操作に必要なメソッドを定義します
type s3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// defaultS3ClientFunc is a function type that returns an s3Client
type defaultS3ClientFunc func() (s3Client, error)

// defaultS3Client is a variable that holds the function to create a default S3 client
var defaultS3Client defaultS3ClientFunc = func() (s3Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg), nil
}

// loadFromS3 は、S3から設定ファイルを読み込みます
func loadFromS3(s3URL string) ([]byte, error) {
	bucket, key, err := parseS3URL(s3URL)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 URL %s: %w", s3URL, err)
	}

	client, err := defaultS3Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	output, err := client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer output.Body.Close()

	return io.ReadAll(output.Body)
}

// readFromS3 reads a file from S3 using the provided S3 client
func readFromS3(client s3Client, bucket, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Request the S3 object
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object s3://%s/%s: %w", bucket, key, err)
	}
	defer resp.Body.Close()

	// Read the object body
	return io.ReadAll(resp.Body)
}

// resolveS3ImportPath は、S3のインポートパスを解決します
func resolveS3ImportPath(baseURL, importPath string) (string, error) {
	if strings.HasPrefix(importPath, "s3://") {
		return importPath, nil
	}

	baseBucket, baseKey, err := parseS3URL(baseURL)
	if err != nil {
		return "", err
	}

	baseDir := path.Dir(baseKey)
	resolvedKey := path.Join(baseDir, importPath)

	return fmt.Sprintf("s3://%s/%s", baseBucket, resolvedKey), nil
}
