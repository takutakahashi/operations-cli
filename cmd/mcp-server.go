package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
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

		server := mcp.NewServer("operation-mcp", "1.0.0")

		registerTools(server, toolMgr)

		fmt.Println("Starting MCP server over stdin/stdout...")
		if err := server.ServeStdio(context.Background()); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	},
}

func AddMCPServerCommand(root *cobra.Command) {
	root.AddCommand(mcpServerCmd)
}

func registerTools(server *mcp.Server, toolMgr *tool.Manager) {
	tools := toolMgr.ListTools()
	for _, toolInfo := range tools {
		registerTool(server, toolMgr, toolInfo, "")
	}
}

func registerTool(server *mcp.Server, toolMgr *tool.Manager, toolInfo tool.Info, parentPath string) {
	toolPath := toolInfo.Name
	if parentPath != "" {
		toolPath = parentPath + "_" + toolInfo.Name
	}

	mcpTool := createMCPTool(toolPath, toolInfo)

	server.RegisterTool(mcpTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleToolCall(ctx, req, toolMgr, toolPath)
	})

	for _, subtool := range toolInfo.Subtools {
		registerTool(server, toolMgr, subtool, toolPath)
	}
}

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
			if param.Required {
				paramOpt = mcp.WithNumber(paramName, mcp.Description(description), mcp.Required())
			} else {
				paramOpt = mcp.WithNumber(paramName, mcp.Description(description))
			}
		case "boolean":
			if param.Required {
				paramOpt = mcp.WithBoolean(paramName, mcp.Description(description), mcp.Required())
			} else {
				paramOpt = mcp.WithBoolean(paramName, mcp.Description(description))
			}
		default:
			if param.Required {
				paramOpt = mcp.WithString(paramName, mcp.Description(description), mcp.Required())
			} else {
				paramOpt = mcp.WithString(paramName, mcp.Description(description))
			}
		}

		toolOpts = append(toolOpts, paramOpt)
	}

	return mcp.NewTool(name, toolOpts...)
}

func handleToolCall(ctx context.Context, request mcp.CallToolRequest, toolMgr *tool.Manager, toolPath string) (*mcp.CallToolResult, error) {
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
