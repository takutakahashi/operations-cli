package config

import (
	"context"
	"errors"
	"io"
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
	GetLatestReleaseFunc     func(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
	GetReleaseByTagFunc      func(ctx context.Context, owner, repo string, tag string) (*github.RepositoryRelease, *github.Response, error)
	ListReleaseAssetsFunc    func(ctx context.Context, owner, repo string, id int64, opts *github.ListOptions) ([]*github.ReleaseAsset, *github.Response, error)
	DownloadReleaseAssetFunc func(ctx context.Context, owner, repo string, id int64) (io.ReadCloser, string, error)
	GetReleaseAssetFunc      func(ctx context.Context, owner, repo, assetPath, tag string) (io.ReadCloser, error)
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

func (m *MockGitHubClient) GetReleaseAsset(ctx context.Context, owner, repo, assetPath, tag string) (io.ReadCloser, error) {
	if m.GetReleaseAssetFunc != nil {
		return m.GetReleaseAssetFunc(ctx, owner, repo, assetPath, tag)
	}
	return nil, errors.New("GetReleaseAsset not implemented")
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
	mockClient := &MockGitHubClient{
		GetReleaseAssetFunc: func(ctx context.Context, owner, repo, assetPath, tag string) (io.ReadCloser, error) {
			return &StringReadCloser{Reader: strings.NewReader("test content")}, nil
		},
	}

	data, err := readFromGitHubRelease(mockClient, "owner", "repo", "config.yaml", "")
	if err != nil {
		t.Fatalf("readFromGitHubRelease() error = %v", err)
	}

	if string(data) != "test content" {
		t.Errorf("readFromGitHubRelease() = %v, want %v", string(data), "test content")
	}
}

func TestReadFromGitHubRelease_TaggedRelease(t *testing.T) {
	mockClient := &MockGitHubClient{
		GetReleaseAssetFunc: func(ctx context.Context, owner, repo, assetPath, tag string) (io.ReadCloser, error) {
			return &StringReadCloser{Reader: strings.NewReader("test content")}, nil
		},
	}

	data, err := readFromGitHubRelease(mockClient, "owner", "repo", "config.yaml", "v1.0.0")
	if err != nil {
		t.Fatalf("readFromGitHubRelease() error = %v", err)
	}

	if string(data) != "test content" {
		t.Errorf("readFromGitHubRelease() = %v, want %v", string(data), "test content")
	}
}

func TestReadFromGitHubRelease_AssetNotFound(t *testing.T) {
	mockClient := &MockGitHubClient{
		GetReleaseAssetFunc: func(ctx context.Context, owner, repo, assetPath, tag string) (io.ReadCloser, error) {
			return nil, errors.New("asset not found")
		},
	}

	_, err := readFromGitHubRelease(mockClient, "owner", "repo", "nonexistent.yaml", "")
	if err == nil {
		t.Fatal("readFromGitHubRelease() error = nil, want error")
	}
}

func TestReadFromGitHubRelease_ReleaseNotFound(t *testing.T) {
	mockClient := &MockGitHubClient{
		GetReleaseAssetFunc: func(ctx context.Context, owner, repo, assetPath, tag string) (io.ReadCloser, error) {
			return nil, errors.New("release not found")
		},
	}

	_, err := readFromGitHubRelease(mockClient, "owner", "repo", "config.yaml", "v1.0.0")
	if err == nil {
		t.Fatal("readFromGitHubRelease() error = nil, want error")
	}
}
