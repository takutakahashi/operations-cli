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
	compiledTools map[string]*CompiledTool
}

// CompiledTool represents a compiled tool
type CompiledTool struct {
	Name        string
	Command     []string
	Script      string
	BeforeExec  string
	AfterExec   string
	Params      map[string]config.Parameter
	DangerLevel string
}

// ToolInfo represents additional information about a tool
type ToolInfo struct {
	Command     []string
	Script      string
	Params      map[string]config.Parameter
	DangerLevel string
	BeforeExec  string
	AfterExec   string
}

// NewManager creates a new tool manager
func NewManager(cfg *config.Config) *Manager {
	mgr := &Manager{
		config:        cfg,
		dangerManager: danger.NewManager(cfg.Actions),
		compiledTools: make(map[string]*CompiledTool),
	}

	// Compile all tools and subtools into flat structure
	for _, tool := range cfg.Tools {
		// ルートツール名のスペースをアンダースコアに置換
		toolName := strings.ReplaceAll(tool.Name, " ", "_")
		// Compile root tool
		root := &CompiledTool{
			Name:       tool.Name,
			Command:    tool.Command,
			Script:     tool.Script,
			BeforeExec: tool.BeforeExec,
			AfterExec:  tool.AfterExec,
			Params:     tool.Params,
		}
		mgr.compiledTools[toolName] = root

		// Compile subtools recursively
		mgr.compileSubtools(toolName, tool.Command, tool.Params, tool.Subtools)
	}

	return mgr
}

// compileSubtools recursively compiles subtools into flat structure
func (m *Manager) compileSubtools(parentPath string, parentCommand []string, parentParams map[string]config.Parameter, subtools []config.Subtool) {
	for _, subtool := range subtools {
		// ツール名のスペースをアンダースコアに置換
		subtoolName := strings.ReplaceAll(subtool.Name, " ", "_")
		toolPath := parentPath + "_" + subtoolName

		// Create base command
		var command []string
		if len(subtool.Args) > 0 {
			// If this is a leaf subtool (no nested subtools), combine parent command with args
			if len(subtool.Subtools) == 0 {
				command = make([]string, len(parentCommand))
				copy(command, parentCommand)
				command = append(command, subtool.Args...)
			} else {
				// For non-leaf subtools, just use parent command
				command = make([]string, len(parentCommand))
				copy(command, parentCommand)
			}
		} else {
			// If subtool has no args, just use parent command
			command = make([]string, len(parentCommand))
			copy(command, parentCommand)
		}

		// Merge parameters
		params := make(map[string]config.Parameter)

		// Add subtool parameters
		for name, param := range subtool.Params {
			params[name] = param
		}

		// Add explicitly referenced parent parameters
		for name, paramRef := range subtool.ParamRefs {
			if param, exists := parentParams[name]; exists {
				if paramRef.Required {
					param.Required = true
				}
				params[name] = param
			}
		}

		// Add implicitly referenced parent parameters (used in Args or Script)
		if len(subtool.Args) > 0 || subtool.Script != "" {
			content := strings.Join(subtool.Args, " ") + subtool.Script
			for name, param := range parentParams {
				if strings.Contains(content, "{{."+name+"}}") {
					if _, exists := params[name]; !exists {
						params[name] = param
					}
				}
			}
		}

		// Create compiled tool
		m.compiledTools[toolPath] = &CompiledTool{
			Name:        subtool.Name,
			Command:     command,
			Script:      subtool.Script,
			BeforeExec:  subtool.BeforeExec,
			AfterExec:   subtool.AfterExec,
			Params:      params,
			DangerLevel: subtool.DangerLevel,
		}

		// Recursively compile nested subtools with merged parameters
		m.compileSubtools(toolPath, command, params, subtool.Subtools)
	}
}

// WithExecutor sets the executor for the tool manager
func (m *Manager) WithExecutor(exec executor.Executor) {
	m.execInstance = exec
}

// FindTool finds a tool by its name
func (m *Manager) FindTool(toolPath string) ([]string, string, map[string]config.Parameter, string, string, string, error) {
	tool, exists := m.compiledTools[toolPath]
	if !exists {
		return nil, "", nil, "", "", "", fmt.Errorf("tool not found: %s", toolPath)
	}

	return tool.Command, tool.Script, tool.Params, tool.DangerLevel, tool.BeforeExec, tool.AfterExec, nil
}

// ExecuteTool executes a tool with the given parameters
func (m *Manager) ExecuteTool(toolPath string, paramValues map[string]string) (string, error) {
	// Find the tool
	command, script, params, dangerLevel, beforeExec, afterExec, err := m.FindTool(toolPath)
	if err != nil {
		return "", err
	}

	var outputs []string

	// Get parent tool
	parentTool := m.getParentTool(toolPath)

	if parentTool != nil && parentTool.BeforeExec != "" {
		output, err := executeScript(parentTool.BeforeExec, paramValues)
		if err != nil {
			return "", fmt.Errorf("failed to execute parent beforeExec: %w", err)
		}
		outputs = append(outputs, strings.TrimSpace(output))
	}

	// Execute beforeExec if it exists
	if beforeExec != "" {
		output, err := executeScript(beforeExec, paramValues)
		if err != nil {
			return "", fmt.Errorf("failed to execute beforeExec: %w", err)
		}
		outputs = append(outputs, strings.TrimSpace(output))
	}

	// Validate parameters
	if err := validateParams(params, paramValues); err != nil {
		return "", err
	}

	// Execute the tool
	output, err := m.executeCommand(command, script, paramValues, dangerLevel)
	if err != nil {
		return "", err
	}
	outputs = append(outputs, strings.TrimSpace(output))

	// Execute afterExec if it exists
	if afterExec != "" {
		output, err := executeScript(afterExec, paramValues)
		if err != nil {
			return "", fmt.Errorf("failed to execute afterExec: %w", err)
		}
		outputs = append(outputs, strings.TrimSpace(output))
	}

	// Execute parent afterExec if it exists
	if parentTool != nil && parentTool.AfterExec != "" {
		output, err := executeScript(parentTool.AfterExec, paramValues)
		if err != nil {
			return "", fmt.Errorf("failed to execute parent afterExec: %w", err)
		}
		outputs = append(outputs, strings.TrimSpace(output))
	}

	return strings.Join(outputs, "\n"), nil
}

// getParentTool returns the parent tool of the given tool path
func (m *Manager) getParentTool(toolPath string) *CompiledTool {
	parts := strings.Split(toolPath, "_")
	if len(parts) <= 1 {
		return nil
	}
	parentPath := parts[0]
	return m.compiledTools[parentPath]
}

// executeScript executes a script with the given parameters
func executeScript(script string, paramValues map[string]string) (string, error) {
	// Replace template parameters in script
	if strings.Contains(script, "{{") {
		tmpl, err := template.New("script").Parse(script)
		if err != nil {
			return "", fmt.Errorf("error parsing template in script: %w", err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, paramValues); err != nil {
			return "", fmt.Errorf("error executing template in script: %w", err)
		}

		script = buf.String()
	}

	// Create a temporary file for the script
	tmpFile, err := os.CreateTemp("", "operation-mcp-*.sh")
	if err != nil {
		return "", fmt.Errorf("error creating temporary script file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up temp file when done

	// Write script content to the temporary file
	if _, err := tmpFile.WriteString(script); err != nil {
		return "", fmt.Errorf("error writing script to temporary file: %w", err)
	}

	// Close the file before execution
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("error closing temporary script file: %w", err)
	}

	// Make the script file executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return "", fmt.Errorf("error making script file executable: %w", err)
	}

	// Execute the script
	// No output to stdout for mcp-server

	// Run the script with bash to ensure compatibility
	cmd := exec.Command("/bin/bash", tmpFile.Name())
	cmd.Stdin = os.Stdin

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing script: %w\noutput: %s", err, string(output))
	}

	return string(output), nil
}

// ExecuteRawTool executes a tool with the given raw arguments
func (m *Manager) ExecuteRawTool(toolPath string, args []string) (string, error) {
	// Find the tool and subtool
	command, script, params, dangerLevel, _, _, err := m.FindTool(toolPath)
	if err != nil {
		return "", err
	}

	// Extract parameter values from the command-line arguments
	paramValues := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--set") {
			// Handle --set key=value format
			if i+1 < len(args) {
				parts := strings.SplitN(args[i+1], "=", 2)
				if len(parts) == 2 {
					paramValues[parts[0]] = parts[1]
				}
				i++ // Skip the next arg since it's the value
			}
		} else if strings.HasPrefix(arg, "-") {
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
			return "", err
		}
		if !proceed {
			return "", fmt.Errorf("operation aborted due to danger level check")
		}
	}

	// Validate required parameters
	for name, param := range params {
		if param.Required {
			value, exists := paramValues[name]
			if !exists || value == "" {
				return "", fmt.Errorf("required parameter missing: %s", name)
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
				return "", fmt.Errorf("error parsing template in argument: %w", err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, paramValues); err != nil {
				return "", fmt.Errorf("error executing template in argument: %w", err)
			}

			finalCommand[i] = buf.String()
		} else {
			finalCommand[i] = arg
		}
	}

	// Execute the command
	// No output to stdout for mcp-server
	cmd := exec.Command(finalCommand[0], finalCommand[1:]...)
	cmd.Stdin = os.Stdin

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing command: %w\noutput: %s", err, string(output))
	}

	return string(output), nil
}

// ListTools returns all tools and subtools defined in the config
func (m *Manager) ListTools() map[string]config.Tool {
	if _, _, _, _, _, _, err := m.FindTool("list"); err != nil {
		return make(map[string]config.Tool)
	}

	result := make(map[string]config.Tool, len(m.compiledTools))
	for path, tool := range m.compiledTools {
		parts := strings.Split(path, ".")
		name := strings.ReplaceAll(parts[len(parts)-1], " ", "_")

		// Find the original tool in config to get its subtools
		var subtools []config.Subtool
		for _, configTool := range m.config.Tools {
			if configTool.Name == name {
				subtools = configTool.Subtools
				break
			}
		}

		toolInfo := config.Tool{
			Name:     name,
			Command:  tool.Command,
			Script:   tool.Script,
			Params:   tool.Params,
			Subtools: subtools,
		}

		result[path] = toolInfo
	}

	return result
}

// GetConfig returns the configuration used by this manager
func (m *Manager) GetConfig() *config.Config {
	return m.config
}

// GetCompiledTools returns the compiled tools map
func (m *Manager) GetCompiledTools() map[string]*CompiledTool {
	return m.compiledTools
}

func (m *Manager) GetToolInfo(toolPath string) (*ToolInfo, error) {
	command, script, params, dangerLevel, beforeExec, afterExec, err := m.FindTool(toolPath)
	if err != nil {
		return nil, err
	}
	return &ToolInfo{
		Command:     command,
		Script:      script,
		Params:      params,
		DangerLevel: dangerLevel,
		BeforeExec:  beforeExec,
		AfterExec:   afterExec,
	}, nil
}

func validateParams(params map[string]config.Parameter, paramValues map[string]string) error {
	// Validate required parameters
	for name, param := range params {
		if param.Required {
			value, exists := paramValues[name]
			if !exists || value == "" {
				return fmt.Errorf("required parameter missing: %s", name)
			}
		}
	}
	return nil
}

func (m *Manager) executeCommand(command []string, script string, paramValues map[string]string, dangerLevel string) (string, error) {
	// Check danger level for the tool itself
	if dangerLevel != "" {
		proceed, err := m.dangerManager.CheckDangerLevel(dangerLevel, "", "", nil)
		if err != nil {
			return "", err
		}
		if !proceed {
			return "", fmt.Errorf("operation aborted due to danger level check")
		}
	}

	var result string
	var err error
	// Execute the main script or command
	if script != "" {
		result, err = executeScript(script, paramValues)
		if err != nil {
			return result, err
		}
	} else if len(command) > 0 {
		// Replace template parameters in command args
		finalCommand := make([]string, len(command))
		for i, arg := range command {
			if strings.Contains(arg, "{{") {
				tmpl, err := template.New("arg").Parse(arg)
				if err != nil {
					return "", fmt.Errorf("error parsing template in argument: %w", err)
				}

				var buf bytes.Buffer
				if err := tmpl.Execute(&buf, paramValues); err != nil {
					return "", fmt.Errorf("error executing template in argument: %w", err)
				}

				finalCommand[i] = buf.String()
			} else {
				finalCommand[i] = arg
			}
		}
		result, err = m.execInstance.ExecuteWithOutput(finalCommand)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}
