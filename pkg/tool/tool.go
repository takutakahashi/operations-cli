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
	"github.com/takutakahashi/operation-mcp/pkg/logger"
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
	logger        logger.Logger
}

// CompiledTool represents a compiled tool
type CompiledTool struct {
	Name         string
	Description  string
	Command      []string
	Script       string
	BeforeExec   []string
	AfterExec    []string
	Params       map[string]config.Parameter
	EnvFrom      config.EnvFrom
	EnvFromLocal []string // Deprecated: Use EnvFrom.Local instead
	DangerLevel  string
}

// ToolInfo represents additional information about a tool
type ToolInfo struct {
	Command      []string
	Script       string
	Params       map[string]config.Parameter
	DangerLevel  string
	BeforeExec   []string
	AfterExec    []string
	EnvFrom      config.EnvFrom
	EnvFromLocal []string // Deprecated: Use EnvFrom.Local instead
}

// NewManager creates a new tool manager
func NewManager(cfg *config.Config) *Manager {
	mgr := &Manager{
		config:        cfg,
		dangerManager: danger.NewManager(cfg.Actions),
		compiledTools: make(map[string]*CompiledTool),
		logger:        logger.NewNullLogger(), // デフォルトはログを出力しない
	}

	// Compile all tools and subtools into flat structure
	for _, tool := range cfg.Tools {
		if tool.Enabled != nil && !*tool.Enabled {
			continue
		}
		
		// ルートツール名のスペースをアンダースコアに置換
		toolName := strings.ReplaceAll(tool.Name, " ", "_")
		// Compile root tool
		root := &CompiledTool{
			Name:         tool.Name,
			Description:  tool.Description,
			Command:      tool.Command,
			Script:       tool.Script,
			BeforeExec:   tool.BeforeExec,
			AfterExec:    tool.AfterExec,
			Params:       tool.Params,
			EnvFrom:      tool.EnvFrom,
			EnvFromLocal: tool.EnvFromLocal,
		}
		mgr.compiledTools[toolName] = root

		// Compile subtools recursively
		mgr.compileSubtools(toolName, tool.Command, tool.Params, tool.EnvFrom, tool.EnvFromLocal, tool.Subtools)
	}

	return mgr
}

// compileSubtools recursively compiles subtools into flat structure
func (m *Manager) compileSubtools(parentPath string, parentCommand []string, parentParams map[string]config.Parameter, parentEnvFrom config.EnvFrom, parentEnvFromLocal []string, subtools []config.Subtool) {
	for _, subtool := range subtools {
		if subtool.Enabled != nil && !*subtool.Enabled {
			continue
		}
		
		// ツール名のスペースをアンダースコアに置換
		subtoolName := strings.ReplaceAll(subtool.Name, " ", "_")
		toolPath := parentPath + "_" + subtoolName
		toolDescription := subtool.Description

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
			Name:         subtool.Name,
			Description:  toolDescription,
			Command:      command,
			Script:       subtool.Script,
			BeforeExec:   subtool.BeforeExec,
			AfterExec:    subtool.AfterExec,
			Params:       params,
			EnvFrom:      mergeEnvFrom(parentEnvFrom, subtool.EnvFrom),
			EnvFromLocal: mergeEnvFromLocal(parentEnvFromLocal, subtool.EnvFromLocal),
			DangerLevel:  subtool.DangerLevel,
		}

		// Recursively compile nested subtools with merged parameters
		m.compileSubtools(toolPath, command, params, mergeEnvFrom(parentEnvFrom, subtool.EnvFrom), mergeEnvFromLocal(parentEnvFromLocal, subtool.EnvFromLocal), subtool.Subtools)
	}
}

func mergeEnvFromLocal(parent, subtool []string) []string {
	if len(subtool) > 0 {
		return subtool
	}
	return parent
}

func mergeEnvFrom(parent, subtool config.EnvFrom) config.EnvFrom {
	if len(subtool.Local) > 0 {
		return subtool
	}
	return parent
}

// WithExecutor sets the executor for the tool manager
func (m *Manager) WithExecutor(exec executor.Executor) {
	m.execInstance = exec
}

// WithLogger sets the logger for the tool manager
func (m *Manager) WithLogger(l logger.Logger) {
	m.logger = l
}

// FindTool finds a tool by its name
func (m *Manager) FindTool(toolPath string) ([]string, string, map[string]config.Parameter, string, []string, []string, config.EnvFrom, error) {
	tool, exists := m.compiledTools[toolPath]
	if !exists {
		return nil, "", nil, "", nil, nil, config.EnvFrom{}, fmt.Errorf("tool not found: %s", toolPath)
	}

	if len(tool.EnvFrom.Local) == 0 && len(tool.EnvFromLocal) > 0 {
		tool.EnvFrom.Local = tool.EnvFromLocal
	}

	return tool.Command, tool.Script, tool.Params, tool.DangerLevel, tool.BeforeExec, tool.AfterExec, tool.EnvFrom, nil
}

// ExecuteTool executes a tool with the given parameters
func (m *Manager) ExecuteTool(toolPath string, paramValues map[string]string) (string, error) {
	// Log tool execution
	m.logger.Debug("Executing tool: %s", toolPath)
	m.logger.Debug("Parameters: %v", paramValues)

	// Find the tool
	command, script, params, dangerLevel, beforeExec, afterExec, envFrom, err := m.FindTool(toolPath)
	if err != nil {
		return "", err
	}

	var outputs []string

	// Get parent tool
	parentTool := m.getParentTool(toolPath)

	// 親のbeforeExecを順に実行
	if parentTool != nil && parentTool.BeforeExec != nil {
		for _, script := range parentTool.BeforeExec {
			m.logger.Debug("Executing parent beforeExec: %s", script)
			output, err := executeScript(script, paramValues, envFrom)
			if err != nil {
				return "", fmt.Errorf("failed to execute parent beforeExec: %w", err)
			}
			outputs = append(outputs, strings.TrimSpace(output))
		}
	}

	// beforeExecを順に実行
	for _, script := range beforeExec {
		m.logger.Debug("Executing beforeExec: %s", script)
		output, err := executeScript(script, paramValues, envFrom)
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
	output, err := m.executeCommand(command, script, paramValues, dangerLevel, envFrom)
	if err != nil {
		return "", err
	}
	outputs = append(outputs, strings.TrimSpace(output))

	// afterExecを順に実行
	for _, script := range afterExec {
		m.logger.Debug("Executing afterExec: %s", script)
		output, err := executeScript(script, paramValues, envFrom)
		if err != nil {
			return "", fmt.Errorf("failed to execute afterExec: %w", err)
		}
		outputs = append(outputs, strings.TrimSpace(output))
	}

	// 親のafterExecを順に実行
	if parentTool != nil && parentTool.AfterExec != nil {
		for _, script := range parentTool.AfterExec {
			m.logger.Debug("Executing parent afterExec: %s", script)
			output, err := executeScript(script, paramValues, envFrom)
			if err != nil {
				return "", fmt.Errorf("failed to execute parent afterExec: %w", err)
			}
			outputs = append(outputs, strings.TrimSpace(output))
		}
	}

	m.logger.Debug("Tool execution completed successfully")
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
func executeScript(script string, paramValues map[string]string, envFrom config.EnvFrom) (string, error) {
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
	
	if len(envFrom.Local) > 0 {
		cmd.Env = []string{}
		for _, envVar := range envFrom.Local {
			if value := os.Getenv(envVar); value != "" {
				cmd.Env = append(cmd.Env, envVar+"="+value)
			}
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing script: %w\noutput: %s", err, string(output))
	}

	return string(output), nil
}

// ExecuteRawTool executes a tool with the given raw arguments
func (m *Manager) ExecuteRawTool(toolPath string, args []string) (string, error) {
	// Find the tool and subtool
	command, script, params, dangerLevel, _, _, envFrom, err := m.FindTool(toolPath)
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
		return executeScript(script, paramValues, envFrom)
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
	if _, _, _, _, _, _, _, err := m.FindTool("list"); err != nil {
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
			Name:         name,
			Command:      tool.Command,
			Script:       tool.Script,
			Params:       tool.Params,
			EnvFrom:      tool.EnvFrom,
			EnvFromLocal: tool.EnvFromLocal,
			Subtools:     subtools,
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
	filteredTools := make(map[string]*CompiledTool)
	for name, tool := range m.compiledTools {
		// command、arg、scriptのいずれも持たないツールは除外
		if len(tool.Command) == 0 && tool.Script == "" {
			continue
		}
		filteredTools[name] = tool
	}
	return filteredTools
}

func (m *Manager) GetToolInfo(toolPath string) (*ToolInfo, error) {
	command, script, params, dangerLevel, beforeExec, afterExec, envFrom, err := m.FindTool(toolPath)
	if err != nil {
		return nil, err
	}
	return &ToolInfo{
		Command:      command,
		Script:       script,
		Params:       params,
		DangerLevel:  dangerLevel,
		BeforeExec:   beforeExec,
		AfterExec:    afterExec,
		EnvFrom:      envFrom,
		EnvFromLocal: envFrom.Local,
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

func (m *Manager) executeCommand(command []string, script string, paramValues map[string]string, dangerLevel string, envFrom config.EnvFrom) (string, error) {
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
		result, err = executeScript(script, paramValues, envFrom)
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

// DescribeTool returns detailed information about a tool
func (m *Manager) DescribeTool(toolPath string) (*Info, error) {
	tool, exists := m.compiledTools[toolPath]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", toolPath)
	}

	// 実行可能なツールかどうかをチェック
	if len(tool.Command) == 0 && tool.Script == "" {
		return nil, fmt.Errorf("tool is not executable: %s", toolPath)
	}

	return &Info{
		Name:        tool.Name,
		Description: tool.Description,
		Params:      tool.Params,
	}, nil
}
