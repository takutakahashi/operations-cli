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
						Name: "get",
						Args: []string{"get", "pod", "-o", "json", "-n", "{{.namespace}}"},
						ParamRefs: map[string]config.ParamRef{
							"namespace": {
								Required: true,
							},
						},
					},
					{
						Name: "describe",
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
						Name:        "delete",
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
			{
				Name:    "parent",
				Command: []string{"echo", "parent"},
				Params: map[string]config.Parameter{
					"parent-param": {
						Description: "A parameter for the parent",
						Type:        "string",
						Required:    false,
					},
				},
				Subtools: []config.Subtool{
					{
						Name: "child",
						Args: []string{"child", "{{.parent-param}}"},
						Params: map[string]config.Parameter{
							"child-param": {
								Description: "A parameter for the child",
								Type:        "string",
								Required:    false,
							},
						},
						Subtools: []config.Subtool{
							{
								Name: "grandchild",
								Args: []string{"grandchild", "{{.parent-param}}", "{{.child-param}}", "{{.grandchild-param}}"},
								Params: map[string]config.Parameter{
									"grandchild-param": {
										Description: "A parameter for the grandchild",
										Type:        "string",
										Required:    false,
									},
								},
							},
						},
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
	command, script, params, dangerLevel, err = mgr.FindTool("kubectl_get")
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
	command, script, params, dangerLevel, err = mgr.FindTool("kubectl_delete")
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

	// Test finding nested subtool
	command, script, params, dangerLevel, err = mgr.FindTool("parent_child_grandchild")
	if err != nil {
		t.Fatalf("FindTool failed for nested subtool: %v", err)
	}
	if len(command) != 6 {
		t.Errorf("Expected 6 command parts, got %d: %v", len(command), command)
	}
	if command[0] != "echo" || command[1] != "parent" || command[2] != "grandchild" {
		t.Errorf("Expected command starting with ['echo', 'parent', 'grandchild'], got %v", command)
	}
	if script != "" {
		t.Errorf("Expected empty script, got '%s'", script)
	}
	if len(params) != 3 {
		t.Errorf("Expected 3 parameters, got %d", len(params))
	}
	_, hasParentParam := params["parent-param"]
	if !hasParentParam {
		t.Errorf("Expected parent-param to be included in parameters")
	}
	_, hasChildParam := params["child-param"]
	if !hasChildParam {
		t.Errorf("Expected child-param to be included in parameters")
	}
	_, hasGrandchildParam := params["grandchild-param"]
	if !hasGrandchildParam {
		t.Errorf("Expected grandchild-param to be included in parameters")
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

	// Test finding non-existent nested subtool
	_, _, _, _, err = mgr.FindTool("parent_child_nonexistent")
	if err == nil {
		t.Errorf("FindTool should fail for non-existent nested subtool")
	}
}

func TestFindToolWithParamRefs(t *testing.T) {
	// Create a test config with root tools containing params and subtools with param_refs
	cfg := &config.Config{
		Tools: []config.Tool{
			{
				Name:    "rootcmd",
				Command: []string{"rootcmd"},
				Params: map[string]config.Parameter{
					"global-param": {
						Description: "A global parameter",
						Type:        "string",
						Required:    false,
					},
					"optional-param": {
						Description: "An optional parameter",
						Type:        "string",
						Required:    false,
					},
				},
				Subtools: []config.Subtool{
					{
						Name: "subcmd",
						Args: []string{"subcmd", "{{.global-param}}"},
						ParamRefs: map[string]config.ParamRef{
							"global-param": {
								Required: true, // Override to make it required for this subtool
							},
						},
					},
					{
						Name: "anothercmd",
						Args: []string{"anothercmd", "{{.global-param}}", "{{.optional-param}}"},
						ParamRefs: map[string]config.ParamRef{
							"global-param": {
								Required: true,
							},
							"optional-param": {
								Required: false, // Keep it optional
							},
						},
					},
				},
			},
		},
	}

	// Create a tool manager
	mgr := NewManager(cfg)

	// Test finding root tool with parameters
	command, script, params, dangerLevel, err := mgr.FindTool("rootcmd")
	if err != nil {
		t.Fatalf("FindTool failed for root tool with params: %v", err)
	}
	if len(params) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(params))
	}
	if param, exists := params["global-param"]; !exists || param.Required {
		t.Errorf("Expected global-param to exist and be not required at root level")
	}

	// Test finding subtool with param_refs
	command, script, params, dangerLevel, err = mgr.FindTool("rootcmd_subcmd")
	if err != nil {
		t.Fatalf("FindTool failed for subtool with param_refs: %v", err)
	}
	if len(params) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(params))
	}
	if param, exists := params["global-param"]; !exists || !param.Required {
		t.Errorf("Expected global-param to exist and be required for subtool")
	}

	// Test finding another subtool with multiple param_refs
	command, script, params, dangerLevel, err = mgr.FindTool("rootcmd_anothercmd")
	if err != nil {
		t.Fatalf("FindTool failed for subtool with multiple param_refs: %v", err)
	}
	if len(params) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(params))
	}
	if param, exists := params["global-param"]; !exists || !param.Required {
		t.Errorf("Expected global-param to exist and be required")
	}
	if param, exists := params["optional-param"]; !exists || param.Required {
		t.Errorf("Expected optional-param to exist and be not required")
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
