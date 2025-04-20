package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

var mcpServerCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Start an MCP server for operation-mcp tools",
	Long: `Start a Model Context Protocol (MCP) server that exposes all operation-mcp tools
as MCP Tools for LLM applications. The server communicates over stdin/stdout by default.`,
	Run: func(cmd *cobra.Command, args []string) {
		if toolMgr == nil {
			fmt.Println("No tools available. Please provide a valid configuration file.")
			return
		}

		// Create MCP server
		s := server.NewMCPServer(
			"Operation MCP",
			"1.0.0",
		)

		// Register all tools
		tools := toolMgr.ListTools()
		for _, toolInfo := range tools {
			registerTool(s, toolInfo, "")
		}

		// Start the stdio server
		if err := server.ServeStdio(s); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

// AddMCPServerCommand adds the MCP server command to the root command
func AddMCPServerCommand(root *cobra.Command) {
	root.AddCommand(mcpServerCmd)
}

// registerTool registers a single tool and its subtools with the MCP server
func registerTool(s *server.MCPServer, toolInfo tool.Info, parentPath string) {
	toolPath := toolInfo.Name
	if parentPath != "" {
		toolPath = parentPath + "_" + toolInfo.Name
	}

	// Create tool options
	opts := []mcp.ToolOption{
		mcp.WithDescription(toolInfo.Description),
	}

	// Add parameters
	for name, param := range toolInfo.Params {
		paramDesc := name
		if param.Description != "" {
			paramDesc = param.Description
		}

		var paramOpt mcp.ToolOption
		switch param.Type {
		case "number", "integer":
			if param.Required {
				paramOpt = mcp.WithNumber(name, mcp.Description(paramDesc), mcp.Required())
			} else {
				paramOpt = mcp.WithNumber(name, mcp.Description(paramDesc))
			}
		case "boolean":
			if param.Required {
				paramOpt = mcp.WithBoolean(name, mcp.Description(paramDesc), mcp.Required())
			} else {
				paramOpt = mcp.WithBoolean(name, mcp.Description(paramDesc))
			}
		default:
			if param.Required {
				paramOpt = mcp.WithString(name, mcp.Description(paramDesc), mcp.Required())
			} else {
				paramOpt = mcp.WithString(name, mcp.Description(paramDesc))
			}
		}

		opts = append(opts, paramOpt)
	}

	// Create and register the tool
	t := mcp.NewTool(toolPath, opts...)
	s.AddTool(t, executeHandler)

	// Register subtools
	for _, subtool := range toolInfo.Subtools {
		registerTool(s, subtool, toolPath)
	}
}

func executeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command, ok := request.Params.Arguments["command"].(string)
	if !ok {
		return mcp.NewToolResultError("command must be a string"), nil
	}

	// Execute command and return result
	// TODO: Implement command execution logic
	return mcp.NewToolResultText(fmt.Sprintf("Executed command: %s", command)), nil
}
