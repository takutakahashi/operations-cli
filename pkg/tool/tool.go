package tool

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/takutakahashi/operation-mcp/pkg/config"
	"github.com/takutakahashi/operation-mcp/pkg/danger"
	"github.com/takutakahashi/operation-mcp/pkg/executor"
)

// Info represents a tool or subtool for hierarchical display
type Info struct {
	Name        string
	Description string
	Params      map[string]config.Parameter
	Subtools    []Info
}

// Manager handles tool execution
type Manager struct {
	config        *config.Config
	dangerManager *danger.Manager
	execInstance  executor.Executor
}

// NewManager creates a new tool manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:        cfg,
		dangerManager: danger.NewManager(cfg.Actions),
	}
}

// WithExecutor sets the executor for the tool manager
func (m *Manager) WithExecutor(exec executor.Executor) {
	m.execInstance = exec
}

// FindTool finds a tool by its name
func (m *Manager) FindTool(toolPath string) ([]string, string, map[string]config.Parameter, string, error) {
	parts := strings.Split(toolPath, "_")
	if len(parts) < 1 {
		return nil, "", nil, "", fmt.Errorf("invalid tool path: %s", toolPath)
	}

	// Find the root tool
	var rootTool *config.Tool
	for i := range m.config.Tools {
		if m.config.Tools[i].Name == parts[0] {
			rootTool = &m.config.Tools[i]
			break
		}
	}

	if rootTool == nil {
		return nil, "", nil, "", fmt.Errorf("tool not found: %s", parts[0])
	}

	// Start with the root tool's command or script
	command := make([]string, 0)
	script := ""
	if len(rootTool.Command) > 0 {
		command = make([]string, len(rootTool.Command))
		copy(command, rootTool.Command)
	} else if rootTool.Script != "" {
		script = rootTool.Script
	}

	// Collect all parameters
	params := make(map[string]config.Parameter)
	for name, param := range rootTool.Params {
		params[name] = param
	}

	// If we only have the root tool, return it
	if len(parts) == 1 {
		return command, script, params, "", nil
	}

	// Navigate through the subtool hierarchy
	currentSubtools := rootTool.Subtools
	var currentSubtool *config.Subtool
	dangerLevel := ""

	// For each part of the path after the root tool
	for _, part := range parts[1:] {
		found := false
		// Look for a matching subtool at the current level
		for j := range currentSubtools {
			subtoolName := strings.ReplaceAll(currentSubtools[j].Name, " ", "_")
			if subtoolName == part {
				currentSubtool = &currentSubtools[j]
				// Add parameters from this level
				for name, param := range currentSubtool.Params {
					params[name] = param
				}

				// Add parameters referenced by this subtool
				for name, paramRef := range currentSubtool.ParamRefs {
					if param, exists := rootTool.Params[name]; exists {
						if paramRef.Required {
							param.Required = true
						}
						params[name] = param
					}
				}

				// Update danger level if specified at this level
				if currentSubtool.DangerLevel != "" {
					dangerLevel = currentSubtool.DangerLevel
				}

				// Move to the next level in the hierarchy
				currentSubtools = currentSubtool.Subtools
				found = true
				break
			}
		}

		if !found {
			return nil, "", nil, "", fmt.Errorf("subtool not found: %s in path %s", part, toolPath)
		}
	}

	// Now currentSubtool is the deepest subtool we found
	// Add the args from the final subtool or override script
	if currentSubtool != nil {
		if len(currentSubtool.Args) > 0 {
			command = append(command, currentSubtool.Args...)
		} else if currentSubtool.Script != "" {
			// Subtool script overrides tool script if both exist
			script = currentSubtool.Script
			// Reset command if script is specified
			if len(command) > 0 {
				command = []string{}
			}
		}
	}

	return command, script, params, dangerLevel, nil
}

// ExecuteTool executes a tool with the given parameters
func (m *Manager) ExecuteTool(toolPath string, paramValues map[string]string) error {
	// Find the tool
	command, script, params, dangerLevel, err := m.FindTool(toolPath)
	if err != nil {
		return err
	}

	// Validate required parameters
	for name, param := range params {
		if param.Required {
			value, exists := paramValues[name]
			if !exists || value == "" {
				return fmt.Errorf("required parameter missing: %s", name)
			}
		}
	}

	// Check danger level for parameters with validation rules
	for name, param := range params {
		value, exists := paramValues[name]
		if exists && len(param.Validate) > 0 {
			for _, validation := range param.Validate {
				proceed, err := m.dangerManager.CheckDangerLevel(
					validation.DangerLevel,
					name,
					value,
					param.Validate,
				)
				if err != nil {
					return err
				}
				if !proceed {
					return fmt.Errorf("operation aborted due to danger level check")
				}
			}
		}
	}

	// Check danger level for the tool itself
	if dangerLevel != "" {
		proceed, err := m.dangerManager.CheckDangerLevel(dangerLevel, "", "", nil)
		if err != nil {
			return err
		}
		if !proceed {
			return fmt.Errorf("operation aborted due to danger level check")
		}
	}

	// スクリプトが指定されている場合はそれを実行
	if script != "" {
		return executeScript(script, paramValues)
	}

	// コマンドが指定されている場合は従来通り実行
	// Replace template parameters in command args
	finalCommand := make([]string, len(command))
	for i, arg := range command {
		if strings.Contains(arg, "{{") {
			tmpl, err := template.New("arg").Parse(arg)
			if err != nil {
				return fmt.Errorf("error parsing template in argument: %w", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, paramValues); err != nil {
				return fmt.Errorf("error executing template in argument: %w", err)
			}

			finalCommand[i] = buf.String()
		} else {
			finalCommand[i] = arg
		}
	}

	// Execute the command
	fmt.Printf("Executing: %s\n", strings.Join(finalCommand, " "))
	cmd := exec.Command(finalCommand[0], finalCommand[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// executeScript executes a script with the given parameters
func executeScript(script string, paramValues map[string]string) error {
	// Replace template parameters in script
	if strings.Contains(script, "{{") {
		tmpl, err := template.New("script").Parse(script)
		if err != nil {
			return fmt.Errorf("error parsing template in script: %w", err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, paramValues); err != nil {
			return fmt.Errorf("error executing template in script: %w", err)
		}

		script = buf.String()
	}

	// Create a temporary file for the script
	tmpFile, err := os.CreateTemp("", "operation-mcp-*.sh")
	if err != nil {
		return fmt.Errorf("error creating temporary script file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up temp file when done

	// Write script content to the temporary file
	if _, err := tmpFile.WriteString(script); err != nil {
		return fmt.Errorf("error writing script to temporary file: %w", err)
	}

	// Close the file before execution
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error closing temporary script file: %w", err)
	}

	// Make the script file executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("error making script file executable: %w", err)
	}

	// Execute the script
	fmt.Printf("Executing script: %s\n", tmpFile.Name())

	// Run the script with bash to ensure compatibility
	cmd := exec.Command("/bin/bash", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// ExecuteRawTool executes a tool with the given raw arguments
func (m *Manager) ExecuteRawTool(toolPath string, args []string) error {
	// Find the tool and subtool
	command, script, params, dangerLevel, err := m.FindTool(toolPath)
	if err != nil {
		return err
	}

	// Extract parameter values from the command-line arguments
	paramValues := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			paramName := strings.TrimLeft(arg, "-")
			// Handle --param=value format
			if strings.Contains(paramName, "=") {
				parts := strings.SplitN(paramName, "=", 2)
				paramName = parts[0]
				paramValues[paramName] = parts[1]
				continue
			}

			// Handle -p value format
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				paramValues[paramName] = args[i+1]
				i++ // Skip the next arg since it's the value
			} else {
				// Handle boolean flags like -f
				paramValues[paramName] = "true"
			}
		}
	}

	// Check danger level for the subtool
	if dangerLevel != "" {
		proceed, err := m.dangerManager.CheckDangerLevel(dangerLevel, "", "", nil)
		if err != nil {
			return err
		}
		if !proceed {
			return fmt.Errorf("operation aborted due to danger level check")
		}
	}

	// Validate required parameters
	for name, param := range params {
		if param.Required {
			value, exists := paramValues[name]
			if !exists || value == "" {
				return fmt.Errorf("required parameter missing: %s", name)
			}
		}
	}

	// スクリプトが指定されている場合はそれを実行
	if script != "" {
		return executeScript(script, paramValues)
	}

	// コマンドが指定されている場合は従来通り実行
	// Replace template parameters in command args
	finalCommand := make([]string, len(command))
	for i, arg := range command {
		if strings.Contains(arg, "{{") {
			tmpl, err := template.New("arg").Parse(arg)
			if err != nil {
				return fmt.Errorf("error parsing template in argument: %w", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, paramValues); err != nil {
				return fmt.Errorf("error executing template in argument: %w", err)
			}

			finalCommand[i] = buf.String()
		} else {
			finalCommand[i] = arg
		}
	}

	// Execute the command
	fmt.Printf("Executing: %s\n", strings.Join(finalCommand, " "))
	cmd := exec.Command(finalCommand[0], finalCommand[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// ListTools returns all tools and subtools defined in the config
func (m *Manager) ListTools() []Info {
	if m.config == nil || len(m.config.Tools) == 0 {
		return []Info{}
	}

	result := make([]Info, 0, len(m.config.Tools))

	for _, tool := range m.config.Tools {
		toolInfo := Info{
			Name:        tool.Name,
			Description: "", // Config doesn't have description field for tools
			Params:      tool.Params,
			Subtools:    make([]Info, 0, len(tool.Subtools)),
		}

		// Add subtools recursively
		for _, subtool := range tool.Subtools {
			toolInfo.Subtools = append(toolInfo.Subtools, convertSubtoolToInfo(subtool, tool.Name))
		}

		result = append(result, toolInfo)
	}

	return result
}

// convertSubtoolToInfo converts a subtool configuration to Info structure
func convertSubtoolToInfo(subtool config.Subtool, parentName string) Info {
	name := strings.ReplaceAll(subtool.Name, " ", "_")

	toolInfo := Info{
		Name:        name,
		Description: "", // Config doesn't have description field for subtools
		Params:      subtool.Params,
		Subtools:    make([]Info, 0, len(subtool.Subtools)),
	}

	// Add nested subtools recursively
	for _, nested := range subtool.Subtools {
		toolInfo.Subtools = append(toolInfo.Subtools,
			convertSubtoolToInfo(nested, parentName+"_"+name))
	}

	return toolInfo
}

// GetConfig returns the configuration used by this manager
func (m *Manager) GetConfig() *config.Config {
	return m.config
}
