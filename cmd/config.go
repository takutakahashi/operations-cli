package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/config"
	"gopkg.in/yaml.v3"
)

var (
	configBuildInput      string
	configBuildOutput     string
	configCompileInput    string
	configCompileOutput   string
	configDecompileInput  string
	configDecompileOutput string
	configInitOutput      string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Config related commands",
}

var configBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "設定ファイルをフラットに展開して出力します",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configBuildInput == "" {
			return fmt.Errorf("-f/--file で入力ファイルを指定してください")
		}
		cfg, err := config.LoadConfig(configBuildInput)
		if err != nil {
			return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
		}
		outCfg := &config.Config{
			Actions: cfg.Actions,
			SSH:     cfg.SSH,
			Tools:   cfg.Tools, // 元の階層構造を維持
		}
		out, err := yaml.Marshal(outCfg)
		if err != nil {
			return fmt.Errorf("YAMLへの変換に失敗しました: %w", err)
		}
		if configBuildOutput != "" {
			f, err := os.Create(configBuildOutput)
			if err != nil {
				return fmt.Errorf("出力ファイルの作成に失敗しました: %w", err)
			}
			defer f.Close()
			_, err = f.Write(out)
			if err != nil {
				return fmt.Errorf("出力ファイルへの書き込みに失敗しました: %w", err)
			}
		} else {
			fmt.Print(string(out))
		}
		return nil
	},
}

var configCompileCmd = &cobra.Command{
	Use:   "compile",
	Short: "ディレクトリ構造から設定ファイルを生成します",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configCompileInput == "" {
			return fmt.Errorf("-d/--dir で入力ディレクトリを指定してください")
		}
		builder := config.NewConfigBuilder(configCompileInput)
		var out *os.File
		if configCompileOutput != "" {
			f, err := os.Create(configCompileOutput)
			if err != nil {
				return fmt.Errorf("出力ファイルの作成に失敗しました: %w", err)
			}
			defer f.Close()
			out = f
		} else {
			out = os.Stdout
		}
		if err := builder.Build(out); err != nil {
			return fmt.Errorf("Configのビルドに失敗しました: %w", err)
		}
		return nil
	},
}

var configDecompileCmd = &cobra.Command{
	Use:   "decompile",
	Short: "設定ファイルからディレクトリ構成に展開します",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configDecompileInput == "" {
			return fmt.Errorf("-f/--file で入力ファイルを指定してください")
		}
		if configDecompileOutput == "" {
			return fmt.Errorf("-d/--dir で出力ディレクトリを指定してください")
		}
		cfg, err := config.LoadConfig(configDecompileInput)
		if err != nil {
			return fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
		}
		builder := config.NewConfigBuilder("")
		if err := builder.ExportToDir(cfg, configDecompileOutput); err != nil {
			return fmt.Errorf("ディレクトリ展開に失敗しました: %w", err)
		}
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "設定テンプレートの初期化コマンド",
}

var configInitAllCmd = &cobra.Command{
	Use:   "all",
	Short: "テンプレートディレクトリ構成を指定したディレクトリに展開します",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configInitOutput == "" {
			return fmt.Errorf("-d/--dir で出力ディレクトリを指定してください")
		}
		
		if err := os.MkdirAll(configInitOutput, 0755); err != nil {
			return fmt.Errorf("ディレクトリの作成に失敗しました: %w", err)
		}
		
		rootMeta := map[string]interface{}{
			"actions": []map[string]interface{}{
				{
					"danger_level": "low",
					"type":         "confirm",
					"message":      "This is a low-risk operation. Proceed?",
				},
				{
					"danger_level": "medium",
					"type":         "confirm",
					"message":      "This is a medium-risk operation. Are you sure you want to proceed?",
				},
				{
					"danger_level": "high",
					"type":         "confirm",
					"message":      "This is a high-risk operation. Please confirm carefully before proceeding.",
				},
			},
			"tools": []map[string]interface{}{
				{"path": "tools/example"},
			},
		}
		
		if err := config.WriteMetadata(filepath.Join(configInitOutput, "metadata.yaml"), rootMeta); err != nil {
			return fmt.Errorf("metadata.yamlの作成に失敗しました: %w", err)
		}
		
		toolDir := filepath.Join(configInitOutput, "tools/example")
		if err := os.MkdirAll(toolDir, 0755); err != nil {
			return fmt.Errorf("ツールディレクトリの作成に失敗しました: %w", err)
		}
		
		toolMeta := map[string]interface{}{
			"params": map[string]interface{}{
				"param1": map[string]interface{}{
					"description": "Example parameter",
					"type":        "string",
					"required":    true,
				},
			},
			"script": "main.sh",
		}
		
		if err := config.WriteMetadata(filepath.Join(toolDir, "metadata.yaml"), toolMeta); err != nil {
			return fmt.Errorf("ツールのmetadata.yamlの作成に失敗しました: %w", err)
		}
		
		mainScript := `#!/bin/bash
# Example script
echo "Example tool with parameter: {{.param1}}"
echo "Current date: $(date)"
`
		if err := os.WriteFile(filepath.Join(toolDir, "main.sh"), []byte(mainScript), 0755); err != nil {
			return fmt.Errorf("main.shの作成に失敗しました: %w", err)
		}
		
		fmt.Printf("テンプレートディレクトリ構成を %s に展開しました\n", configInitOutput)
		return nil
	},
}

var configInitToolCmd = &cobra.Command{
	Use:   "tool [tool-name]",
	Short: "新しいツールを指定のディレクトリに展開します",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if configInitOutput == "" {
			return fmt.Errorf("-d/--dir で出力ディレクトリを指定してください")
		}
		
		toolName := args[0]
		toolDir := filepath.Join(configInitOutput, toolName)
		
		if err := os.MkdirAll(toolDir, 0755); err != nil {
			return fmt.Errorf("ツールディレクトリの作成に失敗しました: %w", err)
		}
		
		meta := map[string]interface{}{
			"params": map[string]interface{}{
				"param1": map[string]interface{}{
					"description": "Tool parameter",
					"type":        "string",
					"required":    true,
				},
			},
			"script": "main.sh",
		}
		
		if err := config.WriteMetadata(filepath.Join(toolDir, "metadata.yaml"), meta); err != nil {
			return fmt.Errorf("metadata.yamlの作成に失敗しました: %w", err)
		}
		
		script := `#!/bin/bash
# Tool script
echo "Tool: ${0##*/}"
echo "Parameter: {{.param1}}"
echo "Executed at: $(date)"
`
		if err := os.WriteFile(filepath.Join(toolDir, "main.sh"), []byte(script), 0755); err != nil {
			return fmt.Errorf("main.shの作成に失敗しました: %w", err)
		}
		
		fmt.Printf("ツール %s を %s に作成しました\n", toolName, configInitOutput)
		return nil
	},
}

func init() {
	configBuildCmd.Flags().StringVarP(&configBuildInput, "file", "f", "", "ベースとなる設定ファイルのパス")
	configBuildCmd.Flags().StringVarP(&configBuildOutput, "output", "o", "", "出力先ファイルパス（省略時は標準出力）")
	configCompileCmd.Flags().StringVarP(&configCompileInput, "dir", "d", "", "ベースとなるディレクトリのパス")
	configCompileCmd.Flags().StringVarP(&configCompileOutput, "output", "o", "", "出力先ファイルパス（省略時は標準出力）")
	configDecompileCmd.Flags().StringVarP(&configDecompileInput, "file", "f", "", "設定ファイルのパス")
	configDecompileCmd.Flags().StringVarP(&configDecompileOutput, "dir", "d", "", "出力ディレクトリのパス")
	configInitAllCmd.Flags().StringVarP(&configInitOutput, "dir", "d", "", "出力ディレクトリのパス")
	configInitToolCmd.Flags().StringVarP(&configInitOutput, "dir", "d", "", "出力ディレクトリのパス")
	
	configInitCmd.AddCommand(configInitAllCmd)
	configInitCmd.AddCommand(configInitToolCmd)
	
	configCmd.AddCommand(configBuildCmd)
	configCmd.AddCommand(configCompileCmd)
	configCmd.AddCommand(configDecompileCmd)
	configCmd.AddCommand(configInitCmd)
}

// 追加: rootCmdにconfigCmdを追加する関数
func AddConfigCommand(root *cobra.Command) {
	root.AddCommand(configCmd)
}
