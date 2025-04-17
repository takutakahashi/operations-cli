package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available tools",
	Long:  `List all available tools and their subtools defined in the configuration file`,
	Run: func(cmd *cobra.Command, args []string) {
		// If toolMgr is nil, we couldn't load a config
		if toolMgr == nil {
			fmt.Println("No tools available. Please provide a valid configuration file.")
			return
		}

		// Get verbose flag
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Get tools list
		tools := toolMgr.ListTools()

		// Display header
		fmt.Println("Available tools:")
		fmt.Println()

		// Display tools
		for _, tool := range tools {
			fmt.Println(tool.Name)

			// Display subtools recursively
			printSubtools(tool.Subtools, 1, tool.Name, verbose)

			fmt.Println() // Empty line between tools
		}
	},
}

// AddListCommand adds the list command to the root command
func AddListCommand(root *cobra.Command) {
	// Add verbose flag
	listCmd.Flags().BoolP("verbose", "v", false, "Show detailed information including parameters")
	root.AddCommand(listCmd)
}

func printSubtools(subtools []tool.Info, level int, parentPath string, verbose bool) {
	for _, subtool := range subtools {
		// Print indentation
		for i := 0; i < level; i++ {
			fmt.Print("  ")
		}

		// Print subtool name
		fmt.Printf("└─ %s\n", subtool.Name)

		// Print parameters if verbose
		if verbose && len(subtool.Params) > 0 {
			for i := 0; i < level+1; i++ {
				fmt.Print("  ")
			}
			fmt.Println("Parameters:")
			for name, param := range subtool.Params {
				for i := 0; i < level+2; i++ {
					fmt.Print("  ")
				}
				fmt.Printf("- %s: %s", name, param.Description)
				if param.Required {
					fmt.Print(" (required)")
				}
				fmt.Println()
			}
		}

		// Recursively print subtools
		printSubtools(subtool.Subtools, level+1, parentPath+"."+subtool.Name, verbose)
	}
}
