package config

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v57/github"
)

func TestIsGitHubReleaseURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "Valid GitHub Release URL without tag",
			url:  "github_release://owner/repo/file.yaml",
			want: true,
		},
		{
			name: "Valid GitHub Release URL with tag",
			url:  "github_release://owner/repo/file.yaml@v1.0.0",
			want: true,
		},
		{
			name: "Valid GitHub Release URL with nested path",
			url:  "github_release://owner/repo/path/to/file.yaml",
			want: true,
		},
		{
			name: "Invalid GitHub Release URL - missing owner",
			url:  "github_release:///repo/file.yaml",
			want: false,
		},
		{
			name: "Invalid GitHub Release URL - missing repo",
			url:  "github_release://owner//file.yaml",
			want: false,
		},
		{
			name: "Invalid GitHub Release URL - missing path",
			url:  "github_release://owner/repo/",
			want: false,
		},
		{
			name: "Invalid GitHub Release URL - wrong scheme",
			url:  "github://owner/repo/file.yaml",
			want: false,
		},
		{
			name: "S3 URL",
			url:  "s3://bucket/key",
			want: false,
		},
		{
			name: "Local file path",
			url:  "/path/to/file.yaml",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGitHubReleaseURL(tt.url); got != tt.want {
				t.Errorf("isGitHubReleaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseGitHubReleaseURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantPath  string
		wantTag   string
		wantErr   bool
	}{
		{
			name:      "Valid GitHub Release URL without tag",
			url:       "github_release://owner/repo/file.yaml",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantPath:  "file.yaml",
			wantTag:   "",
			wantErr:   false,
		},
		{
			name:      "Valid GitHub Release URL with tag",
			url:       "github_release://owner/repo/file.yaml@v1.0.0",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantPath:  "file.yaml",
			wantTag:   "v1.0.0",
			wantErr:   false,
		},
		{
			name:      "Valid GitHub Release URL with nested path",
			url:       "github_release://owner/repo/path/to/file.yaml",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantPath:  "path/to/file.yaml",
			wantTag:   "",
			wantErr:   false,
		},
		{
			name:      "Valid GitHub Release URL with nested path and tag",
			url:       "github_release://owner/repo/path/to/file.yaml@v1.0.0",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantPath:  "path/to/file.yaml",
			wantTag:   "v1.0.0",
			wantErr:   false,
		},
		{
			name:    "Invalid GitHub Release URL",
			url:     "invalid://owner/repo/file.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo, gotPath, gotTag, err := parseGitHubReleaseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGitHubReleaseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOwner != tt.wantOwner {
				t.Errorf("parseGitHubReleaseURL() gotOwner = %v, want %v", gotOwner, tt.wantOwner)
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("parseGitHubReleaseURL() gotRepo = %v, want %v", gotRepo, tt.wantRepo)
			}
			if gotPath != tt.wantPath {
				t.Errorf("parseGitHubReleaseURL() gotPath = %v, want %v", gotPath, tt.wantPath)
			}
			if gotTag != tt.wantTag {
				t.Errorf("parseGitHubReleaseURL() gotTag = %v, want %v", gotTag, tt.wantTag)
			}
		})
	}
}

func TestResolveGitHubReleaseImportPath(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		importURL string
		want      string
		wantErr   bool
	}{
		{
			name:      "Absolute import URL",
			baseURL:   "github_release://owner/repo/config.yaml",
			importURL: "github_release://owner2/repo2/imported.yaml",
			want:      "github_release://owner2/repo2/imported.yaml",
			wantErr:   false,
		},
		{
			name:      "Relative import URL - same directory",
			baseURL:   "github_release://owner/repo/config.yaml",
			importURL: "imported.yaml",
			want:      "github_release://owner/repo/imported.yaml",
			wantErr:   false,
		},
		{
			name:      "Relative import URL - subdirectory",
			baseURL:   "github_release://owner/repo/config.yaml",
			importURL: "configs/imported.yaml",
			want:      "github_release://owner/repo/configs/imported.yaml",
			wantErr:   false,
		},
		{
			name:      "Relative import URL - parent directory",
			baseURL:   "github_release://owner/repo/configs/config.yaml",
			importURL: "../imported.yaml",
			want:      "github_release://owner/repo/imported.yaml",
			wantErr:   false,
		},
		{
			name:      "Relative import URL with tag",
			baseURL:   "github_release://owner/repo/config.yaml@v1.0.0",
			importURL: "imported.yaml",
			want:      "github_release://owner/repo/imported.yaml@v1.0.0",
			wantErr:   false,
		},
		{
			name:      "Invalid base URL",
			baseURL:   "invalid://owner/repo/config.yaml",
			importURL: "imported.yaml",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveGitHubReleaseImportPath(tt.baseURL, tt.importURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveGitHubReleaseImportPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolveGitHubReleaseImportPath() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// MockGitHubClient is a mock implementation of the githubClient interface for testing
type MockGitHubClient struct {
	GetLatestReleaseFunc    func(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
	GetReleaseByTagFunc     func(ctx context.Context, owner, repo string, tag string) (*github.RepositoryRelease, *github.Response, error)
	ListReleaseAssetsFunc   func(ctx context.Context, owner, repo string, id int64, opts *github.ListOptions) ([]*github.ReleaseAsset, *github.Response, error)
	DownloadReleaseAssetFunc func(ctx context.Context, owner, repo string, id int64) (io.ReadCloser, string, error)
}

func (m *MockGitHubClient) GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
	return m.GetLatestReleaseFunc(ctx, owner, repo)
}

func (m *MockGitHubClient) GetReleaseByTag(ctx context.Context, owner, repo string, tag string) (*github.RepositoryRelease, *github.Response, error) {
	return m.GetReleaseByTagFunc(ctx, owner, repo, tag)
}

func (m *MockGitHubClient) ListReleaseAssets(ctx context.Context, owner, repo string, id int64, opts *github.ListOptions) ([]*github.ReleaseAsset, *github.Response, error) {
	return m.ListReleaseAssetsFunc(ctx, owner, repo, id, opts)
}

func (m *MockGitHubClient) DownloadReleaseAsset(ctx context.Context, owner, repo string, id int64) (io.ReadCloser, string, error) {
	return m.DownloadReleaseAssetFunc(ctx, owner, repo, id)
}

// StringReadCloser is a simple io.ReadCloser implementation for testing
type StringReadCloser struct {
	Reader io.Reader
}

func (s *StringReadCloser) Read(p []byte) (n int, err error) {
	return s.Reader.Read(p)
}

func (s *StringReadCloser) Close() error {
	return nil
}

func TestReadFromGitHubRelease_LatestRelease(t *testing.T) {
	// Mock response
	testAssetContent := "test: asset content"
	mockRelease := &github.RepositoryRelease{
		ID: github.Int64(123),
	}
	mockAsset := &github.ReleaseAsset{
		ID:   github.Int64(456),
		Name: github.String("config.yaml"),
	}
	mockResp := &github.Response{
		Response: &http.Response{},
	}

	// Create mock client
	mockClient := &MockGitHubClient{
		GetLatestReleaseFunc: func(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
			if owner != "testowner" || repo != "testrepo" {
				return nil, mockResp, errors.New("invalid owner or repo")
			}
			return mockRelease, mockResp, nil
		},
		ListReleaseAssetsFunc: func(ctx context.Context, owner, repo string, id int64, opts *github.ListOptions) ([]*github.ReleaseAsset, *github.Response, error) {
			if id != 123 {
				return nil, mockResp, errors.New("invalid release ID")
			}
			return []*github.ReleaseAsset{mockAsset}, mockResp, nil
		},
		DownloadReleaseAssetFunc: func(ctx context.Context, owner, repo string, id int64) (io.ReadCloser, string, error) {
			if id != 456 {
				return nil, "", errors.New("invalid asset ID")
			}
			return &StringReadCloser{
				Reader: strings.NewReader(testAssetContent),
			}, "application/yaml", nil
		},
	}

	// Test reading from latest release
	data, err := readFromGitHubRelease(mockClient, "testowner", "testrepo", "config.yaml", "")
	if err != nil {
		t.Fatalf("readFromGitHubRelease() error = %v", err)
	}

	// Verify content
	if string(data) != testAssetContent {
		t.Errorf("readFromGitHubRelease() content = %v, want %v", string(data), testAssetContent)
	}
}

func TestReadFromGitHubRelease_TaggedRelease(t *testing.T) {
	// Mock response
	testAssetContent := "test: asset content"
	mockRelease := &github.RepositoryRelease{
		ID: github.Int64(123),
	}
	mockAsset := &github.ReleaseAsset{
		ID:   github.Int64(456),
		Name: github.String("config.yaml"),
	}
	mockResp := &github.Response{
		Response: &http.Response{},
	}

	// Create mock client
	mockClient := &MockGitHubClient{
		GetReleaseByTagFunc: func(ctx context.Context, owner, repo string, tag string) (*github.RepositoryRelease, *github.Response, error) {
			if owner != "testowner" || repo != "testrepo" || tag != "v1.0.0" {
				return nil, mockResp, errors.New("invalid owner, repo, or tag")
			}
			return mockRelease, mockResp, nil
		},
		ListReleaseAssetsFunc: func(ctx context.Context, owner, repo string, id int64, opts *github.ListOptions) ([]*github.ReleaseAsset, *github.Response, error) {
			if id != 123 {
				return nil, mockResp, errors.New("invalid release ID")
			}
			return []*github.ReleaseAsset{mockAsset}, mockResp, nil
		},
		DownloadReleaseAssetFunc: func(ctx context.Context, owner, repo string, id int64) (io.ReadCloser, string, error) {
			if id != 456 {
				return nil, "", errors.New("invalid asset ID")
			}
			return &StringReadCloser{
				Reader: strings.NewReader(testAssetContent),
			}, "application/yaml", nil
		},
	}

	// Test reading from tagged release
	data, err := readFromGitHubRelease(mockClient, "testowner", "testrepo", "config.yaml", "v1.0.0")
	if err != nil {
		t.Fatalf("readFromGitHubRelease() error = %v", err)
	}

	// Verify content
	if string(data) != testAssetContent {
		t.Errorf("readFromGitHubRelease() content = %v, want %v", string(data), testAssetContent)
	}
}

func TestReadFromGitHubRelease_AssetNotFound(t *testing.T) {
	mockRelease := &github.RepositoryRelease{
		ID: github.Int64(123),
	}
	mockResp := &github.Response{
		Response: &http.Response{},
	}

	// Create mock client
	mockClient := &MockGitHubClient{
		GetLatestReleaseFunc: func(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
			return mockRelease, mockResp, nil
		},
		ListReleaseAssetsFunc: func(ctx context.Context, owner, repo string, id int64, opts *github.ListOptions) ([]*github.ReleaseAsset, *github.Response, error) {
			// Return empty assets list
			return []*github.ReleaseAsset{}, mockResp, nil
		},
	}

	// Test asset not found
	_, err := readFromGitHubRelease(mockClient, "testowner", "testrepo", "nonexistent.yaml", "")
	if err == nil {
		t.Fatalf("readFromGitHubRelease() expected error but got nil")
	}

	// Verify error message
	expected := "asset nonexistent.yaml not found"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("readFromGitHubRelease() error = %v, expected to contain %v", err, expected)
	}
}

func TestReadFromGitHubRelease_ReleaseNotFound(t *testing.T) {
	mockResp := &github.Response{
		Response: &http.Response{},
	}
	mockResp.StatusCode = 404

	// Create mock client
	mockClient := &MockGitHubClient{
		GetLatestReleaseFunc: func(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
			return nil, mockResp, errors.New("not found")
		},
	}

	// Test release not found
	_, err := readFromGitHubRelease(mockClient, "testowner", "testrepo", "config.yaml", "")
	if err == nil {
		t.Fatalf("readFromGitHubRelease() expected error but got nil")
	}

	// Verify error message
	expected := "release not found"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("readFromGitHubRelease() error = %v, expected to contain %v", err, expected)
	}
}