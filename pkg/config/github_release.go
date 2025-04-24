package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// githubReleaseURLPattern is a regexp pattern to match GitHub Release URLs in the format:
// github_release://owner/repo/path/to/file.yaml or github_release://owner/repo/path/to/file.yaml@tag
var githubReleaseURLPattern = regexp.MustCompile(`^github_release://([^/]+)/([^/]+)/(.+?)(?:@(.+))?$`)

// isGitHubReleaseURL checks if a path is a GitHub Release URL starting with github_release://
func isGitHubReleaseURL(path string) bool {
	return githubReleaseURLPattern.MatchString(path)
}

// parseGitHubReleaseURL parses a GitHub Release URL and returns the owner, repo, path, and optional tag
func parseGitHubReleaseURL(urlStr string) (owner, repo, path, tag string, err error) {
	matches := githubReleaseURLPattern.FindStringSubmatch(urlStr)
	if len(matches) < 4 {
		return "", "", "", "", fmt.Errorf("invalid GitHub Release URL format: %s", urlStr)
	}

	owner = matches[1]
	repo = matches[2]
	path = matches[3]

	// Tag might be missing if URL doesn't have @tag part
	if len(matches) > 4 {
		tag = matches[4]
	}

	return owner, repo, path, tag, nil
}

// githubClient wraps GitHub client for easier testing
type githubClient interface {
	GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
	GetReleaseByTag(ctx context.Context, owner, repo string, tag string) (*github.RepositoryRelease, *github.Response, error)
	DownloadReleaseAsset(ctx context.Context, owner, repo string, id int64) (io.ReadCloser, string, error)
	ListReleaseAssets(ctx context.Context, owner, repo string, id int64, opts *github.ListOptions) ([]*github.ReleaseAsset, *github.Response, error)
}

// defaultGitHubClientWrapper implements the githubClient interface using the actual GitHub client
type defaultGitHubClientWrapper struct {
	client *github.Client
}

func (w *defaultGitHubClientWrapper) GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
	return w.client.Repositories.GetLatestRelease(ctx, owner, repo)
}

func (w *defaultGitHubClientWrapper) GetReleaseByTag(ctx context.Context, owner, repo string, tag string) (*github.RepositoryRelease, *github.Response, error) {
	return w.client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
}

func (w *defaultGitHubClientWrapper) DownloadReleaseAsset(ctx context.Context, owner, repo string, id int64) (io.ReadCloser, string, error) {
	return w.client.Repositories.DownloadReleaseAsset(ctx, owner, repo, id, http.DefaultClient)
}

func (w *defaultGitHubClientWrapper) ListReleaseAssets(ctx context.Context, owner, repo string, id int64, opts *github.ListOptions) ([]*github.ReleaseAsset, *github.Response, error) {
	return w.client.Repositories.ListReleaseAssets(ctx, owner, repo, id, opts)
}

// A function type for creating GitHub clients
var defaultGitHubClient = func() (githubClient, error) {
	// Get GitHub token if available
	token := os.Getenv("GITHUB_TOKEN")
	var httpClient *http.Client

	if token != "" {
		// Create an authenticated client if token is available
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(ctx, ts)
	}

	// Create GitHub client with base URL configuration
	client := github.NewClient(httpClient)

	// Check for custom GitHub Enterprise host
	enterpriseHost := os.Getenv("GITHUB_HOST")
	apiURL := os.Getenv("GITHUB_API_URL")

	// If both are specified, apiURL takes precedence
	if apiURL != "" {
		apiEndpoint, err := url.Parse(apiURL)
		if err != nil {
			return nil, fmt.Errorf("invalid GITHUB_API_URL: %w", err)
		}
		client.BaseURL = apiEndpoint
	} else if enterpriseHost != "" {
		// Construct API URL from enterprise host
		apiEndpoint, err := url.Parse(fmt.Sprintf("https://%s/api/v3/", enterpriseHost))
		if err != nil {
			return nil, fmt.Errorf("invalid GITHUB_HOST: %w", err)
		}
		client.BaseURL = apiEndpoint
	}

	return &defaultGitHubClientWrapper{client: client}, nil
}

// getTagMsg returns a formatted string with tag information if present
func getTagMsg(tag string) string {
	if tag != "" {
		return " with tag " + tag
	}
	return ""
}

// readFromGitHubRelease reads a file from GitHub Release using the provided GitHub client
func readFromGitHubRelease(client githubClient, owner, repo, path, tag string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var release *github.RepositoryRelease
	var err error
	var resp *github.Response

	// Get the release either by tag or latest
	if tag != "" {
		release, resp, err = client.GetReleaseByTag(ctx, owner, repo, tag)
	} else {
		release, resp, err = client.GetLatestRelease(ctx, owner, repo)
	}

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("release not found for %s/%s%s",
				owner, repo, getTagMsg(tag))
		}
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("access forbidden for %s/%s. Try setting GITHUB_TOKEN environment variable",
				owner, repo)
		}
		return nil, fmt.Errorf("failed to get release for %s/%s: %w", owner, repo, err)
	}

	// Check if we have a release
	if release == nil {
		return nil, fmt.Errorf("no release found for %s/%s%s",
			owner, repo, getTagMsg(tag))
	}

	// Find the asset with the matching name
	var asset *github.ReleaseAsset
	opts := &github.ListOptions{PerPage: 100}

	for {
		assets, resp, err := client.ListReleaseAssets(ctx, owner, repo, release.GetID(), opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list assets for %s/%s: %w", owner, repo, err)
		}

		for _, a := range assets {
			if a.GetName() == path || a.GetName() == filepath.Base(path) {
				asset = a
				break
			}
		}

		if asset != nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if asset == nil {
		return nil, fmt.Errorf("asset %s not found in release for %s/%s%s",
			path, owner, repo, getTagMsg(tag))
	}

	// Download the asset
	reader, _, err := client.DownloadReleaseAsset(ctx, owner, repo, asset.GetID())
	if err != nil {
		return nil, fmt.Errorf("failed to download asset %s: %w", path, err)
	}
	defer reader.Close()

	// Read the asset content
	return io.ReadAll(reader)
}

// resolveGitHubReleaseImportPath resolves a relative import path against a GitHub Release base URL
func resolveGitHubReleaseImportPath(baseURL, importPath string) (string, error) {
	// If importPath is already a GitHub Release URL, return it as is
	if isGitHubReleaseURL(importPath) {
		return importPath, nil
	}

	// Parse the base GitHub Release URL
	owner, repo, path, tag, err := parseGitHubReleaseURL(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base GitHub Release URL: %w", err)
	}

	// Get the directory of the base path
	baseDir := filepath.Dir(path)

	// Resolve the import path relative to the base directory
	resolvedPath := filepath.Join(baseDir, importPath)

	// Clean up the resolved path
	resolvedPath = filepath.Clean(resolvedPath)

	// Ensure there's no leading slash in the path
	resolvedPath = strings.TrimPrefix(resolvedPath, "/")

	// Construct the full GitHub Release URL with the same tag if it exists
	if tag != "" {
		return fmt.Sprintf("github_release://%s/%s/%s@%s", owner, repo, resolvedPath, tag), nil
	}

	return fmt.Sprintf("github_release://%s/%s/%s", owner, repo, resolvedPath), nil
}
