package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

var (
	logFile *os.File
	logger  *log.Logger
)

func init() {
	// ログファイルのパスを設定
	logPath := filepath.Join("./tmp", "operation-mcp.log")
	var err error
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		os.Exit(1)
	}

	// ログの設定
	logger = log.New(logFile, "", log.LstdFlags)
	logger.Printf("MCP Server starting, log file: %s", logPath)
}

var mcpServerCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Start an MCP server for operation-mcp tools",
	Long: `Start a Model Context Protocol (MCP) server that exposes all operation-mcp tools
as MCP Tools for LLM applications. The server communicates over stdin/stdout by default.`,
	Run: func(cmd *cobra.Command, args []string) {
		if toolMgr == nil {
			logger.Println("No tools available. Please provide a valid configuration file.")
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
			logger.Printf("Server error: %v\n", err)
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
	// Replace spaces with underscores in tool name
	toolPath := strings.ReplaceAll(toolInfo.Name, " ", "_")
	if parentPath != "" {
		toolPath = parentPath + "_" + toolPath
	}

	// Add debug logging
	logger.Printf("Registering tool: %s\n", toolPath)
	logger.Printf("Parameters: %v\n", toolInfo.Params)

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
	if toolMgr == nil {
		return mcp.NewToolResultError("Tool manager is not initialized"), nil
	}

	// Replace spaces with underscores in tool name
	toolName := strings.ReplaceAll(request.Params.Name, " ", "_")

	// Add debug logging
	logger.Printf("Executing tool: %s\n", toolName)
	logger.Printf("Parameters: %v\n", request.Params.Arguments)

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
	err := toolMgr.ExecuteTool(toolName, params)
	if err != nil {
		logger.Printf("Error executing tool: %v\n", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText("Tool executed successfully"), nil
}
