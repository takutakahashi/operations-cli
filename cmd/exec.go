package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

		toolName := args[0]
		var subtoolName string
		if len(args) > 1 {
			subtoolName = args[1]
		}

		// パラメータの取得
		params := make(map[string]string)
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Changed {
				params[f.Name] = f.Value.String()
			}
		})

		// ツールの実行
		toolPath := toolName
		if subtoolName != "" {
			toolPath = fmt.Sprintf("%s_%s", toolName, subtoolName)
		}
		return toolMgr.ExecuteTool(toolPath, params)
	},
}

// AddExecCommand adds the exec command to the root command
func AddExecCommand(root *cobra.Command) {
	root.AddCommand(execCmd)
}
