package tool

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteToolWithBeforeAfter(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := ioutil.TempDir("", "tool_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create parent tool directory
	parentToolDir := filepath.Join(tempDir, "parenttool")
	if err := os.MkdirAll(parentToolDir, 0755); err != nil {
		t.Fatalf("Failed to create parent tool dir: %v", err)
	}

	// Create child tool directory
	childToolDir := filepath.Join(parentToolDir, "childtool")
	if err := os.MkdirAll(childToolDir, 0755); err != nil {
		t.Fatalf("Failed to create child tool dir: %v", err)
	}

	// Create parent tool.yaml
	parentToolYAML := `name: parenttool
description: Parent tool for testing
script: echo 'Parent main script'
beforeExec: echo 'Parent before script'
afterExec: echo 'Parent after script'
`
	if err := ioutil.WriteFile(filepath.Join(parentToolDir, "tool.yaml"), []byte(parentToolYAML), 0644); err != nil {
		t.Fatalf("Failed to write parent tool.yaml: %v", err)
	}

	// Create child tool.yaml
	childToolYAML := `name: childtool
description: Child tool for testing
script: echo 'Child main script'
beforeExec: echo 'Child before script'
afterExec: echo 'Child after script'
`
	if err := ioutil.WriteFile(filepath.Join(childToolDir, "tool.yaml"), []byte(childToolYAML), 0644); err != nil {
		t.Fatalf("Failed to write child tool.yaml: %v", err)
	}

	// Create manager for testing
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test execution of child tool with before/after scripts
	output, err := manager.ExecuteToolWithBeforeAfter("parenttool/childtool", nil)
	if err != nil {
		t.Fatalf("Failed to execute tool: %v", err)
	}

	// Verify execution order through output
	t.Logf("Output: %s", output)
	
	// Check that output contains all scripts executed in correct order
	expectedParts := []string{
		"Parent before script",
		"Child before script",
		"Child main script",
		"Child after script",
		"Parent after script",
	}

	// Check each expected part is in the output and in the correct order
	lastIndex := -1
	for i, part := range expectedParts {
		index := strings.Index(output, part)
		if index == -1 {
			t.Errorf("Output should contain %q but doesn't: %q", part, output)
		} else if index <= lastIndex {
			t.Errorf("Output has %q out of order. Expected after %q", part, expectedParts[i-1])
		}
		lastIndex = index
	}
}