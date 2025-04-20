package config

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
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

// parseS3URL parses an S3 URL and returns the bucket and key
func parseS3URL(url string) (bucket, key string, err error) {
	matches := s3URLPattern.FindStringSubmatch(url)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("invalid S3 URL format: %s", url)
	}
	bucket = matches[1]
	// Key might be missing if URL is just s3://bucket
	if len(matches) > 2 && matches[2] != "" {
		key = matches[2]
	}
	return bucket, key, nil
}

// s3Client wraps AWS S3 client and its methods for easier testing
type s3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// A function type for creating S3 clients
var defaultS3Client = func() (s3Client, error) {
	// Use the default AWS configuration
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create an S3 client
	return s3.NewFromConfig(cfg), nil
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

// resolveS3ImportPath resolves a relative import path against an S3 base URL
func resolveS3ImportPath(baseURL, importPath string) (string, error) {
	// If importPath is already an S3 URL, return it as is
	if isS3URL(importPath) {
		return importPath, nil
	}

	// Parse the base S3 URL
	bucket, key, err := parseS3URL(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base S3 URL: %w", err)
	}

	// Get the directory of the base S3 key
	baseDir := filepath.Dir(key)

	// Resolve the import path relative to the base directory
	resolvedKey := filepath.Join(baseDir, importPath)

	// Clean up the resolved key (remove unnecessary "./" and handle "../" properly)
	resolvedKey = filepath.Clean(resolvedKey)

	// Ensure there's no leading slash in the key
	resolvedKey = strings.TrimPrefix(resolvedKey, "/")

	// Construct the full S3 URL
	return fmt.Sprintf("s3://%s/%s", bucket, resolvedKey), nil
}
