package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/logger"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

var mcpServerCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Start an MCP server for operation-mcp tools",
	Long: `Start a Model Context Protocol (MCP) server that exposes all operation-mcp tools
as MCP Tools for LLM applications. The server communicates over stdin/stdout by default.`,
	Run: func(cmd *cobra.Command, args []string) {
		if toolMgr == nil {
			fmt.Println("Error: No tools available. Please provide a valid configuration file.")
			return
		}

		// ロガーの設定
		logDir := os.Getenv("OM_LOG_DIR")
		var l logger.Logger
		if logDir != "" {
			fileLogger, err := logger.NewFileLogger(logDir)
			if err != nil {
				fmt.Printf("Warning: Failed to create file logger: %v\n", err)
				l = logger.NewStdoutLogger()
			} else {
				l = fileLogger
				defer fileLogger.Close()
			}
		} else {
			l = logger.NewStdoutLogger()
		}
		toolMgr.WithLogger(l)

		l.Println("Starting MCP server...")

		// Create MCP server
		s := server.NewMCPServer(
			"Operation MCP",
			"1.0.0",
		)

		// Register all tools from compiledTools
		l.Println("Registering tools...")
		for toolPath, compiledTool := range toolMgr.GetCompiledTools() {
			l.Printf("Registering tool: %s", toolPath)
			toolInfo := tool.Info{
				Name:        toolPath,
				Description: "",
				Params:      compiledTool.Params,
			}
			registerTool(s, toolInfo, "", l)
		}

		l.Println("Starting stdio server...")
		// Start the stdio server
		if err := server.ServeStdio(s); err != nil {
			l.Printf("Fatal error: Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

// AddMCPServerCommand adds the MCP server command to the root command
func AddMCPServerCommand(root *cobra.Command) {
	root.AddCommand(mcpServerCmd)
}

func registerTool(s *server.MCPServer, toolInfo tool.Info, parentPath string, l logger.Logger) {
	toolPath := toolInfo.Name
	if parentPath != "" {
		toolPath = parentPath + "_" + toolInfo.Name
	}

	l.Printf("Registering tool handler for: %s", toolPath)

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
	s.AddTool(t, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		l.Printf("Received tool call request for: %s", toolPath)
		l.Printf("Request parameters: %v", request.Params.Arguments)
		return executeHandler(ctx, request, l)
	})

	// Register subtools
	for _, subtool := range toolInfo.Subtools {
		registerTool(s, subtool, toolPath, l)
	}
}

func executeHandler(ctx context.Context, request mcp.CallToolRequest, l logger.Logger) (*mcp.CallToolResult, error) {
	if toolMgr == nil {
		l.Println("Error: Tool manager is not initialized")
		return mcp.NewToolResultError("Tool manager is not initialized"), nil
	}

	// Replace spaces with underscores in tool name
	toolName := strings.ReplaceAll(request.Params.Name, " ", "_")

	// Add debug logging
	l.Printf("Executing tool: %s", toolName)
	l.Printf("Parameters: %v", request.Params.Arguments)

	// Convert parameters to string map
	params := make(map[string]string)
	for key, value := range request.Params.Arguments {
		if str, ok := value.(string); ok {
			params[key] = str
		} else {
			// Convert non-string values to string
			params[key] = fmt.Sprintf("%v", value)
		}
	}

	// Execute the tool
	output, err := toolMgr.ExecuteTool(toolName, params)
	if err != nil {
		l.Printf("Error executing tool: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("%v\n%s", err, output)), nil
	}

	l.Printf("Tool execution completed successfully")
	return mcp.NewToolResultText(output), nil
}
