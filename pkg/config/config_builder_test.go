package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildToolWithParamRefs(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "config-builder-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// テスト用のツールディレクトリを作成
	toolDir := filepath.Join(tmpDir, "test-tool")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("Failed to create tool dir: %v", err)
	}

	// テスト用のサブツールディレクトリを作成
	subtoolDir := filepath.Join(toolDir, "test-subtool")
	if err := os.MkdirAll(subtoolDir, 0755); err != nil {
		t.Fatalf("Failed to create subtool dir: %v", err)
	}

	// テスト用のmetadata.yamlを作成
	toolMeta := `params:
  param1:
    description: Test parameter 1
    type: string
    required: true
  param2:
    description: Test parameter 2
    type: string
    required: false
script: main.sh
tools:
  - path: test-subtool`

	subtoolMeta := `params:
  param3:
    description: Test parameter 3
    type: string
    required: true
param_refs:
  param1:
    required: true
  param2:
    required: false
danger_level: high
script: main.sh`

	// ルートのmetadata.yamlを作成
	rootMeta := `tools:
  - path: test-tool`

	if err := os.WriteFile(filepath.Join(tmpDir, "metadata.yaml"), []byte(rootMeta), 0644); err != nil {
		t.Fatalf("Failed to write root metadata: %v", err)
	}

	if err := os.WriteFile(filepath.Join(toolDir, "metadata.yaml"), []byte(toolMeta), 0644); err != nil {
		t.Fatalf("Failed to write tool metadata: %v", err)
	}

	if err := os.WriteFile(filepath.Join(subtoolDir, "metadata.yaml"), []byte(subtoolMeta), 0644); err != nil {
		t.Fatalf("Failed to write subtool metadata: %v", err)
	}

	// テスト用のスクリプトファイルを作成
	scriptContent := "#!/bin/bash\necho 'test script'"
	if err := os.WriteFile(filepath.Join(toolDir, "main.sh"), []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write tool script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subtoolDir, "main.sh"), []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to write subtool script: %v", err)
	}

	// ConfigBuilderを作成してビルド
	builder := NewConfigBuilder(tmpDir)
	cfg, err := builder.Compile()
	if err != nil {
		t.Fatalf("Failed to compile config: %v", err)
	}

	// ツールの検証
	if len(cfg.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(cfg.Tools))
	}

	tool := cfg.Tools[0]
	if tool.Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tool.Name)
	}

	// ツールのパラメータの検証
	if len(tool.Params) != 2 {
		t.Fatalf("Expected 2 parameters in tool, got %d", len(tool.Params))
	}

	param1, ok := tool.Params["param1"]
	if !ok {
		t.Fatal("Expected param1 to exist in tool params")
	}
	if param1.Description != "Test parameter 1" {
		t.Errorf("Expected param1 description 'Test parameter 1', got '%s'", param1.Description)
	}
	if !param1.Required {
		t.Error("Expected param1 to be required")
	}

	// サブツールの検証
	if len(tool.Subtools) != 1 {
		t.Fatalf("Expected 1 subtool, got %d", len(tool.Subtools))
	}

	subtool := tool.Subtools[0]
	if subtool.Name != "test-subtool" {
		t.Errorf("Expected subtool name 'test-subtool', got '%s'", subtool.Name)
	}

	// サブツールのパラメータの検証
	if len(subtool.Params) != 1 {
		t.Fatalf("Expected 1 parameter in subtool, got %d", len(subtool.Params))
	}

	param3, ok := subtool.Params["param3"]
	if !ok {
		t.Fatal("Expected param3 to exist in subtool params")
	}
	if param3.Description != "Test parameter 3" {
		t.Errorf("Expected param3 description 'Test parameter 3', got '%s'", param3.Description)
	}

	// サブツールのparam_refsの検証
	if len(subtool.ParamRefs) != 2 {
		t.Fatalf("Expected 2 param_refs in subtool, got %d", len(subtool.ParamRefs))
	}

	param1Ref, ok := subtool.ParamRefs["param1"]
	if !ok {
		t.Fatal("Expected param1 to exist in subtool param_refs")
	}
	if !param1Ref.Required {
		t.Error("Expected param1 to be required in param_refs")
	}

	param2Ref, ok := subtool.ParamRefs["param2"]
	if !ok {
		t.Fatal("Expected param2 to exist in subtool param_refs")
	}
	if param2Ref.Required {
		t.Error("Expected param2 to be not required in param_refs")
	}

	// サブツールのdanger_levelの検証
	if subtool.DangerLevel != "high" {
		t.Errorf("Expected danger_level 'high', got '%s'", subtool.DangerLevel)
	}
}

func TestBuildToolWithBeforeAfterExec(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "config-builder-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// テスト用のツールディレクトリを作成
	toolDir := filepath.Join(tmpDir, "test-tool")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("Failed to create tool dir: %v", err)
	}

	// beforeExec/afterExecディレクトリを作成
	beforeExecDir := filepath.Join(toolDir, "beforeExec")
	afterExecDir := filepath.Join(toolDir, "afterExec")
	if err := os.MkdirAll(beforeExecDir, 0755); err != nil {
		t.Fatalf("Failed to create beforeExec dir: %v", err)
	}
	if err := os.MkdirAll(afterExecDir, 0755); err != nil {
		t.Fatalf("Failed to create afterExec dir: %v", err)
	}

	// テスト用のmetadata.yamlを作成
	rootMeta := `tools:
  - path: test-tool`

	toolMeta := `params:
  param1:
    description: Test parameter
    type: string
    required: true
script: main.sh
beforeExec:
  - path: beforeExec/00-echo.sh
afterExec:
  - path: afterExec/00-echo.sh`

	// テスト用のスクリプトファイルを作成
	beforeExecContent := "#!/bin/bash\necho 'before exec'"
	afterExecContent := "#!/bin/bash\necho 'after exec'"
	mainScriptContent := "#!/bin/bash\necho 'main script'"

	if err := os.WriteFile(filepath.Join(tmpDir, "metadata.yaml"), []byte(rootMeta), 0644); err != nil {
		t.Fatalf("Failed to write root metadata: %v", err)
	}

	if err := os.WriteFile(filepath.Join(toolDir, "metadata.yaml"), []byte(toolMeta), 0644); err != nil {
		t.Fatalf("Failed to write tool metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(beforeExecDir, "00-echo.sh"), []byte(beforeExecContent), 0755); err != nil {
		t.Fatalf("Failed to write beforeExec script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(afterExecDir, "00-echo.sh"), []byte(afterExecContent), 0755); err != nil {
		t.Fatalf("Failed to write afterExec script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(toolDir, "main.sh"), []byte(mainScriptContent), 0755); err != nil {
		t.Fatalf("Failed to write main script: %v", err)
	}

	// ConfigBuilderを作成してビルド
	builder := NewConfigBuilder(tmpDir)
	cfg, err := builder.Compile()
	if err != nil {
		t.Fatalf("Failed to compile config: %v", err)
	}

	// ツールの検証
	if len(cfg.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(cfg.Tools))
	}

	tool := cfg.Tools[0]
	if tool.Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tool.Name)
	}

	// beforeExec/afterExecの検証
	if len(tool.BeforeExec) != 1 {
		t.Fatalf("Expected 1 beforeExec script, got %d", len(tool.BeforeExec))
	}
	if tool.BeforeExec[0] != beforeExecContent {
		t.Errorf("Expected beforeExec content '%s', got '%s'", beforeExecContent, tool.BeforeExec[0])
	}

	if len(tool.AfterExec) != 1 {
		t.Fatalf("Expected 1 afterExec script, got %d", len(tool.AfterExec))
	}
	if tool.AfterExec[0] != afterExecContent {
		t.Errorf("Expected afterExec content '%s', got '%s'", afterExecContent, tool.AfterExec[0])
	}

	// メインスクリプトの検証
	if tool.Script != mainScriptContent {
		t.Errorf("Expected main script content '%s', got '%s'", mainScriptContent, tool.Script)
	}
}
