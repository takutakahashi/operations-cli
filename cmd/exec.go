package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/takutakahashi/operation-mcp/pkg/logger"
)

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec [tool] [subtool]",
	Short: "Execute a tool or subtool",
	Long: `Execute a tool or subtool with the specified parameters.
If no subtool is specified, the main tool will be executed.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if toolMgr == nil {
			return fmt.Errorf("tool manager not initialized")
		}

		// execコマンドでは標準出力ロガーを使用
		toolMgr.WithLogger(logger.NewStdoutLogger())

		toolName := args[0]
		var subtoolName string
		if len(args) > 1 {
			subtoolName = args[1]
		}

		// パラメータの取得
		params := make(map[string]string)
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Changed && f.Name != "set" {
				params[f.Name] = f.Value.String()
			}
		})

		// --set フラグで指定されたパラメータを追加
		setParams, _ := cmd.Flags().GetStringArray("set")
		for _, param := range setParams {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) == 2 {
				params[parts[0]] = parts[1]
			}
		}

		// ツールの実行
		toolPath := toolName
		if subtoolName != "" {
			toolPath = fmt.Sprintf("%s_%s", toolName, subtoolName)
		}
		output, err := toolMgr.ExecuteTool(toolPath, params)
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	},
}

func init() {
	execCmd.Flags().StringArrayP("set", "s", []string{}, "Set a parameter (format: name=value)")
}

// AddExecCommand adds the exec command to the root command
func AddExecCommand(root *cobra.Command) {
	root.AddCommand(execCmd)
}
