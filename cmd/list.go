package cmd

import (
	"fmt"
	"sort"
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
		if toolMgr != nil {
			tools := toolMgr.GetCompiledTools()

			// ツール名をソートして表示
			var toolNames []string
			for name := range tools {
				toolNames = append(toolNames, name)
			}
			sort.Strings(toolNames)

			// ツールの階層構造を表示
			currentPrefix := ""
			for _, name := range toolNames {
				parts := strings.Split(name, "_")

				// ルートツールの場合
				if len(parts) == 1 {
					if currentPrefix != "" {
						fmt.Println()
					}
					fmt.Printf("%s:\n", name)
					currentPrefix = name
				} else {
					// サブツールの場合
					if parts[0] != currentPrefix {
						if currentPrefix != "" {
							fmt.Println()
						}
						fmt.Printf("%s:\n", parts[0])
						currentPrefix = parts[0]
					}
					// ツール名を完全な形で表示
					fmt.Printf("  └─ %s\n", name)
				}
			}
		}
	},
}
