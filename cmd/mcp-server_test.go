package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/config"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

func TestMCPServerCommand(t *testing.T) {
	// テスト用のログディレクトリを作成
	tmpDir := filepath.Join(".", "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("Failed to create tmp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir) // テスト終了後にディレクトリを削除

	cfg = &config.Config{
		Tools: []config.Tool{
			{
				Name:    "test",
				Command: []string{"echo", "test"},
				Params: map[string]config.Parameter{
					"param1": {
						Description: "Test parameter",
						Required:    true,
					},
				},
			},
		},
	}

	toolMgr = tool.NewManager(cfg)

	rootCmd := &cobra.Command{Use: "test"}

	AddMCPServerCommand(rootCmd)

	mcpCmd, _, err := rootCmd.Find([]string{"mcp-server"})
	if err != nil {
		t.Fatalf("Failed to find mcp-server command: %v", err)
	}

	if mcpCmd.Name() != "mcp-server" {
		t.Errorf("Expected command name to be 'mcp-server', got '%s'", mcpCmd.Name())
	}

	mcpCmd.SetArgs([]string{})
	mcpCmd.SetOut(&bytes.Buffer{})
	mcpCmd.SetErr(&bytes.Buffer{})
}
