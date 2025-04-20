package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	// 環境変数からログディレクトリを取得
	logDir := os.Getenv("OM_LOG_DIR")
	if logDir == "" {
		// 環境変数が設定されていない場合はログを出力しない
		logger = log.New(io.Discard, "", 0)
		return
	}

	// ログディレクトリの作成
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create log directory: %v\n", err)
		return
	}

	// ログファイル名にタイムスタンプを追加
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logDir, fmt.Sprintf("mcp-server_%s.log", timestamp))

	var err error
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Warning: Failed to open log file: %v\n", err)
		// エラー時はstdoutにフォールバック
		logger = log.New(os.Stdout, "", log.LstdFlags)
		return
	}

	// マルチライターを作成して、ファイルとstdoutの両方に出力
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// ログの設定
	logger = log.New(multiWriter, "", log.LstdFlags)
	logger.Printf("MCP Server starting, log file: %s", logPath)
}

var mcpServerCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Start an MCP server for operation-mcp tools",
	Long: `Start a Model Context Protocol (MCP) server that exposes all operation-mcp tools
as MCP Tools for LLM applications. The server communicates over stdin/stdout by default.`,
	Run: func(cmd *cobra.Command, args []string) {
		if toolMgr == nil {
			logger.Println("Error: No tools available. Please provide a valid configuration file.")
			return
		}

		logger.Println("Initializing MCP server...")

		// Create MCP server
		s := server.NewMCPServer(
			"Operation MCP",
			"1.0.0",
		)

		// Register all tools from compiledTools
		logger.Println("Registering tools...")
		for toolPath, compiledTool := range toolMgr.GetCompiledTools() {
			logger.Printf("Registering tool: %s", toolPath)
			toolInfo := tool.Info{
				Name:        toolPath,
				Description: "",
				Params:      compiledTool.Params,
			}
			registerTool(s, toolInfo, "")
		}

		logger.Println("Starting stdio server...")
		// Start the stdio server
		if err := server.ServeStdio(s); err != nil {
			logger.Printf("Fatal error: Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

// AddMCPServerCommand adds the MCP server command to the root command
func AddMCPServerCommand(root *cobra.Command) {
	root.AddCommand(mcpServerCmd)
}

func registerTool(s *server.MCPServer, toolInfo tool.Info, parentPath string) {
	toolPath := toolInfo.Name
	if parentPath != "" {
		toolPath = parentPath + "_" + toolInfo.Name
	}

	logger.Printf("Registering tool handler for: %s", toolPath)

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
		logger.Printf("Received tool call request for: %s", toolPath)
		logger.Printf("Request parameters: %v", request.Params.Arguments)
		return executeHandler(ctx, request)
	})

	// Register subtools
	for _, subtool := range toolInfo.Subtools {
		registerTool(s, subtool, toolPath)
	}
}

func executeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if toolMgr == nil {
		logger.Println("Error: Tool manager is not initialized")
		return mcp.NewToolResultError("Tool manager is not initialized"), nil
	}

	// Replace spaces with underscores in tool name
	toolName := strings.ReplaceAll(request.Params.Name, " ", "_")

	// Add debug logging
	logger.Printf("Executing tool: %s", toolName)
	logger.Printf("Parameters: %v", request.Params.Arguments)

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
	logger.Printf("Starting tool execution with parameters: %v", params)
	output, err := toolMgr.ExecuteTool(toolName, params)
	if err != nil {
		logger.Printf("Error executing tool: %v", err)
		logger.Printf("Tool output: %s", output)
		return mcp.NewToolResultError(fmt.Sprintf("%v\n%s", err, output)), nil
	}

	logger.Printf("Tool execution completed successfully")
	logger.Printf("Tool output: %s", output)
	return mcp.NewToolResultText(output), nil
}
