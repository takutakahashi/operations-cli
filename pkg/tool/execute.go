package tool

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/takutakahashi/operation-mcp/pkg/config"
)

// ExecuteToolWithBeforeAfter executes a tool with before and after scripts
func (m *Manager) ExecuteToolWithBeforeAfter(toolPath string, paramValues map[string]string) (string, error) {
	tool, script, params, description, err := m.FindTool(toolPath)
	if err != nil {
		return "", err
	}
	
	// In this enhanced version, we extract BeforeExec and AfterExec from the tool
	// and execute them in the correct order
	
	// Create a full toolInfo struct to work with
	toolInfo := &Info{
		Name:        filepath.Base(toolPath),
		Description: description,
		Path:        toolPath,
		Script:      script,
		// Extract BeforeExec and AfterExec
		BeforeExec:  extractScript("beforeExec", tool),
		AfterExec:   extractScript("afterExec", tool),
	}
	
	// Find parent tool if exists
	var parentInfo *Info
	if strings.Contains(toolPath, "/") {
		parentPath := filepath.Dir(toolPath)
		parentTool, parentScript, _, parentDesc, _ := m.FindTool(parentPath)
		if len(parentTool) > 0 {
			parentInfo = &Info{
				Name:        filepath.Base(parentPath),
				Description: parentDesc,
				Path:        parentPath,
				Script:      parentScript,
				BeforeExec:  extractScript("beforeExec", parentTool),
				AfterExec:   extractScript("afterExec", parentTool),
			}
		}
	}
	
	// Build full output
	var output string
	
	// 1. Execute parent's BeforeExec if it exists
	if parentInfo != nil && parentInfo.BeforeExec != "" {
		parentBeforeOutput, err := executeScript(parentInfo.BeforeExec, paramValues)
		output += parentBeforeOutput
		if err != nil {
			return output, fmt.Errorf("parent BeforeExec failed: %w", err)
		}
	}
	
	// 2. Execute tool's BeforeExec if it exists
	if toolInfo.BeforeExec != "" {
		beforeOutput, err := executeScript(toolInfo.BeforeExec, paramValues)
		output += beforeOutput
		if err != nil {
			return output, fmt.Errorf("BeforeExec failed: %w", err)
		}
	}
	
	// 3. Execute the main script
	mainOutput, err := executeScript(toolInfo.Script, paramValues)
	output += mainOutput
	if err != nil {
		return output, err
	}
	
	// 4. Execute tool's AfterExec if it exists
	if toolInfo.AfterExec != "" {
		afterOutput, err := executeScript(toolInfo.AfterExec, paramValues)
		output += afterOutput
		if err != nil {
			return output, fmt.Errorf("AfterExec failed: %w", err)
		}
	}
	
	// 5. Execute parent's AfterExec if it exists
	if parentInfo != nil && parentInfo.AfterExec != "" {
		parentAfterOutput, err := executeScript(parentInfo.AfterExec, paramValues)
		output += parentAfterOutput
		if err != nil {
			return output, fmt.Errorf("parent AfterExec failed: %w", err)
		}
	}
	
	return output, nil
}

// Helper function to extract BeforeExec or AfterExec from tool configuration
func extractScript(scriptType string, tool []string) string {
	for _, line := range tool {
		if strings.HasPrefix(line, scriptType+": ") {
			return strings.TrimPrefix(line, scriptType+": ")
		}
	}
	return ""
}

// Helper function to execute a script with parameters
func executeScript(script string, paramValues map[string]string) (string, error) {
	if script == "" {
		return "", nil
	}
	
	// Template the script
	tmpl, err := template.New("script").Parse(script)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, paramValues); err != nil {
		return "", err
	}
	
	// Execute the command
	cmd := exec.Command("sh", "-c", buf.String())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return stderr.String(), err
	}
	
	return stdout.String(), nil
}