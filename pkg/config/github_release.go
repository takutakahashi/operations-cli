package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

// githubClient インターフェースは、GitHub操作に必要なメソッドを定義します
type githubClient interface {
	GetReleaseAsset(ctx context.Context, owner, repo, assetPath, tag string) (io.ReadCloser, error)
}

// defaultGitHubClient は、デフォルトのGitHubクライアントを返します
type defaultGitHubClientImpl struct {
	client *http.Client
}

func (c *defaultGitHubClientImpl) GetReleaseAsset(ctx context.Context, owner, repo, assetPath, tag string) (io.ReadCloser, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, tag, assetPath)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to get release asset: status code %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// defaultGitHubClient は、デフォルトのGitHubクライアントを返します
func defaultGitHubClient() (githubClient, error) {
	return &defaultGitHubClientImpl{
		client: &http.Client{},
	}, nil
}

// loadFromGitHubRelease は、GitHub Releaseから設定ファイルを読み込みます
func loadFromGitHubRelease(githubURL string) ([]byte, error) {
	owner, repo, assetPath, tag, err := parseGitHubReleaseURL(githubURL)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub Release URL %s: %w", githubURL, err)
	}

	return readFromGitHubRelease(nil, owner, repo, assetPath, tag)
}

// isGitHubReleaseURL checks if a path is a GitHub Release URL
func isGitHubReleaseURL(path string) bool {
	if !strings.HasPrefix(path, "github_release://") {
		return false
	}

	parts := strings.SplitN(strings.TrimPrefix(path, "github_release://"), "@", 2)
	components := strings.SplitN(parts[0], "/", 3)

	// Check if we have owner, repo, and path components
	if len(components) < 3 || components[0] == "" || components[1] == "" || components[2] == "" {
		return false
	}

	return true
}

// parseGitHubReleaseURL は、GitHub ReleaseのURLをパースします
func parseGitHubReleaseURL(githubURL string) (owner, repo, assetPath, tag string, err error) {
	prefix := "github_release://"
	if !strings.HasPrefix(githubURL, prefix) {
		return "", "", "", "", fmt.Errorf("invalid GitHub Release URL format: %s", githubURL)
	}

	parts := strings.SplitN(strings.TrimPrefix(githubURL, prefix), "@", 2)
	path := parts[0]
	if len(parts) > 1 {
		tag = parts[1]
	}

	components := strings.SplitN(path, "/", 3)
	if len(components) < 3 || components[0] == "" || components[1] == "" || components[2] == "" {
		return "", "", "", "", fmt.Errorf("invalid GitHub Release URL path: %s", path)
	}

	owner = components[0]
	repo = components[1]
	assetPath = components[2]

	return owner, repo, assetPath, tag, nil
}

// resolveGitHubReleaseImportPath は、GitHub Releaseのインポートパスを解決します
func resolveGitHubReleaseImportPath(baseURL, importPath string) (string, error) {
	if strings.HasPrefix(importPath, "github_release://") {
		return importPath, nil
	}

	owner, repo, basePath, tag, err := parseGitHubReleaseURL(baseURL)
	if err != nil {
		return "", err
	}

	baseDir := path.Dir(basePath)
	resolvedPath := path.Join(baseDir, importPath)

	if tag != "" {
		return fmt.Sprintf("github_release://%s/%s/%s@%s", owner, repo, resolvedPath, tag), nil
	}
	return fmt.Sprintf("github_release://%s/%s/%s", owner, repo, resolvedPath), nil
}

// readFromGitHubRelease reads configuration from a GitHub Release URL
func readFromGitHubRelease(client githubClient, owner, repo, assetPath, tag string) ([]byte, error) {
	if client == nil {
		var err error
		client, err = defaultGitHubClient()
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub client: %w", err)
		}
	}

	// タグが指定されていない場合は "main" をデフォルト値として使用
	if tag == "" {
		tag = "main"
	}

	body, err := client.GetReleaseAsset(context.Background(), owner, repo, assetPath, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get release asset: %w", err)
	}
	defer body.Close()

	return io.ReadAll(body)
}
