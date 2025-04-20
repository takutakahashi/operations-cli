# MCP Server Design for operation-mcp

## 概要

このドキュメントでは、[Model Context Protocol (MCP)](https://modelcontextprotocol.io) を使用して operation-mcp のツールを LLM アプリケーションに公開するサーバーの設計について説明します。[mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) ライブラリを使用して、operation-mcp の全てのツールを MCP Tool として実装します。

## 設計目標

1. operation-mcp の全てのツールを MCP Tool として実装する
2. operation-mcp のパラメーターを MCP Tool の入力として使用する
3. operation-mcp のパラメーター説明を MCP Tool のパラメーター説明として使用する
4. 階層的なツール構造を MCP でサポートする

## システムアーキテクチャ

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│                 │      │                 │      │                 │
│   LLM Client    │◄────►│   MCP Server    │◄────►│  operation-mcp  │
│                 │      │                 │      │                 │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

### コンポーネント

1. **MCP Server**: mark3labs/mcp-go を使用して実装されるサーバー
2. **Tool Converter**: operation-mcp のツールを MCP Tool に変換するコンポーネント
3. **Tool Executor**: MCP Tool のリクエストを operation-mcp の実行に変換するコンポーネント

## 詳細設計

### 1. MCP サーバーの初期化

```go
package main

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/takutakahashi/operation-mcp/pkg/config"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

func main() {
	// 設定ファイルを読み込む
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// ツールマネージャーを初期化
	toolMgr := tool.NewManager(cfg)

	// MCP サーバーを作成
	s := server.NewMCPServer(
		"operation-mcp",
		"1.0.0",
		server.WithLogging(),
		server.WithRecovery(),
	)

	// ツールを MCP サーバーに登録
	registerTools(s, toolMgr)

	// サーバーを起動
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

### 2. operation-mcp ツールの MCP Tool への変換

```go
// registerTools は operation-mcp のツールを MCP サーバーに登録します
func registerTools(s *server.MCPServer, toolMgr *tool.Manager) {
	// 全てのツールを取得
	tools := toolMgr.ListTools()

	// 各ツールを MCP Tool として登録
	for _, toolInfo := range tools {
		registerTool(s, toolMgr, toolInfo, "")
	}
}

// registerTool は単一のツールとそのサブツールを再帰的に登録します
func registerTool(s *server.MCPServer, toolMgr *tool.Manager, toolInfo tool.Info, parentPath string) {
	// ツールパスを構築
	toolPath := toolInfo.Name
	if parentPath != "" {
		toolPath = parentPath + "_" + toolInfo.Name
	}

	// MCP Tool を作成
	mcpTool := createMCPTool(toolPath, toolInfo)

	// ツールハンドラーを登録
	s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleToolCall(ctx, request, toolMgr, toolPath)
	})

	// サブツールを再帰的に登録
	for _, subtool := range toolInfo.Subtools {
		registerTool(s, toolMgr, subtool, toolPath)
	}
}
```

### 3. MCP Tool の作成

```go
// createMCPTool は operation-mcp のツール情報から MCP Tool を作成します
func createMCPTool(toolPath string, toolInfo tool.Info) *mcp.Tool {
	// ツールの説明を設定
	description := fmt.Sprintf("Tool: %s", toolPath)
	if toolInfo.Description != "" {
		description = toolInfo.Description
	}

	// MCP Tool のオプションを作成
	options := []mcp.ToolOption{
		mcp.WithDescription(description),
	}

	// パラメーターを MCP Tool の入力として追加
	for name, param := range toolInfo.Params {
		paramDescription := name
		if param.Description != "" {
			paramDescription = param.Description
		}

		// パラメータータイプに基づいて適切な MCP 入力タイプを選択
		switch param.Type {
		case "string":
			options = append(options, mcp.WithString(name,
				mcp.Description(paramDescription),
				getRequiredOption(param.Required),
			))
		case "number", "integer":
			options = append(options, mcp.WithNumber(name,
				mcp.Description(paramDescription),
				getRequiredOption(param.Required),
			))
		case "boolean":
			options = append(options, mcp.WithBoolean(name,
				mcp.Description(paramDescription),
				getRequiredOption(param.Required),
			))
		default:
			// デフォルトは文字列として扱う
			options = append(options, mcp.WithString(name,
				mcp.Description(paramDescription),
				getRequiredOption(param.Required),
			))
		}
	}

	// MCP Tool を作成して返す
	return mcp.NewTool(toolPath, options...)
}

// getRequiredOption はパラメーターが必須かどうかに基づいて適切な MCP オプションを返します
func getRequiredOption(required bool) mcp.ParameterOption {
	if required {
		return mcp.Required()
	}
	return mcp.Optional()
}
```

### 4. ツール実行ハンドラー

```go
// handleToolCall は MCP Tool の呼び出しを処理し、operation-mcp のツールを実行します
func handleToolCall(ctx context.Context, request mcp.CallToolRequest, toolMgr *tool.Manager, toolPath string) (*mcp.CallToolResult, error) {
	// リクエストからパラメーター値を抽出
	paramValues := make(map[string]string)
	for name, value := range request.Params.Arguments {
		// 各値を文字列に変換
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

	// 出力をキャプチャするためのバッファを設定
	var stdout, stderr bytes.Buffer
	oldStdout, oldStderr := os.Stdout, os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// ツールを実行
	err := toolMgr.ExecuteTool(toolPath, paramValues)

	// パイプを閉じて出力を読み取る
	w.Close()
	io.Copy(&stdout, r)
	os.Stdout, os.Stderr = oldStdout, oldStderr

	// エラーがあれば、エラー結果を返す
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error executing tool: %v\nOutput: %s", err, stdout.String())), nil
	}

	// 成功した場合、出力をテキスト結果として返す
	return mcp.NewToolResultText(stdout.String()), nil
}
```

### 5. 危険レベルと検証の処理

operation-mcp には危険レベルと検証ルールがあります。これらを MCP Tool の実行時に処理する必要があります。

```go
// 危険レベルチェックを含むツール実行ハンドラー
func handleToolCallWithDangerCheck(ctx context.Context, request mcp.CallToolRequest, toolMgr *tool.Manager, toolPath string) (*mcp.CallToolResult, error) {
	// リクエストからパラメーター値を抽出
	paramValues := make(map[string]string)
	for name, value := range request.Params.Arguments {
		// 各値を文字列に変換
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

	// ツールを検索して危険レベルを取得
	_, _, params, dangerLevel, err := toolMgr.FindTool(toolPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Tool not found: %v", err)), nil
	}

	// 危険レベルチェックを実行
	// 注意: MCP では確認プロンプトを直接サポートしていないため、
	// 高危険度のツールは実行前に警告を返すか、別の承認メカニズムを実装する必要があります
	if dangerLevel == "high" {
		return mcp.NewToolResultError(fmt.Sprintf(
			"This tool has a high danger level and requires explicit confirmation. " +
			"Please confirm by calling this tool with an additional 'confirm: true' parameter.")), nil
	}

	// 確認パラメーターをチェック（高危険度ツールの場合）
	if dangerLevel == "high" {
		confirm, ok := request.Params.Arguments["confirm"].(bool)
		if !ok || !confirm {
			return mcp.NewToolResultError("This tool requires confirmation. Set 'confirm: true' to proceed."), nil
		}
		// 確認パラメーターを削除（operation-mcp には渡さない）
		delete(paramValues, "confirm")
	}

	// 出力をキャプチャするためのバッファを設定
	var stdout bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// ツールを実行
	err = toolMgr.ExecuteTool(toolPath, paramValues)

	// パイプを閉じて出力を読み取る
	w.Close()
	io.Copy(&stdout, r)
	os.Stdout = oldStdout

	// エラーがあれば、エラー結果を返す
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error executing tool: %v\nOutput: %s", err, stdout.String())), nil
	}

	// 成功した場合、出力をテキスト結果として返す
	return mcp.NewToolResultText(stdout.String()), nil
}
```

## 実装上の考慮事項

### 1. ツール名の変換

operation-mcp では、ツールパスはアンダースコアで区切られています（例：`kubectl_get_pod`）。MCP でも同じ命名規則を使用して一貫性を保ちます。

### 2. パラメーターの型変換

operation-mcp のパラメーターは主に文字列として扱われますが、MCP では型付きパラメーターをサポートしています。パラメーターの型情報を使用して、適切な MCP パラメーター型を選択します。

### 3. 危険レベルの処理

operation-mcp には危険レベルの概念があり、特定のアクションを実行する前に確認が必要な場合があります。MCP にはこの概念が直接組み込まれていないため、以下のアプローチを検討します：

1. 高危険度のツールには追加の確認パラメーターを要求する
2. 危険なツールの実行前に警告メッセージを返す
3. 危険レベルに基づいて特定のツールへのアクセスを制限する

### 4. エラー処理

operation-mcp のエラーを MCP クライアントに適切に伝えるために、エラーメッセージをキャプチャして MCP のエラー結果として返します。

### 5. 非同期実行

長時間実行されるツールの場合、MCP のストリーミング機能を使用して進捗状況を報告することを検討します。

## セキュリティ考慮事項

1. **認証**: MCP サーバーへのアクセスを制限するための認証メカニズムを実装する
2. **承認**: 特定のツールへのアクセスを制限するためのロールベースのアクセス制御を検討する
3. **入力検証**: すべてのパラメーター入力を検証して、コマンドインジェクションなどの攻撃を防止する
4. **監査ログ**: すべてのツール実行を記録して、後で監査できるようにする

## 将来の拡張

1. **リソースサポート**: operation-mcp の設定や出力を MCP リソースとして公開する
2. **プロンプトサポート**: 一般的なタスクのための MCP プロンプトを定義する
3. **Web インターフェース**: MCP サーバーを管理するための Web インターフェースを提供する
4. **複数のトランスポート**: stdio だけでなく、HTTP や WebSocket などの他のトランスポートもサポートする

## 結論

この設計により、operation-mcp のすべてのツールを MCP サーバーを通じて LLM アプリケーションに公開できます。パラメーターと説明は MCP Tool の入力として使用され、ツールの実行は operation-mcp のコア機能を通じて処理されます。
