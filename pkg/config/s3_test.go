package config

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// mockS3Client implements the s3Client interface for testing
type mockS3Client struct {
	getObjectFunc func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// GetObject calls the mocked getObjectFunc
func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.getObjectFunc(ctx, params, optFns...)
}

func TestIsS3URL(t *testing.T) {
	testCases := []struct {
		url      string
		expected bool
	}{
		{"s3://my-bucket/path/to/file.yaml", true},
		{"s3://my-bucket/file.yaml", true},
		{"s3://my-bucket/", true},
		{"file:///path/to/file.yaml", false},
		{"/path/to/file.yaml", false},
		{"https://example.com/file.yaml", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			result := isS3URL(tc.url)
			if result != tc.expected {
				t.Errorf("isS3URL(%q) = %v; want %v", tc.url, result, tc.expected)
			}
		})
	}
}

func TestParseS3URL(t *testing.T) {
	testCases := []struct {
		url            string
		expectedBucket string
		expectedKey    string
		expectError    bool
	}{
		{"s3://my-bucket/path/to/file.yaml", "my-bucket", "path/to/file.yaml", false},
		{"s3://my-bucket/file.yaml", "my-bucket", "file.yaml", false},
		{"s3://my-bucket/", "my-bucket", "", false},
		{"s3://my-bucket", "my-bucket", "", false}, // This should pass now because we handle URLs without trailing slashes
		{"file:///path/to/file.yaml", "", "", true},
		{"/path/to/file.yaml", "", "", true},
		{"", "", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			bucket, key, err := parseS3URL(tc.url)

			if tc.expectError {
				if err == nil {
					t.Errorf("parseS3URL(%q) = (%q, %q, nil); want error", tc.url, bucket, key)
				}
			} else {
				if err != nil {
					t.Errorf("parseS3URL(%q) = (_, _, %v); want nil error", tc.url, err)
				}
				if bucket != tc.expectedBucket {
					t.Errorf("parseS3URL(%q) got bucket = %q; want %q", tc.url, bucket, tc.expectedBucket)
				}
				if key != tc.expectedKey {
					t.Errorf("parseS3URL(%q) got key = %q; want %q", tc.url, key, tc.expectedKey)
				}
			}
		})
	}
}

func TestReadFromS3(t *testing.T) {
	testCases := []struct {
		name         string
		bucket       string
		key          string
		mockResponse []byte
		mockError    error
		expectError  bool
	}{
		{
			name:         "successful read",
			bucket:       "my-bucket",
			key:          "path/to/file.yaml",
			mockResponse: []byte("test content"),
			mockError:    nil,
			expectError:  false,
		},
		{
			name:         "error reading",
			bucket:       "my-bucket",
			key:          "path/to/nonexistent.yaml",
			mockResponse: nil,
			mockError:    io.EOF,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockS3Client{
				getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					// Verify input parameters
					if *params.Bucket != tc.bucket {
						t.Errorf("expected bucket %q, got %q", tc.bucket, *params.Bucket)
					}
					if *params.Key != tc.key {
						t.Errorf("expected key %q, got %q", tc.key, *params.Key)
					}

					if tc.mockError != nil {
						return nil, tc.mockError
					}

					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader(tc.mockResponse)),
					}, nil
				},
			}

			data, err := readFromS3(mockClient, tc.bucket, tc.key)

			if tc.expectError {
				if err == nil {
					t.Errorf("readFromS3() = (%v, nil); want error", data)
				}
			} else {
				if err != nil {
					t.Errorf("readFromS3() = (_, %v); want nil error", err)
				}
				if !bytes.Equal(data, tc.mockResponse) {
					t.Errorf("readFromS3() = (%v, _); want %v", data, tc.mockResponse)
				}
			}
		})
	}
}

func TestResolveS3ImportPath(t *testing.T) {
	testCases := []struct {
		name        string
		baseURL     string
		importPath  string
		expected    string
		expectError bool
	}{
		{
			name:        "absolute S3 URL in importPath",
			baseURL:     "s3://base-bucket/path/to/config.yaml",
			importPath:  "s3://other-bucket/other-config.yaml",
			expected:    "s3://other-bucket/other-config.yaml",
			expectError: false,
		},
		{
			name:        "relative path in same directory",
			baseURL:     "s3://base-bucket/path/to/config.yaml",
			importPath:  "other-config.yaml",
			expected:    "s3://base-bucket/path/to/other-config.yaml",
			expectError: false,
		},
		{
			name:        "relative path in parent directory",
			baseURL:     "s3://base-bucket/path/to/config.yaml",
			importPath:  "../other-config.yaml",
			expected:    "s3://base-bucket/path/other-config.yaml",
			expectError: false,
		},
		{
			name:        "relative path in subdirectory",
			baseURL:     "s3://base-bucket/path/config.yaml",
			importPath:  "subdir/other-config.yaml",
			expected:    "s3://base-bucket/path/subdir/other-config.yaml",
			expectError: false,
		},
		{
			name:        "relative path with multiple parent directories",
			baseURL:     "s3://base-bucket/a/b/c/config.yaml",
			importPath:  "../../d/other-config.yaml",
			expected:    "s3://base-bucket/a/d/other-config.yaml",
			expectError: false,
		},
		{
			name:        "invalid base URL",
			baseURL:     "file:///path/to/config.yaml",
			importPath:  "other-config.yaml",
			expected:    "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := resolveS3ImportPath(tc.baseURL, tc.importPath)

			if tc.expectError {
				if err == nil {
					t.Errorf("resolveS3ImportPath(%q, %q) = (%q, nil); want error", tc.baseURL, tc.importPath, result)
				}
			} else {
				if err != nil {
					t.Errorf("resolveS3ImportPath(%q, %q) = (_, %v); want nil error", tc.baseURL, tc.importPath, err)
				}
				if result != tc.expected {
					t.Errorf("resolveS3ImportPath(%q, %q) = (%q, _); want %q", tc.baseURL, tc.importPath, result, tc.expected)
				}
			}
		})
	}
}

func TestLoadConfigWithS3URLs(t *testing.T) {
	// Setup mock S3 client for testing
	testConfigContent := `
actions:
  - danger_level: high
    type: confirm
    message: "This is from S3 config."
tools:
  - name: s3tool
    command:
      - s3tool
    params:
      bucket:
        description: The bucket to access
        type: string
        required: true
imports:
  - imported-config.yaml
`

	importedConfigContent := `
actions:
  - danger_level: medium
    type: timeout
    message: "This is from imported S3 config."
    timeout: 5
tools:
  - name: importedtool
    command:
      - importedtool
`

	// Create a custom mock client and inject it for this test
	mockClient := &mockS3Client{
		getObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			bucket := aws.ToString(params.Bucket)
			key := aws.ToString(params.Key)

			// Mock different responses based on the requested S3 object
			switch {
			case bucket == "test-bucket" && key == "config.yaml":
				return &s3.GetObjectOutput{
					Body: io.NopCloser(bytes.NewReader([]byte(testConfigContent))),
				}, nil
			case bucket == "test-bucket" && key == "imported-config.yaml":
				return &s3.GetObjectOutput{
					Body: io.NopCloser(bytes.NewReader([]byte(importedConfigContent))),
				}, nil
			default:
				return nil, io.EOF
			}
		},
	}

	// Replace the defaultS3Client function with a wrapper that returns our mock
	originalDefaultS3Client := defaultS3Client
	defaultS3Client = func() (s3Client, error) {
		return mockClient, nil
	}
	// Restore the original function after the test
	defer func() { defaultS3Client = originalDefaultS3Client }()

	// Test loading a config from S3
	cfg, err := LoadConfig("s3://test-bucket/config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig from S3 failed: %v", err)
	}

	// Verify the config was loaded correctly
	if len(cfg.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(cfg.Actions))
	}

	if len(cfg.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(cfg.Tools))
	}

	// Check for specific tools
	var hasS3Tool, hasImportedTool bool
	for _, tool := range cfg.Tools {
		if tool.Name == "s3tool" {
			hasS3Tool = true
		} else if tool.Name == "importedtool" {
			hasImportedTool = true
		}
	}

	if !hasS3Tool {
		t.Errorf("Expected s3tool from main config not found")
	}
	if !hasImportedTool {
		t.Errorf("Expected importedtool from imported config not found")
	}
}
