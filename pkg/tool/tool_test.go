package tool

import (
	"os"
	"testing"

	"github.com/takutakahashi/operation-mcp/pkg/config"
)

func TestFindTool(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Tools: []config.Tool{
			{
				Name:    "kubectl",
				Command: []string{"kubectl"},
				Params: map[string]config.Parameter{
					"namespace": {
						Description: "The namespace to run the command in",
						Type:        "string",
						Required:    true,
					},
				},
				Subtools: []config.Subtool{
					{
						Name: "get pod",
						Args: []string{"get", "pod", "-o", "json", "-n", "{{.namespace}}"},
					},
					{
						Name: "describe pod",
						Params: map[string]config.Parameter{
							"pod": {
								Description: "The pod to describe",
								Type:        "string",
								Required:    true,
							},
						},
						Args: []string{"describe", "pod", "{{.pod}}", "-n", "{{.namespace}}"},
					},
					{
						Name:        "delete pod",
						DangerLevel: "high",
						Params: map[string]config.Parameter{
							"pod": {
								Description: "The pod to delete",
								Type:        "string",
								Required:    true,
							},
						},
						Args: []string{"delete", "pod", "{{.pod}}", "-n", "{{.namespace}}"},
					},
				},
			},
			{
				Name:   "script-tool",
				Script: "#!/bin/bash\necho 'This is a script tool'",
				Params: map[string]config.Parameter{
					"param1": {
						Description: "A parameter for the script",
						Type:        "string",
						Required:    true,
					},
				},
				Subtools: []config.Subtool{
					{
						Name:   "script-subtool",
						Script: "#!/bin/bash\necho 'Parameter value: {{.param1}}'",
					},
				},
			},
		},
	}

	// Create a tool manager
	mgr := NewManager(cfg)

	// Test finding root tool
	command, script, params, dangerLevel, err := mgr.FindTool("kubectl")
	if err != nil {
		t.Fatalf("FindTool failed for root tool: %v", err)
	}
	if len(command) != 1 || command[0] != "kubectl" {
		t.Errorf("Expected command ['kubectl'], got %v", command)
	}
	if script != "" {
		t.Errorf("Expected empty script, got '%s'", script)
	}
	if len(params) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(params))
	}
	if dangerLevel != "" {
		t.Errorf("Expected empty danger level, got '%s'", dangerLevel)
	}

	// Test finding subtool
	command, script, params, dangerLevel, err = mgr.FindTool("kubectl_get_pod")
	if err != nil {
		t.Fatalf("FindTool failed for subtool: %v", err)
	}
	if len(command) != 7 {
		t.Errorf("Expected 7 command parts, got %d", len(command))
	}
	if command[0] != "kubectl" || command[1] != "get" || command[2] != "pod" {
		t.Errorf("Expected command starting with ['kubectl', 'get', 'pod'], got %v", command)
	}
	if script != "" {
		t.Errorf("Expected empty script, got '%s'", script)
	}
	if len(params) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(params))
	}
	if dangerLevel != "" {
		t.Errorf("Expected empty danger level, got '%s'", dangerLevel)
	}

	// Test finding subtool with danger level
	command, script, params, dangerLevel, err = mgr.FindTool("kubectl_delete_pod")
	if err != nil {
		t.Fatalf("FindTool failed for subtool with danger level: %v", err)
	}
	if len(command) != 6 {
		t.Errorf("Expected 6 command parts, got %d", len(command))
	}
	if command[0] != "kubectl" || command[1] != "delete" || command[2] != "pod" {
		t.Errorf("Expected command starting with ['kubectl', 'delete', 'pod'], got %v", command)
	}
	if script != "" {
		t.Errorf("Expected empty script, got '%s'", script)
	}
	if len(params) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(params))
	}
	if dangerLevel != "high" {
		t.Errorf("Expected danger level 'high', got '%s'", dangerLevel)
	}

	// Test finding script tool
	command, script, params, dangerLevel, err = mgr.FindTool("script-tool")
	if err != nil {
		t.Fatalf("FindTool failed for script tool: %v", err)
	}
	if len(command) != 0 {
		t.Errorf("Expected empty command, got %v", command)
	}
	if script != "#!/bin/bash\necho 'This is a script tool'" {
		t.Errorf("Expected script content, got '%s'", script)
	}
	if len(params) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(params))
	}
	if dangerLevel != "" {
		t.Errorf("Expected empty danger level, got '%s'", dangerLevel)
	}

	// Test finding script subtool
	command, script, params, dangerLevel, err = mgr.FindTool("script-tool_script-subtool")
	if err != nil {
		t.Fatalf("FindTool failed for script subtool: %v", err)
	}
	if len(command) != 0 {
		t.Errorf("Expected empty command, got %v", command)
	}
	if script != "#!/bin/bash\necho 'Parameter value: {{.param1}}'" {
		t.Errorf("Expected script content with template, got '%s'", script)
	}
	if len(params) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(params))
	}
	if dangerLevel != "" {
		t.Errorf("Expected empty danger level, got '%s'", dangerLevel)
	}

	// Test finding non-existent tool
	_, _, _, _, err = mgr.FindTool("nonexistent")
	if err == nil {
		t.Errorf("FindTool should fail for non-existent tool")
	}

	// Test finding non-existent subtool
	_, _, _, _, err = mgr.FindTool("kubectl_nonexistent")
	if err == nil {
		t.Errorf("FindTool should fail for non-existent subtool")
	}
}

func TestExecuteRawTool(t *testing.T) {
	// Skip test if running in CI environment
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping test in CI environment")
	}

	// Create a test config
	cfg := &config.Config{
		Tools: []config.Tool{
			{
				Name:    "echo",
				Command: []string{"echo"},
				Params: map[string]config.Parameter{
					"message": {
						Description: "The message to echo",
						Type:        "string",
						Required:    true,
					},
				},
				Subtools: []config.Subtool{
					{
						Name: "hello",
						Args: []string{"Hello, {{.message}}!"},
					},
					{
						Name: "goodbye",
						Args: []string{"Goodbye, {{.message}}!"},
					},
				},
			},
			{
				Name:   "script-echo",
				Script: "#!/bin/bash\necho \"Script says: {{.message}}\"",
				Params: map[string]config.Parameter{
					"message": {
						Description: "The message to echo from script",
						Type:        "string",
						Required:    true,
					},
				},
			},
		},
	}

	// Create a tool manager
	mgr := NewManager(cfg)

	// Test executing a valid subtool
	err := mgr.ExecuteRawTool("echo_hello", []string{"--message=World"})
	if err != nil {
		t.Fatalf("ExecuteRawTool failed for echo_hello: %v", err)
	}

	// Test executing another valid subtool
	err = mgr.ExecuteRawTool("echo_goodbye", []string{"--message=World"})
	if err != nil {
		t.Fatalf("ExecuteRawTool failed for echo_goodbye: %v", err)
	}

	// Test executing a script tool
	err = mgr.ExecuteRawTool("script-echo", []string{"--message=ScriptWorld"})
	if err != nil {
		t.Fatalf("ExecuteRawTool failed for script-echo: %v", err)
	}

	// Test executing with invalid tool path
	err = mgr.ExecuteRawTool("nonexistent", []string{})
	if err == nil {
		t.Errorf("ExecuteRawTool should fail for non-existent tool")
	}

	// Test executing with invalid subtool
	err = mgr.ExecuteRawTool("echo_invalid", []string{})
	if err == nil {
		t.Errorf("ExecuteRawTool should fail for non-existent subtool")
	}

	// Test executing without required parameter
	err = mgr.ExecuteRawTool("echo_hello", []string{})
	if err == nil {
		t.Errorf("ExecuteRawTool should fail when required parameter is missing")
	}

	// Test executing script without required parameter
	err = mgr.ExecuteRawTool("script-echo", []string{})
	if err == nil {
		t.Errorf("ExecuteRawTool should fail when required script parameter is missing")
	}
}
