package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
actions:
  - danger_level: high
    type: confirm
    message: "This is a high danger operation. Proceed?"
  - danger_level: medium
    type: timeout
    message: "This is a medium danger operation. Proceeding in 5 seconds."
    timeout: 5
  - danger_level: low
    type: force
    message: "This is a low danger operation."

tools:
  - name: kubectl
    command:
      - kubectl
    params:
      namespace:
        description: The namespace to run the command in
        type: string
        required: true
        validate:
          - danger_level: high
            exclude:
              - kube-system
              - kube-public
    subtools:
      - name: get pod
        args: ["get", "pod", "-o", "json", "-n", "{{.namespace}}"]
      - name: describe pod
        params:
          pod:
            description: The pod to describe
            type: string
            required: true
        args: ["describe", "pod", "{{.pod}}", "-n", "{{.namespace}}"]
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify actions
	if len(cfg.Actions) != 3 {
		t.Errorf("Expected 3 actions, got %d", len(cfg.Actions))
	}

	// Verify tools
	if len(cfg.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(cfg.Tools))
	}

	// Verify tool name
	if cfg.Tools[0].Name != "kubectl" {
		t.Errorf("Expected tool name 'kubectl', got '%s'", cfg.Tools[0].Name)
	}

	// Verify tool command
	if len(cfg.Tools[0].Command) != 1 || cfg.Tools[0].Command[0] != "kubectl" {
		t.Errorf("Expected tool command ['kubectl'], got %v", cfg.Tools[0].Command)
	}

	// Verify tool parameters
	if len(cfg.Tools[0].Params) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(cfg.Tools[0].Params))
	}

	// Verify parameter details
	param, exists := cfg.Tools[0].Params["namespace"]
	if !exists {
		t.Errorf("Expected parameter 'namespace' not found")
	} else {
		if param.Type != "string" {
			t.Errorf("Expected parameter type 'string', got '%s'", param.Type)
		}
		if !param.Required {
			t.Errorf("Expected parameter to be required")
		}
		if len(param.Validate) != 1 {
			t.Errorf("Expected 1 validation rule, got %d", len(param.Validate))
		}
	}

	// Verify subtools
	if len(cfg.Tools[0].Subtools) != 2 {
		t.Errorf("Expected 2 subtools, got %d", len(cfg.Tools[0].Subtools))
	}

	// Verify subtool name
	if cfg.Tools[0].Subtools[0].Name != "get pod" {
		t.Errorf("Expected subtool name 'get pod', got '%s'", cfg.Tools[0].Subtools[0].Name)
	}

	// Verify subtool args
	if len(cfg.Tools[0].Subtools[0].Args) != 6 {
		t.Errorf("Expected 6 args, got %d", len(cfg.Tools[0].Subtools[0].Args))
	}
}

func TestConfigValidate(t *testing.T) {
	// Test valid config with command
	validConfig := &Config{
		Actions: []Action{
			{
				DangerLevel: "high",
				Type:        "confirm",
				Message:     "This is a high danger operation. Proceed?",
			},
		},
		Tools: []Tool{
			{
				Name:    "kubectl",
				Command: []string{"kubectl"},
				Params: map[string]Parameter{
					"namespace": {
						Description: "The namespace to run the command in",
						Type:        "string",
						Required:    true,
					},
				},
				Subtools: []Tool{
					{
						Name: "get pod",
						Args: []string{"get", "pod", "-o", "json", "-n", "{{.namespace}}"},
					},
				},
			},
		},
	}

	if err := validConfig.Validate(); err != nil {
		t.Errorf("Validation failed for valid config with command: %v", err)
	}

	// Test valid config with only subtools
	validConfigWithSubtoolsOnly := &Config{
		Tools: []Tool{
			{
				Name: "parent-tool",
				Params: map[string]Parameter{
					"param1": {
						Description: "A parameter",
						Type:        "string",
						Required:    false,
					},
				},
				Subtools: []Tool{
					{
						Name: "subtool1",
						Args: []string{"arg1", "arg2"},
					},
					{
						Name:   "subtool2",
						Script: "echo 'Hello World'",
					},
				},
			},
		},
	}

	if err := validConfigWithSubtoolsOnly.Validate(); err != nil {
		t.Errorf("Validation failed for valid config with only subtools: %v", err)
	}

	// Test invalid config - missing action type
	invalidConfig1 := &Config{
		Actions: []Action{
			{
				DangerLevel: "high",
				// Missing Type
				Message: "This is a high danger operation. Proceed?",
			},
		},
		Tools: []Tool{
			{
				Name:    "kubectl",
				Command: []string{"kubectl"},
			},
		},
	}

	if err := invalidConfig1.Validate(); err == nil {
		t.Errorf("Validation should fail for config with missing action type")
	}

	// Test invalid config - invalid action type
	invalidConfig2 := &Config{
		Actions: []Action{
			{
				DangerLevel: "high",
				Type:        "invalid",
				Message:     "This is a high danger operation. Proceed?",
			},
		},
		Tools: []Tool{
			{
				Name:    "kubectl",
				Command: []string{"kubectl"},
			},
		},
	}

	if err := invalidConfig2.Validate(); err == nil {
		t.Errorf("Validation should fail for config with invalid action type")
	}

	// Test invalid config - missing tool name
	invalidConfig3 := &Config{
		Actions: []Action{
			{
				DangerLevel: "high",
				Type:        "confirm",
				Message:     "This is a high danger operation. Proceed?",
			},
		},
		Tools: []Tool{
			{
				// Missing Name
				Command: []string{"kubectl"},
			},
		},
	}

	if err := invalidConfig3.Validate(); err == nil {
		t.Errorf("Validation should fail for config with missing tool name")
	}

	// Test invalid config - missing command, script, and subtools
	invalidConfig4 := &Config{
		Actions: []Action{
			{
				DangerLevel: "high",
				Type:        "confirm",
				Message:     "This is a high danger operation. Proceed?",
			},
		},
		Tools: []Tool{
			{
				Name: "kubectl",
				// Missing Command, Script, and has no Subtools
			},
		},
	}

	if err := invalidConfig4.Validate(); err == nil {
		t.Errorf("Validation should fail for config with missing command, script, and subtools")
	}
}

func TestConfigImport(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a base config file
	baseConfigPath := filepath.Join(tempDir, "base_config.yaml")
	baseConfigContent := `
actions:
  - danger_level: high
    type: confirm
    message: "This is a high danger operation from base config."
tools:
  - name: kubectl
    command:
      - kubectl
    params:
      namespace:
        description: The namespace to run the command in
        type: string
        required: true
imports:
  - imported_config.yaml
`
	if err := os.WriteFile(baseConfigPath, []byte(baseConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write base config file: %v", err)
	}

	// Create an imported config file
	importedConfigPath := filepath.Join(tempDir, "imported_config.yaml")
	importedConfigContent := `
actions:
  - danger_level: medium
    type: timeout
    message: "This is a medium danger operation from imported config."
    timeout: 10
tools:
  - name: helm
    command:
      - helm
    params:
      namespace:
        description: The namespace to run the command in
        type: string
        required: true
  - name: kubectl
    command:
      - kubectl
      - --kubeconfig=/other/path
    params:
      namespace:
        description: The namespace to run the command in
        type: string
        required: true
`
	if err := os.WriteFile(importedConfigPath, []byte(importedConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write imported config file: %v", err)
	}

	// Test loading the config with import
	cfg, err := LoadConfig(baseConfigPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify combined actions
	if len(cfg.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(cfg.Actions))
	}

	// Verify action from base config
	if cfg.Actions[0].DangerLevel != "high" || cfg.Actions[0].Type != "confirm" {
		t.Errorf("First action should be from base config")
	}

	// Verify action from imported config
	if cfg.Actions[1].DangerLevel != "medium" || cfg.Actions[1].Type != "timeout" {
		t.Errorf("Second action should be from imported config")
	}

	// Verify tools (base tools should take precedence over imported tools)
	if len(cfg.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(cfg.Tools))
	}

	// Find kubectl tool
	var kubectlTool *Tool
	for i := range cfg.Tools {
		if cfg.Tools[i].Name == "kubectl" {
			kubectlTool = &cfg.Tools[i]
			break
		}
	}

	// Verify kubectl tool from base config (should take precedence)
	if kubectlTool == nil {
		t.Errorf("Expected kubectl tool not found")
	} else if len(kubectlTool.Command) != 1 || kubectlTool.Command[0] != "kubectl" {
		t.Errorf("kubectl command should be from base config, got %v", kubectlTool.Command)
	}

	// Verify imports field is cleared
	if len(cfg.Imports) > 0 {
		t.Errorf("Imports should be cleared after processing")
	}
}

func TestConfigImportMultiple(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a base config file with multiple imports
	baseConfigPath := filepath.Join(tempDir, "base_config.yaml")
	baseConfigContent := `
actions:
  - danger_level: high
    type: confirm
    message: "This is a high danger operation from base config."
imports:
  - imported_config1.yaml
  - imported_config2.yaml
`
	if err := os.WriteFile(baseConfigPath, []byte(baseConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write base config file: %v", err)
	}

	// Create first imported config file
	importedConfigPath1 := filepath.Join(tempDir, "imported_config1.yaml")
	importedConfigContent1 := `
actions:
  - danger_level: medium
    type: timeout
    message: "This is a medium danger operation from imported config 1."
    timeout: 10
tools:
  - name: tool1
    command:
      - tool1
`
	if err := os.WriteFile(importedConfigPath1, []byte(importedConfigContent1), 0644); err != nil {
		t.Fatalf("Failed to write imported config 1 file: %v", err)
	}

	// Create second imported config file
	importedConfigPath2 := filepath.Join(tempDir, "imported_config2.yaml")
	importedConfigContent2 := `
actions:
  - danger_level: low
    type: force
    message: "This is a low danger operation from imported config 2."
tools:
  - name: tool2
    command:
      - tool2
`
	if err := os.WriteFile(importedConfigPath2, []byte(importedConfigContent2), 0644); err != nil {
		t.Fatalf("Failed to write imported config 2 file: %v", err)
	}

	// Test loading the config with multiple imports
	cfg, err := LoadConfig(baseConfigPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify combined actions from all configs
	if len(cfg.Actions) != 3 {
		t.Errorf("Expected 3 actions, got %d", len(cfg.Actions))
	}

	// Verify tools from all configs
	if len(cfg.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(cfg.Tools))
	}

	// Check for tool1 and tool2
	var hasTool1, hasTool2 bool
	for _, tool := range cfg.Tools {
		if tool.Name == "tool1" {
			hasTool1 = true
		} else if tool.Name == "tool2" {
			hasTool2 = true
		}
	}

	if !hasTool1 {
		t.Errorf("Expected tool1 from imported_config1.yaml not found")
	}
	if !hasTool2 {
		t.Errorf("Expected tool2 from imported_config2.yaml not found")
	}
}

func TestConfigImportCircular(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create config file A that imports config B
	configPathA := filepath.Join(tempDir, "config_a.yaml")
	configContentA := `
actions:
  - danger_level: high
    type: confirm
    message: "This is from config A."
imports:
  - config_b.yaml
`
	if err := os.WriteFile(configPathA, []byte(configContentA), 0644); err != nil {
		t.Fatalf("Failed to write config A file: %v", err)
	}

	// Create config file B that imports config A (circular)
	configPathB := filepath.Join(tempDir, "config_b.yaml")
	configContentB := `
actions:
  - danger_level: medium
    type: timeout
    message: "This is from config B."
    timeout: 5
imports:
  - config_a.yaml
`
	if err := os.WriteFile(configPathB, []byte(configContentB), 0644); err != nil {
		t.Fatalf("Failed to write config B file: %v", err)
	}

	// Test loading the config with circular import
	_, err := LoadConfig(configPathA)
	if err == nil {
		t.Errorf("Expected circular import error, but got nil")
	} else if matched := "circular import detected"; err.Error() == "" || !contains(err.Error(), matched) {
		t.Errorf("Expected error message to contain '%s', got: %v", matched, err)
	}
}

func TestConfigImportHierarchical(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create top-level config file
	configPathA := filepath.Join(tempDir, "config_a.yaml")
	configContentA := `
actions:
  - danger_level: high
    type: confirm
    message: "This is from config A."
tools:
  - name: toolA
    command:
      - toolA
imports:
  - config_b.yaml
`
	if err := os.WriteFile(configPathA, []byte(configContentA), 0644); err != nil {
		t.Fatalf("Failed to write config A file: %v", err)
	}

	// Create middle-level config file
	configPathB := filepath.Join(tempDir, "config_b.yaml")
	configContentB := `
actions:
  - danger_level: medium
    type: timeout
    message: "This is from config B."
    timeout: 5
tools:
  - name: toolB
    command:
      - toolB
imports:
  - config_c.yaml
`
	if err := os.WriteFile(configPathB, []byte(configContentB), 0644); err != nil {
		t.Fatalf("Failed to write config B file: %v", err)
	}

	// Create bottom-level config file
	configPathC := filepath.Join(tempDir, "config_c.yaml")
	configContentC := `
actions:
  - danger_level: low
    type: force
    message: "This is from config C."
tools:
  - name: toolC
    command:
      - toolC
`
	if err := os.WriteFile(configPathC, []byte(configContentC), 0644); err != nil {
		t.Fatalf("Failed to write config C file: %v", err)
	}

	// Test loading the config with hierarchical imports
	cfg, err := LoadConfig(configPathA)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify combined actions from all configs
	if len(cfg.Actions) != 3 {
		t.Errorf("Expected 3 actions, got %d", len(cfg.Actions))
	}

	// Verify tools from all configs
	if len(cfg.Tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(cfg.Tools))
	}

	// Check for toolA, toolB, and toolC
	var hasToolA, hasToolB, hasToolC bool
	for _, tool := range cfg.Tools {
		if tool.Name == "toolA" {
			hasToolA = true
		} else if tool.Name == "toolB" {
			hasToolB = true
		} else if tool.Name == "toolC" {
			hasToolC = true
		}
	}

	if !hasToolA {
		t.Errorf("Expected toolA not found")
	}
	if !hasToolB {
		t.Errorf("Expected toolB not found")
	}
	if !hasToolC {
		t.Errorf("Expected toolC not found")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestConfigBuilder_Build(t *testing.T) {
	builder := NewConfigBuilder("../../misc/generate")
	buf := &bytes.Buffer{}
	err := builder.Build(buf)
	if err != nil {
		t.Fatalf("ConfigBuilder.Build failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "actions:") || !strings.Contains(out, "tools:") {
		t.Errorf("output does not contain expected keys: %s", out)
	}
}
