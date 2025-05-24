package tool

import (
	"os"
	"strings"
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
						Params: map[string]config.Parameter{
							"namespace": {
								Description: "The namespace to run the command in",
								Type:        "string",
								Required:    true,
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
	command, script, params, dangerLevel, beforeExec, afterExec, envFrom, err := mgr.FindTool("kubectl")
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
	if len(beforeExec) > 0 {
		t.Errorf("Expected empty beforeExec, got '%v'", beforeExec)
	}
	if len(afterExec) > 0 {
		t.Errorf("Expected empty afterExec, got '%v'", afterExec)
	}
	if len(envFrom.Local) > 0 {
		t.Errorf("Expected empty envFrom.Local, got '%v'", envFrom.Local)
	}

	// Test finding subtool
	command, script, params, dangerLevel, beforeExec, afterExec, envFrom, err = mgr.FindTool("kubectl_get")
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
	if len(beforeExec) > 0 {
		t.Errorf("Expected empty beforeExec, got '%v'", beforeExec)
	}
	if len(afterExec) > 0 {
		t.Errorf("Expected empty afterExec, got '%v'", afterExec)
	}
	if len(envFrom.Local) > 0 {
		t.Errorf("Expected empty envFrom.Local, got '%v'", envFrom.Local)
	}

	// Test finding subtool with danger level
	command, script, params, dangerLevel, beforeExec, afterExec, envFrom, err = mgr.FindTool("kubectl_delete")
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
	if len(beforeExec) > 0 {
		t.Errorf("Expected empty beforeExec, got '%v'", beforeExec)
	}
	if len(afterExec) > 0 {
		t.Errorf("Expected empty afterExec, got '%v'", afterExec)
	}
	if len(envFrom.Local) > 0 {
		t.Errorf("Expected empty envFrom.Local, got '%v'", envFrom.Local)
	}

	// Test finding script tool
	command, script, params, dangerLevel, beforeExec, afterExec, envFrom, err = mgr.FindTool("script-tool")
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
	if len(beforeExec) > 0 {
		t.Errorf("Expected empty beforeExec, got '%v'", beforeExec)
	}
	if len(afterExec) > 0 {
		t.Errorf("Expected empty afterExec, got '%v'", afterExec)
	}
	if len(envFrom.Local) > 0 {
		t.Errorf("Expected empty envFrom.Local, got '%v'", envFrom.Local)
	}

	// Test finding script subtool
	command, script, params, dangerLevel, beforeExec, afterExec, envFrom, err = mgr.FindTool("script-tool_script-subtool")
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
	if len(beforeExec) > 0 {
		t.Errorf("Expected empty beforeExec, got '%v'", beforeExec)
	}
	if len(afterExec) > 0 {
		t.Errorf("Expected empty afterExec, got '%v'", afterExec)
	}
	if len(envFrom.Local) > 0 {
		t.Errorf("Expected empty envFrom.Local, got '%v'", envFrom.Local)
	}

	// Test finding nested subtool
	command, script, params, dangerLevel, beforeExec, afterExec, envFrom, err = mgr.FindTool("parent_child_grandchild")
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
	if len(beforeExec) > 0 {
		t.Errorf("Expected empty beforeExec, got '%v'", beforeExec)
	}
	if len(afterExec) > 0 {
		t.Errorf("Expected empty afterExec, got '%v'", afterExec)
	}
	if len(envFrom.Local) > 0 {
		t.Errorf("Expected empty envFrom.Local, got '%v'", envFrom.Local)
	}

	// Test finding non-existent tool
	_, _, _, _, _, _, _, err = mgr.FindTool("nonexistent")
	if err == nil {
		t.Errorf("FindTool should fail for non-existent tool")
	}

	// Test finding non-existent subtool
	_, _, _, _, _, _, _, err = mgr.FindTool("kubectl_nonexistent")
	if err == nil {
		t.Errorf("FindTool should fail for non-existent subtool")
	}

	// Test finding non-existent nested subtool
	_, _, _, _, _, _, _, err = mgr.FindTool("parent_child_nonexistent")
	if err == nil {
		t.Errorf("FindTool should fail for non-existent nested subtool")
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
	_, err := mgr.ExecuteRawTool("echo_hello", []string{"--message=World"})
	if err != nil {
		t.Fatalf("ExecuteRawTool failed for echo_hello: %v", err)
	}

	// Test executing another valid subtool
	_, err = mgr.ExecuteRawTool("echo_goodbye", []string{"--message=World"})
	if err != nil {
		t.Fatalf("ExecuteRawTool failed for echo_goodbye: %v", err)
	}

	// Test executing a script tool
	_, err = mgr.ExecuteRawTool("script-echo", []string{"--message=ScriptWorld"})
	if err != nil {
		t.Fatalf("ExecuteRawTool failed for script-echo: %v", err)
	}

	// Test executing with invalid tool path
	_, err = mgr.ExecuteRawTool("nonexistent", []string{})
	if err == nil {
		t.Errorf("ExecuteRawTool should fail for non-existent tool")
	}

	// Test executing with invalid subtool
	_, err = mgr.ExecuteRawTool("echo_invalid", []string{})
	if err == nil {
		t.Errorf("ExecuteRawTool should fail for non-existent subtool")
	}

	// Test executing without required parameter
	_, err = mgr.ExecuteRawTool("echo_hello", []string{})
	if err == nil {
		t.Errorf("ExecuteRawTool should fail when required parameter is missing")
	}

	// Test executing script without required parameter
	_, err = mgr.ExecuteRawTool("script-echo", []string{})
	if err == nil {
		t.Errorf("ExecuteRawTool should fail when required script parameter is missing")
	}
}

func TestBeforeAfterExec(t *testing.T) {
	cfg := &config.Config{
		Tools: []config.Tool{
			{
				Name:       "parent",
				BeforeExec: []string{"echo 'parent before'"},
				AfterExec:  []string{"echo 'parent after'"},
				Subtools: []config.Subtool{
					{
						Name:       "child",
						BeforeExec: []string{"echo 'child before'"},
						Script:     "echo 'child main'",
						AfterExec:  []string{"echo 'child after'"},
					},
				},
			},
		},
	}

	mgr := NewManager(cfg)
	mgr.WithExecutor(&mockExecutor{})

	output, err := mgr.ExecuteTool("parent_child", map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "parent before\nchild before\nchild main\nchild after\nparent after"
	if output != expected {
		t.Errorf("expected output %q, got %q", expected, output)
	}
}

type mockExecutor struct{}

func (m *mockExecutor) Execute(command []string) error {
	return nil
}

func (m *mockExecutor) ExecuteWithOutput(command []string) (string, error) {
	return strings.Join(command, " "), nil
}

func (m *mockExecutor) Close() error {
	return nil
}

func TestEnvFromLocal(t *testing.T) {
	// Skip test if running in CI environment
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping test in CI environment")
	}

	os.Setenv("TEST_ENV_VAR1", "value1")
	os.Setenv("TEST_ENV_VAR2", "value2")
	os.Setenv("TEST_ENV_VAR3", "value3")
	os.Setenv("TEST_ENV_VAR4", "value4")
	defer func() {
		os.Unsetenv("TEST_ENV_VAR1")
		os.Unsetenv("TEST_ENV_VAR2")
		os.Unsetenv("TEST_ENV_VAR3")
		os.Unsetenv("TEST_ENV_VAR4")
	}()

	// Create a test config with envFromLocal settings
	cfg := &config.Config{
		Tools: []config.Tool{
			{
				Name:   "env-test",
				Script: "#!/bin/bash\necho \"ENV1=$TEST_ENV_VAR1 ENV2=$TEST_ENV_VAR2 ENV3=$TEST_ENV_VAR3 ENV4=$TEST_ENV_VAR4\"",
				EnvFrom: config.EnvFrom{
					Local: []string{
						"TEST_ENV_VAR1",
						"TEST_ENV_VAR2",
					},
				},

				Subtools: []config.Subtool{
					{
						Name:   "inherit",
						Script: "#!/bin/bash\necho \"ENV1=$TEST_ENV_VAR1 ENV2=$TEST_ENV_VAR2 ENV3=$TEST_ENV_VAR3 ENV4=$TEST_ENV_VAR4\"",
					},
					{
						Name:   "override",
						Script: "#!/bin/bash\necho \"ENV1=$TEST_ENV_VAR1 ENV2=$TEST_ENV_VAR2 ENV3=$TEST_ENV_VAR3 ENV4=$TEST_ENV_VAR4\"",
						EnvFrom: config.EnvFrom{
							Local: []string{
								"TEST_ENV_VAR3",
								"TEST_ENV_VAR4",
							},
						},

					},
				},
			},
			{
				Name:   "no-env",
				Script: "#!/bin/bash\necho \"ENV1=$TEST_ENV_VAR1 ENV2=$TEST_ENV_VAR2 ENV3=$TEST_ENV_VAR3 ENV4=$TEST_ENV_VAR4\"",
			},
		},
	}

	// Create a tool manager
	mgr := NewManager(cfg)

	output, err := mgr.ExecuteTool("env-test", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteTool failed for env-test: %v", err)
	}
	if !strings.Contains(output, "ENV1=value1") || !strings.Contains(output, "ENV2=value2") {
		t.Errorf("Expected output to contain ENV1=value1 and ENV2=value2, got: %s", output)
	}
	if strings.Contains(output, "ENV3=value3") || strings.Contains(output, "ENV4=value4") {
		t.Errorf("Expected output to not contain ENV3=value3 or ENV4=value4, got: %s", output)
	}

	output, err = mgr.ExecuteTool("env-test_inherit", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteTool failed for env-test_inherit: %v", err)
	}
	if !strings.Contains(output, "ENV1=value1") || !strings.Contains(output, "ENV2=value2") {
		t.Errorf("Expected output to contain ENV1=value1 and ENV2=value2, got: %s", output)
	}
	if strings.Contains(output, "ENV3=value3") || strings.Contains(output, "ENV4=value4") {
		t.Errorf("Expected output to not contain ENV3=value3 or ENV4=value4, got: %s", output)
	}

	output, err = mgr.ExecuteTool("env-test_override", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteTool failed for env-test_override: %v", err)
	}
	if strings.Contains(output, "ENV1=value1") || strings.Contains(output, "ENV2=value2") {
		t.Errorf("Expected output to not contain ENV1=value1 or ENV2=value2, got: %s", output)
	}
	if !strings.Contains(output, "ENV3=value3") || !strings.Contains(output, "ENV4=value4") {
		t.Errorf("Expected output to contain ENV3=value3 and ENV4=value4, got: %s", output)
	}

	output, err = mgr.ExecuteTool("no-env", map[string]string{})
	if err != nil {
		t.Fatalf("ExecuteTool failed for no-env: %v", err)
	}
	if !strings.Contains(output, "ENV1=value1") || !strings.Contains(output, "ENV2=value2") ||
		!strings.Contains(output, "ENV3=value3") || !strings.Contains(output, "ENV4=value4") {
		t.Errorf("Expected output to contain all environment variables, got: %s", output)
	}
}
