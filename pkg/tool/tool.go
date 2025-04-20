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
	Command     []string
	Script      string
	Params      map[string]config.Parameter
	DangerLevel string
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
		mgr.compiledTools[toolName] = &CompiledTool{
			Command: tool.Command,
			Script:  tool.Script,
			Params:  tool.Params,
		}

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
			Command:     command,
			Script:      subtool.Script,
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
func (m *Manager) FindTool(toolPath string) ([]string, string, map[string]config.Parameter, string, error) {
	tool, exists := m.compiledTools[toolPath]
	if !exists {
		return nil, "", nil, "", fmt.Errorf("tool not found: %s", toolPath)
	}

	return tool.Command, tool.Script, tool.Params, tool.DangerLevel, nil
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

	result := make([]Info, 0, len(m.compiledTools))
	for path, tool := range m.compiledTools {
		// デバッグログの出力
		if os.Getenv("OM_DEBUG") == "true" {
			fmt.Printf("[DEBUG] Tool: %s\n", path)
			fmt.Printf("[DEBUG]   Command: %v\n", tool.Command)
			fmt.Printf("[DEBUG]   Script: %s\n", tool.Script)
			fmt.Printf("[DEBUG]   Params: %v\n", tool.Params)
			fmt.Printf("[DEBUG]   DangerLevel: %s\n", tool.DangerLevel)
		}

		toolInfo := Info{
			Name:        path, // 完全なパスを名前として使用
			Description: "",   // Config doesn't have description field for tools
			Params:      tool.Params,
			Subtools:    []Info{}, // フラット化された構造なので空
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
