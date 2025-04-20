package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

// CustomMCPServer は、operation-mcpツールのためのMCPサーバーを表します。
type CustomMCPServer struct {
	Name        string
	Version     string
	Tools       map[string]*CustomTool
	ToolManager *tool.Manager
}

// CustomTool は、MCPサーバーで利用可能なツールを表します。
type CustomTool struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Parameters  map[string]Parameter `json:"parameters,omitempty"`
	Required    []string             `json:"required,omitempty"`
}

// Parameter は、ツールのパラメータを表します。
type Parameter struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

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

		server := NewCustomMCPServer("operation-mcp", "1.0.0", toolMgr)

		server.RegisterTools()

		fmt.Println("Starting MCP server over stdin/stdout...")
		if err := server.ServeStdio(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	},
}

// AddMCPServerCommand は、rootコマンドにMCPサーバーコマンドを追加します。
func AddMCPServerCommand(root *cobra.Command) {
	root.AddCommand(mcpServerCmd)
}

// NewCustomMCPServer は、新しいCustomMCPServerインスタンスを作成します。
func NewCustomMCPServer(name, version string, toolMgr *tool.Manager) *CustomMCPServer {
	return &CustomMCPServer{
		Name:        name,
		Version:     version,
		Tools:       make(map[string]*CustomTool),
		ToolManager: toolMgr,
	}
}

// RegisterTools は、利用可能な全てのツールをMCPサーバーに登録します。
func (s *CustomMCPServer) RegisterTools() {
	tools := s.ToolManager.ListTools()
	for _, toolInfo := range tools {
		s.registerTool(toolInfo, "")
	}
}

// registerTool registers a single tool and its subtools with the MCP server.
func (s *CustomMCPServer) registerTool(toolInfo tool.Info, parentPath string) {
	toolPath := toolInfo.Name
	if parentPath != "" {
		toolPath = parentPath + "_" + toolInfo.Name
	}

	customTool := &CustomTool{
		Name:        toolPath,
		Description: toolInfo.Description,
		Parameters:  make(map[string]Parameter),
		Required:    []string{},
	}

	for name, param := range toolInfo.Params {
		paramDesc := name
		if param.Description != "" {
			paramDesc = param.Description
		}

		paramType := "string"
		switch param.Type {
		case "number", "integer":
			paramType = "number"
		case "boolean":
			paramType = "boolean"
		}

		customTool.Parameters[name] = Parameter{
			Type:        paramType,
			Description: paramDesc,
		}

		if param.Required {
			customTool.Required = append(customTool.Required, name)
		}
	}

	s.Tools[toolPath] = customTool

	for _, subtool := range toolInfo.Subtools {
		s.registerTool(subtool, toolPath)
	}
}

// ServeStdio は、標準入出力を使用してMCPサーバーを起動します。
func (s *CustomMCPServer) ServeStdio() error {
	log.Println("Starting MCP server over stdin/stdout")

	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var request json.RawMessage
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF {
				return nil
			}
			log.Printf("Error reading request: %v", err)
			continue
		}

		response, err := s.HandleRequest(request)
		if err != nil {
			log.Printf("Error handling request: %v", err)
			continue
		}

		var responseObj interface{}
		if err := json.Unmarshal(response, &responseObj); err != nil {
			log.Printf("Error unmarshaling response: %v", err)
			continue
		}

		if err := encoder.Encode(responseObj); err != nil {
			log.Printf("Error writing response: %v", err)
		}
	}
}

// HandleRequest は、クライアントからのリクエストを処理します。
func (s *CustomMCPServer) HandleRequest(request []byte) ([]byte, error) {
	var req struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(request, &req); err != nil {
		return createErrorResponse("invalid_request", "Failed to parse request", err)
	}

	switch req.Method {
	case "tools/list":
		return s.handleListTools()
	case "tools/call":
		return s.handleCallTool(req.Params)
	default:
		return createErrorResponse("method_not_found", fmt.Sprintf("Method not supported: %s", req.Method), nil)
	}
}

func (s *CustomMCPServer) handleListTools() ([]byte, error) {
	tools := make([]*CustomTool, 0, len(s.Tools))
	for _, tool := range s.Tools {
		tools = append(tools, tool)
	}

	response := struct {
		Result struct {
			Tools []*CustomTool `json:"tools"`
		} `json:"result"`
	}{
		Result: struct {
			Tools []*CustomTool `json:"tools"`
		}{
			Tools: tools,
		},
	}

	return json.Marshal(response)
}

func (s *CustomMCPServer) handleCallTool(params json.RawMessage) ([]byte, error) {
	var callParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &callParams); err != nil {
		return createErrorResponse("invalid_params", "Failed to parse tool call parameters", err)
	}

	_, exists := s.Tools[callParams.Name]
	if !exists {
		return createErrorResponse("tool_not_found", fmt.Sprintf("Tool not found: %s", callParams.Name), nil)
	}

	paramValues := make(map[string]string)
	for name, value := range callParams.Arguments {
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

	_, _, _, dangerLevel, err := s.ToolManager.FindTool(callParams.Name)
	if err == nil && dangerLevel == "high" {
		confirm, ok := callParams.Arguments["confirm"].(bool)
		if !ok || !confirm {
			return createToolErrorResponse("This tool has a high danger level and requires explicit confirmation. " +
				"Please confirm by calling this tool with an additional 'confirm: true' parameter.")
		}
		delete(paramValues, "confirm")
	}

	var stdout bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = s.ToolManager.ExecuteTool(callParams.Name, paramValues)

	w.Close()
	if _, err := io.Copy(&stdout, r); err != nil {
		return createToolErrorResponse(fmt.Sprintf("Error copying output: %v", err))
	}
	os.Stdout = oldStdout

	if err != nil {
		return createToolErrorResponse(fmt.Sprintf("Error executing tool: %v\nOutput: %s", err, stdout.String()))
	}

	return createToolSuccessResponse(stdout.String())
}

func createErrorResponse(code, message string, err error) ([]byte, error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}

	response := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: errMsg,
		},
	}

	return json.Marshal(response)
}

func createToolSuccessResponse(text string) ([]byte, error) {
	response := struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}{
		Result: struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		}{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{
					Type: "text",
					Text: text,
				},
			},
			IsError: false,
		},
	}

	return json.Marshal(response)
}

func createToolErrorResponse(text string) ([]byte, error) {
	response := struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}{
		Result: struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		}{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{
					Type: "text",
					Text: text,
				},
			},
			IsError: true,
		},
	}

	return json.Marshal(response)
}
