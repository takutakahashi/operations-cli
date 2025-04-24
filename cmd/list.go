package cmd

import (
	"fmt"

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
			for name := range tools {
				fmt.Println(name)
			}
		}
	},
}
