package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/config"
	"gopkg.in/yaml.v3"
)

var (
	configBuildInput  string
	configBuildOutput string
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
		out, err := yaml.Marshal(cfg)
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

func init() {
	configBuildCmd.Flags().StringVarP(&configBuildInput, "file", "f", "", "ベースとなる設定ファイルのパス")
	configBuildCmd.Flags().StringVarP(&configBuildOutput, "output", "o", "", "出力先ファイルパス（省略時は標準出力）")
	configCmd.AddCommand(configBuildCmd)
}

// 追加: rootCmdにconfigCmdを追加する関数
func AddConfigCommand(root *cobra.Command) {
	root.AddCommand(configCmd)
}
