package main

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

func createMCPTool(name string, toolInfo tool.Info) mcp.Tool {
	toolOpts := []mcp.ToolOption{
		mcp.WithDescription(toolInfo.Description),
	}

	for paramName, param := range toolInfo.Params {
		description := paramName
		if param.Description != "" {
			description = param.Description
		}

		var paramOpt mcp.ToolOption
		switch param.Type {
		case "number", "integer":
			paramOpt = mcp.WithNumber(paramName, mcp.Description(description))
		case "boolean":
			paramOpt = mcp.WithBoolean(paramName, mcp.Description(description))
		default:
			paramOpt = mcp.WithString(paramName, mcp.Description(description))
		}

		if param.Required {
			switch param.Type {
			case "number", "integer":
				paramOpt = mcp.WithNumber(paramName, mcp.Description(description), mcp.Required())
			case "boolean":
				paramOpt = mcp.WithBoolean(paramName, mcp.Description(description), mcp.Required())
			default:
				paramOpt = mcp.WithString(paramName, mcp.Description(description), mcp.Required())
			}
		}

		toolOpts = append(toolOpts, paramOpt)
	}

	return mcp.NewTool(name, toolOpts...)
}

func getRequiredOption(required bool) mcp.PropertyOption {
	if required {
		return mcp.Required()
	}
	return func(schema map[string]interface{}) {
	}
}
