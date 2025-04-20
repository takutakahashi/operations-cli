package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

func handleToolCallWithDangerCheck(ctx context.Context, request mcp.CallToolRequest, toolMgr *tool.Manager, toolPath string) (*mcp.CallToolResult, error) {
	paramValues := make(map[string]string)
	for name, value := range request.Params.Arguments {
		switch v := value.(type) {
		case string:
			paramValues[name] = v
		case float64:
			paramValues[name] = fmt.Sprintf("%g", v)
		case bool:
			paramValues[name] = fmt.Sprintf("%t", v)
		default:
			paramValues[name] = fmt.Sprintf("%v", v)
		}
	}

	_, _, _, dangerLevel, err := toolMgr.FindTool(toolPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Tool not found: %v", err)), nil
	}

	if dangerLevel == "high" {
		confirm, ok := request.Params.Arguments["confirm"].(bool)
		if !ok || !confirm {
			return mcp.NewToolResultError(
				"This tool has a high danger level and requires explicit confirmation. " +
					"Please confirm by calling this tool with an additional 'confirm: true' parameter."), nil
		}
		delete(paramValues, "confirm")
	}

	var stdout bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = toolMgr.ExecuteTool(toolPath, paramValues)

	w.Close()
	io.Copy(&stdout, r)
	os.Stdout = oldStdout

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error executing tool: %v\nOutput: %s", err, stdout.String())), nil
	}

	return mcp.NewToolResultText(stdout.String()), nil
}
