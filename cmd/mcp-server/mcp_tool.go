package main

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/takutakahashi/operation-mcp/pkg/config"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

func createMCPTool(name string, toolInfo tool.Info) *mcp.Tool {
	mcpTool := &mcp.Tool{
		Name:        name,
		Description: toolInfo.Description,
		Parameters:  make(map[string]*mcp.Parameter),
	}

	for paramName, param := range toolInfo.Params {
		paramType := "string"
		switch param.Type {
		case "number", "integer":
			paramType = "number"
		case "boolean":
			paramType = "boolean"
		}

		description := paramName
		if param.Description != "" {
			description = param.Description
		}

		mcpTool.Parameters[paramName] = &mcp.Parameter{
			Type:        paramType,
			Description: description,
		}

		if param.Required {
			getRequiredOption(true)(mcpTool.Parameters[paramName])
		} else {
			getRequiredOption(false)(mcpTool.Parameters[paramName])
		}
	}

	return mcpTool
}

func getRequiredOption(required bool) func(*mcp.Parameter) {
	return func(p *mcp.Parameter) {
		p.Required = required
	}
}
