package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestFetchConfigFromURL(t *testing.T) {
	// テスト用のサーバーを作成
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		if _, err := w.Write([]byte(`
tools:
  - name: test-tool
    command: ["echo", "test"]
`)); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// サーバーのURLからコンフィグデータを取得
	data, err := fetchConfigFromURL(server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch config from test server: %v", err)
	}

	// データの内容を確認
	if !bytes.Contains(data, []byte("test-tool")) {
		t.Errorf("Fetched data does not contain expected content")
	}
}

func TestLoadConfigFromURL(t *testing.T) {
	// 元のconfigFileを保存
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// テスト用のサーバーを作成
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		if _, err := w.Write([]byte(`
tools:
  - name: test-tool
    command: ["echo", "test"]
`)); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// configFileをテストサーバーのURLに設定
	configFile = server.URL + "/config.yaml"

	// viper の cleanup
	defer viper.Reset()

	// loadConfig を呼び出し
	err := loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config from URL: %v", err)
	}

	// 設定が正しく読み込まれたか確認
	if len(cfg.Tools) != 1 || cfg.Tools[0].Name != "test-tool" {
		t.Errorf("Config was not loaded correctly from URL")
	}
}

func TestLoadConfigFromInvalidURL(t *testing.T) {
	// 元のconfigFileを保存
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// 無効なURLを設定
	configFile = "https://nonexistent.example.com/config.yaml"

	// loadConfig を呼び出し
	err := loadConfig()
	if err == nil {
		t.Fatalf("Expected error when loading from invalid URL, but got nil")
	}
}

func TestLoadConfigFromLocalFile(t *testing.T) {
	// 元のconfigFileを保存
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// テスト用の一時ファイルを作成
	tempFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // テスト終了後に削除

	// テスト用の設定データを書き込む
	_, err = io.WriteString(tempFile, `
tools:
  - name: local-test-tool
    command: ["echo", "local-test"]
`)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// configFileを一時ファイルのパスに設定
	configFile = tempFile.Name()

	// loadConfig を呼び出し
	err = loadConfig()
	if err != nil {
		t.Fatalf("Failed to load config from local file: %v", err)
	}

	// 設定が正しく読み込まれたか確認
	if len(cfg.Tools) != 1 || cfg.Tools[0].Name != "local-test-tool" {
		t.Errorf("Config was not loaded correctly from local file")
	}
}
