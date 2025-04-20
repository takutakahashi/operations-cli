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

	if mcpTool.InputSchema.Type != "object" {
		t.Errorf("Expected input schema type to be 'object', got '%s'", mcpTool.InputSchema.Type)
	}

	properties := mcpTool.InputSchema.Properties
	if len(properties) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(properties))
	}

	stringProp, ok := properties["string-param"]
	if !ok {
		t.Errorf("Expected property 'string-param' to exist")
	} else {
		propMap, ok := stringProp.(map[string]interface{})
		if !ok {
			t.Errorf("Expected property to be a map")
		} else {
			if propType, ok := propMap["type"].(string); !ok || propType != "string" {
				t.Errorf("Expected property type to be 'string', got '%v'", propMap["type"])
			}
			if desc, ok := propMap["description"].(string); !ok || desc != "A string parameter" {
				t.Errorf("Expected property description to be 'A string parameter', got '%v'", propMap["description"])
			}
		}
	}

	numberProp, ok := properties["number-param"]
	if !ok {
		t.Errorf("Expected property 'number-param' to exist")
	} else {
		propMap, ok := numberProp.(map[string]interface{})
		if !ok {
			t.Errorf("Expected property to be a map")
		} else {
			if propType, ok := propMap["type"].(string); !ok || propType != "number" {
				t.Errorf("Expected property type to be 'number', got '%v'", propMap["type"])
			}
			if desc, ok := propMap["description"].(string); !ok || desc != "A number parameter" {
				t.Errorf("Expected property description to be 'A number parameter', got '%v'", propMap["description"])
			}
		}
	}

	boolProp, ok := properties["bool-param"]
	if !ok {
		t.Errorf("Expected property 'bool-param' to exist")
	} else {
		propMap, ok := boolProp.(map[string]interface{})
		if !ok {
			t.Errorf("Expected property to be a map")
		} else {
			if propType, ok := propMap["type"].(string); !ok || propType != "boolean" {
				t.Errorf("Expected property type to be 'boolean', got '%v'", propMap["type"])
			}
			if desc, ok := propMap["description"].(string); !ok || desc != "A boolean parameter" {
				t.Errorf("Expected property description to be 'A boolean parameter', got '%v'", propMap["description"])
			}
		}
	}

	required := mcpTool.InputSchema.Required
	if len(required) != 2 {
		t.Errorf("Expected 2 required parameters, got %d", len(required))
	}

	foundStringParam := false
	foundBoolParam := false
	for _, r := range required {
		if r == "string-param" {
			foundStringParam = true
		}
		if r == "bool-param" {
			foundBoolParam = true
		}
	}

	if !foundStringParam {
		t.Errorf("Expected 'string-param' to be required")
	}
	if !foundBoolParam {
		t.Errorf("Expected 'bool-param' to be required")
	}
}

func TestGetRequiredOption(t *testing.T) {
	requiredTool := mcp.NewTool("test-tool")
	mcp.WithString("required-param", mcp.Required())(&requiredTool)
	
	optionalTool := mcp.NewTool("test-tool")
	mcp.WithString("optional-param")(&optionalTool)
	
	if len(requiredTool.InputSchema.Required) != 1 {
		t.Errorf("Expected 1 required parameter, got %d", len(requiredTool.InputSchema.Required))
	} else if requiredTool.InputSchema.Required[0] != "required-param" {
		t.Errorf("Expected required parameter name to be 'required-param', got '%s'", requiredTool.InputSchema.Required[0])
	}
	
	if len(optionalTool.InputSchema.Required) != 0 {
		t.Errorf("Expected 0 required parameters, got %d", len(optionalTool.InputSchema.Required))
	}
}
