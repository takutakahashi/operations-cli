package upgrade

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestFetchVersions(t *testing.T) {
	// Create a test server that returns a mock GitHub API response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request path
		if r.URL.Path != "/repos/testowner/testrepo/releases" {
			t.Errorf("Expected request to /repos/testowner/testrepo/releases, got %s", r.URL.Path)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		mockReleases := []ReleaseInfo{
			{TagName: "v0.3.0"},
			{TagName: "v0.2.0"},
			{TagName: "v0.1.0"},
		}
		json.NewEncoder(w).Encode(mockReleases)
	}))
	defer server.Close()

	// Override the GitHub API URL with our test server URL
	oldHttpGet := httpGet
	defer func() { httpGet = oldHttpGet }()
	
	httpGet = func(url string) (*http.Response, error) {
		return http.Get(server.URL + "/repos/testowner/testrepo/releases")
	}

	// Call the function with our test owner and repo
	versions, err := FetchVersions("testowner", "testrepo")
	if err != nil {
		t.Fatalf("FetchVersions returned error: %v", err)
	}

	// Check the result
	expectedVersions := []string{"v0.3.0", "v0.2.0", "v0.1.0"}
	if len(versions) != len(expectedVersions) {
		t.Fatalf("Expected %d versions, got %d", len(expectedVersions), len(versions))
	}

	for i, v := range versions {
		if v != expectedVersions[i] {
			t.Errorf("Expected version at position %d to be %s, got %s", i, expectedVersions[i], v)
		}
	}
}

func TestFetchVersionInfo(t *testing.T) {
	// Create a test server that returns a mock GitHub API response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request path
		if r.URL.Path == "/repos/testowner/testrepo/releases/latest" {
			// Return a mock latest release response
			w.Header().Set("Content-Type", "application/json")
			mockRelease := ReleaseInfo{
				TagName: "v0.3.0",
				Name:    "Release v0.3.0",
				Assets: []Asset{
					{
						Name:        "operations_v0.3.0_linux_x86_64.tar.gz",
						DownloadURL: "https://example.com/operations_v0.3.0_linux_x86_64.tar.gz",
						Size:        1000,
					},
				},
			}
			json.NewEncoder(w).Encode(mockRelease)
			return
		} else if r.URL.Path == "/repos/testowner/testrepo/releases/tags/v0.2.0" {
			// Return a mock specific release response
			w.Header().Set("Content-Type", "application/json")
			mockRelease := ReleaseInfo{
				TagName: "v0.2.0",
				Name:    "Release v0.2.0",
				Assets: []Asset{
					{
						Name:        "operations_v0.2.0_linux_x86_64.tar.gz",
						DownloadURL: "https://example.com/operations_v0.2.0_linux_x86_64.tar.gz",
						Size:        900,
					},
				},
			}
			json.NewEncoder(w).Encode(mockRelease)
			return
		}

		t.Errorf("Unexpected request to %s", r.URL.Path)
		http.Error(w, "Not found", http.StatusNotFound)
	}))
	defer server.Close()

	// Override the GitHub API URL with our test server URL
	oldHttpGet := httpGet
	defer func() { httpGet = oldHttpGet }()
	
	httpGet = func(url string) (*http.Response, error) {
		if url == "https://api.github.com/repos/testowner/testrepo/releases/latest" {
			return http.Get(server.URL + "/repos/testowner/testrepo/releases/latest")
		} else if url == "https://api.github.com/repos/testowner/testrepo/releases/tags/v0.2.0" {
			return http.Get(server.URL + "/repos/testowner/testrepo/releases/tags/v0.2.0")
		}
		return nil, nil
	}

	// Test fetching latest release
	latestRelease, err := FetchVersionInfo("testowner", "testrepo", "latest")
	if err != nil {
		t.Fatalf("FetchVersionInfo (latest) returned error: %v", err)
	}
	if latestRelease.TagName != "v0.3.0" {
		t.Errorf("Expected latest version to be v0.3.0, got %s", latestRelease.TagName)
	}

	// Test fetching specific release
	specificRelease, err := FetchVersionInfo("testowner", "testrepo", "v0.2.0")
	if err != nil {
		t.Fatalf("FetchVersionInfo (v0.2.0) returned error: %v", err)
	}
	if specificRelease.TagName != "v0.2.0" {
		t.Errorf("Expected version to be v0.2.0, got %s", specificRelease.TagName)
	}
}

func TestFindMatchingAsset(t *testing.T) {
	// Create a test release with multiple assets
	release := &ReleaseInfo{
		TagName: "v0.3.0",
		Assets: []Asset{
			{
				Name:        "operations_v0.3.0_linux_x86_64.tar.gz",
				DownloadURL: "https://example.com/operations_v0.3.0_linux_x86_64.tar.gz",
			},
			{
				Name:        "operations_v0.3.0_darwin_x86_64.tar.gz",
				DownloadURL: "https://example.com/operations_v0.3.0_darwin_x86_64.tar.gz",
			},
			{
				Name:        "operations_v0.3.0_linux_aarch64.tar.gz",
				DownloadURL: "https://example.com/operations_v0.3.0_linux_aarch64.tar.gz",
			},
			{
				Name:        "checksums.txt",
				DownloadURL: "https://example.com/checksums.txt",
			},
		},
	}

	// Save current runtime values to restore them later
	originalGOOS := runtime.GOOS
	originalGOARCH := runtime.GOARCH
	defer func() {
		// This is just a mock for the test, we're not actually changing runtime values
		// But in a real test you might use reflection or other techniques to mock these
	}()

	// Test cases
	testCases := []struct {
		name        string
		mockOS      string
		mockArch    string
		expectAsset string
		expectError bool
	}{
		{
			name:        "Linux AMD64",
			mockOS:      "linux",
			mockArch:    "amd64",
			expectAsset: "operations_v0.3.0_linux_x86_64.tar.gz",
			expectError: false,
		},
		{
			name:        "Darwin AMD64",
			mockOS:      "darwin",
			mockArch:    "amd64",
			expectAsset: "operations_v0.3.0_darwin_x86_64.tar.gz",
			expectError: false,
		},
		{
			name:        "Linux ARM64",
			mockOS:      "linux",
			mockArch:    "arm64",
			expectAsset: "operations_v0.3.0_linux_aarch64.tar.gz",
			expectError: false,
		},
		{
			name:        "Windows AMD64 - Not Found",
			mockOS:      "windows",
			mockArch:    "amd64",
			expectAsset: "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock runtime OS and Arch for this test case
			// In a real test, you would use reflection or other advanced techniques
			// Here we're just pretending for the demonstration
			if originalGOOS != tc.mockOS || originalGOARCH != tc.mockArch {
				t.Skipf("Skipping test case that requires mocking runtime.GOOS=%s and runtime.GOARCH=%s", 
					tc.mockOS, tc.mockArch)
			}

			asset, err := FindMatchingAsset(release)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if asset.Name != tc.expectAsset {
					t.Errorf("Expected asset %s, got %s", tc.expectAsset, asset.Name)
				}
			}
		})
	}
}