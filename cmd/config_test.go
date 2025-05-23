package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigInitAll(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-init-all-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	configInitOutput = tempDir
	
	err = configInitAllCmd.RunE(configInitAllCmd, []string{})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	
	paths := []string{
		filepath.Join(tempDir, "metadata.yaml"),
		filepath.Join(tempDir, "tools/example/metadata.yaml"),
		filepath.Join(tempDir, "tools/example/main.sh"),
	}
	
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", path)
		}
	}
}

func TestConfigInitTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-init-tool-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	configInitOutput = tempDir
	
	err = configInitToolCmd.RunE(configInitToolCmd, []string{"test-tool"})
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	
	paths := []string{
		filepath.Join(tempDir, "test-tool/metadata.yaml"),
		filepath.Join(tempDir, "test-tool/main.sh"),
	}
	
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", path)
		}
	}
}
