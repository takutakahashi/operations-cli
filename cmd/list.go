package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available tools",
	Long:  `List all available tools that can be executed.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available tools:")
		if toolMgr != nil && toolMgr.Tools != nil {
			// 親ツール名とサブツール名を出力
			for _, tool := range toolMgr.Tools {
				if len(tool.SubTools) > 0 {
					for _, subtool := range tool.SubTools {
						// サブツール名を親ツール名を含めた完全名で出力
						fmt.Printf("%s.%s\n", tool.Name, subtool.Name)
					}
				} else {
					fmt.Println(tool.Name)
				}
			}
			// 余分な改行を削除するため、末尾の改行は出力しない
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
