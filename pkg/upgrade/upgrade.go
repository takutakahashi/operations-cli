package upgrade

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// httpGet is a variable so it can be mocked in tests
var httpGet = http.Get

// ReleaseInfo represents information about a GitHub release
type ReleaseInfo struct {
	TagName string    `json:"tag_name"`
	Name    string    `json:"name"`
	Assets  []Asset   `json:"assets"`
	Created time.Time `json:"created_at"`
}

// Asset represents a binary asset in a GitHub release
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int    `json:"size"`
}

// FetchVersions retrieves all available versions from the GitHub repository
func FetchVersions(owner, repo string) ([]string, error) {
	releasesURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	resp, err := httpGet(releasesURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch releases, status: %d", resp.StatusCode)
	}

	var releases []ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	versions := make([]string, 0, len(releases))
	for _, release := range releases {
		versions = append(versions, release.TagName)
	}

	// Sort versions in descending order (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})

	return versions, nil
}

// FetchLatestRelease retrieves the latest release information
func FetchLatestRelease(owner, repo string) (*ReleaseInfo, error) {
	latestURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	resp, err := httpGet(latestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch latest release, status: %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode latest release: %w", err)
	}

	return &release, nil
}

// FetchVersionInfo retrieves information about a specific version
func FetchVersionInfo(owner, repo, version string) (*ReleaseInfo, error) {
	// If version is empty or "latest", fetch the latest release
	if version == "" || version == "latest" {
		return FetchLatestRelease(owner, repo)
	}

	// Ensure version has 'v' prefix
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	// Fetch specific release by tag
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, version)

	resp, err := httpGet(releaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release %s: %w", version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release %s not found, status: %d", version, resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release %s: %w", version, err)
	}

	return &release, nil
}

// FindMatchingAsset finds the appropriate asset for the current OS and architecture
func FindMatchingAsset(release *ReleaseInfo) (*Asset, error) {
	// Detect current OS and architecture
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go architecture names to release asset names
	archMapping := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
	}

	archName := arch
	if mappedName, ok := archMapping[arch]; ok {
		archName = mappedName
	}

	// Look for a matching asset
	for _, asset := range release.Assets {
		// Check if this asset matches our OS and architecture
		// Example format: operations_v0.1.0_linux_x86_64.tar.gz
		expectedPart := fmt.Sprintf("_%s_%s", os, archName)
		if strings.Contains(asset.Name, expectedPart) {
			return &asset, nil
		}
	}

	return nil, fmt.Errorf("no matching asset found for %s_%s", os, archName)
}

// DownloadAsset downloads an asset to the specified output directory
func DownloadAsset(asset *Asset, outputDir string) (string, error) {
	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create the output file
	outputPath := filepath.Join(outputDir, asset.Name)
	out, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	// Download the asset
	resp, err := httpGet(asset.DownloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download asset, status: %d", resp.StatusCode)
	}

	// Copy the response body to the output file
	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("failed to save asset: %w", err)
	}

	return outputPath, nil
}

// ExtractBinary extracts the operations binary from the downloaded archive
func ExtractBinary(archivePath, outputDir string) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Check archive type and extract accordingly
	if strings.HasSuffix(archivePath, ".tar.gz") {
		return extractTarGz(archivePath, outputDir)
	} else if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, outputDir)
	}
	
	return "", fmt.Errorf("unsupported archive format: %s", archivePath)
}

// extractTarGz extracts a .tar.gz archive
func extractTarGz(archivePath, outputDir string) (string, error) {
	// This is where we'd implement tar.gz extraction
	// For now, we'll use an external command for simplicity
	cmd := exec.Command("tar", "-xzf", archivePath, "-C", outputDir)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract tar.gz: %w", err)
	}
	
	// Look for the operations binary in the extracted contents
	binaryPath := filepath.Join(outputDir, "operations")
	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("binary not found after extraction: %w", err)
	}
	
	// Make it executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}
	
	return binaryPath, nil
}

// extractZip extracts a .zip archive
func extractZip(archivePath, outputDir string) (string, error) {
	// This is where we'd implement zip extraction
	// For now, we'll use an external command for simplicity
	cmd := exec.Command("unzip", archivePath, "-d", outputDir)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract zip: %w", err)
	}
	
	// Look for the operations binary in the extracted contents
	binaryPath := filepath.Join(outputDir, "operations")
	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("binary not found after extraction: %w", err)
	}
	
	// Make it executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}
	
	return binaryPath, nil
}

// InstallBinary installs the binary to the specified path
func InstallBinary(binaryPath, destPath string) error {
	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Copy the binary to the destination
	srcFile, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer srcFile.Close()
	
	// If destination exists, check if we can write to it
	if _, err := os.Stat(destPath); err == nil {
		// Check if file is writable
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("cannot overwrite existing binary, permission denied: %w", err)
		}
	}
	
	destFile, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()
	
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	
	return nil
}

// Upgrade performs the complete upgrade process
func Upgrade(owner, repo, version, outputPath string, dryRun bool, force bool) error {
	// Get current executable path if outputPath is not specified
	if outputPath == "" {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine current executable path: %w", err)
		}
		outputPath = exePath
	}
	
	// If dry-run, just list available versions and exit
	if dryRun {
		versions, err := FetchVersions(owner, repo)
		if err != nil {
			return fmt.Errorf("failed to fetch available versions: %w", err)
		}
		
		fmt.Println("Available versions:")
		for _, v := range versions {
			fmt.Println(v)
		}
		
		fmt.Printf("\nLatest version: %s\n", versions[0])
		fmt.Printf("Current requested version: %s\n", version)
		
		return nil
	}
	
	// Fetch the release information
	release, err := FetchVersionInfo(owner, repo, version)
	if err != nil {
		return fmt.Errorf("failed to fetch version information: %w", err)
	}
	
	// Find the appropriate asset for the current platform
	asset, err := FindMatchingAsset(release)
	if err != nil {
		return fmt.Errorf("failed to find appropriate binary: %w", err)
	}
	
	// Confirm the upgrade
	if !force {
		fmt.Printf("Upgrading to version %s\n", release.TagName)
		fmt.Printf("Binary: %s\n", asset.Name)
		fmt.Printf("Destination: %s\n", outputPath)
		fmt.Printf("Proceed? [y/N] ")
		
		var response string
		fmt.Scanln(&response)
		
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			return fmt.Errorf("upgrade aborted by user")
		}
	}
	
	// Create a temporary directory for the download
	tempDir, err := os.MkdirTemp("", "operations-upgrade-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up on exit
	
	// Download the asset
	fmt.Printf("Downloading %s...\n", asset.Name)
	archivePath, err := DownloadAsset(asset, tempDir)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	
	// Extract the binary
	fmt.Println("Extracting binary...")
	binaryPath, err := ExtractBinary(archivePath, tempDir)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}
	
	// Install the binary
	fmt.Println("Installing binary...")
	if err := InstallBinary(binaryPath, outputPath); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}
	
	fmt.Printf("Successfully upgraded to version %s\n", release.TagName)
	return nil
}