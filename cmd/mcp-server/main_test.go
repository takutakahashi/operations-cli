package main

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/takutakahashi/operation-mcp/pkg/config"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

func TestCreateMCPTool(t *testing.T) {
	toolInfo := tool.Info{
		Name: "test-tool",
		Params: map[string]config.Parameter{
			"string-param": {
				Description: "A string parameter",
				Type:        "string",
				Required:    true,
			},
			"number-param": {
				Description: "A number parameter",
				Type:        "number",
				Required:    false,
			},
			"bool-param": {
				Description: "A boolean parameter",
				Type:        "boolean",
				Required:    true,
			},
		},
	}

	mcpTool := createMCPTool("test-tool", toolInfo)

	if mcpTool.Name != "test-tool" {
		t.Errorf("Expected tool name to be 'test-tool', got '%s'", mcpTool.Name)
	}

	params := mcpTool.Parameters
	if len(params) != 3 {
		t.Errorf("Expected 3 parameters, got %d", len(params))
	}

	stringParam, ok := params["string-param"]
	if !ok {
		t.Errorf("Expected parameter 'string-param' to exist")
	} else {
		if stringParam.Type != "string" {
			t.Errorf("Expected parameter type to be 'string', got '%s'", stringParam.Type)
		}
		if !stringParam.Required {
			t.Errorf("Expected parameter to be required")
		}
		if stringParam.Description != "A string parameter" {
			t.Errorf("Expected parameter description to be 'A string parameter', got '%s'", stringParam.Description)
		}
	}

	numberParam, ok := params["number-param"]
	if !ok {
		t.Errorf("Expected parameter 'number-param' to exist")
	} else {
		if numberParam.Type != "number" {
			t.Errorf("Expected parameter type to be 'number', got '%s'", numberParam.Type)
		}
		if numberParam.Required {
			t.Errorf("Expected parameter to be optional")
		}
		if numberParam.Description != "A number parameter" {
			t.Errorf("Expected parameter description to be 'A number parameter', got '%s'", numberParam.Description)
		}
	}

	boolParam, ok := params["bool-param"]
	if !ok {
		t.Errorf("Expected parameter 'bool-param' to exist")
	} else {
		if boolParam.Type != "boolean" {
			t.Errorf("Expected parameter type to be 'boolean', got '%s'", boolParam.Type)
		}
		if !boolParam.Required {
			t.Errorf("Expected parameter to be required")
		}
		if boolParam.Description != "A boolean parameter" {
			t.Errorf("Expected parameter description to be 'A boolean parameter', got '%s'", boolParam.Description)
		}
	}
}

func TestGetRequiredOption(t *testing.T) {
	requiredOption := getRequiredOption(true)
	optionalOption := getRequiredOption(false)

	requiredParam := &mcp.Parameter{}
	optionalParam := &mcp.Parameter{}

	requiredOption(requiredParam)
	optionalOption(optionalParam)

	if !requiredParam.Required {
		t.Errorf("Expected parameter to be required")
	}

	if optionalParam.Required {
		t.Errorf("Expected parameter to be optional")
	}
}
