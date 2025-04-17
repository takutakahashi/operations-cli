package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/config"
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

		// Display tools and subtools in a flat structure
		for _, tool := range tools {
			// Display the root tool
			fmt.Printf("%s\n", tool.Name)
			if verbose && len(tool.Params) > 0 {
				printParameters(tool.Params, 1)
			}
			fmt.Println()

			// Display all subtools recursively with tool_subtool format
			printSubtoolsFlat(tool.Subtools, tool.Name, verbose)
		}
	},
}

// AddListCommand adds the list command to the root command
func AddListCommand(root *cobra.Command) {
	// Add verbose flag
	listCmd.Flags().BoolP("verbose", "v", false, "Show detailed information including parameters")
	root.AddCommand(listCmd)
}

// printSubtoolsFlat prints all subtools in a flat list with tool_subtool format
func printSubtoolsFlat(subtools []tool.Info, parentPath string, verbose bool) {
	for _, subtool := range subtools {
		// Format the full tool path (tool_subtool)
		fullPath := fmt.Sprintf("%s_%s", parentPath, subtool.Name)
		
		// Print the full tool path
		fmt.Printf("%s\n", fullPath)
		
		// Print parameters if verbose
		if verbose && len(subtool.Params) > 0 {
			printParameters(subtool.Params, 1)
		}
		
		fmt.Println() // Empty line for readability
		
		// Recursively print nested subtools
		nextParentPath := fullPath
		printSubtoolsFlat(subtool.Subtools, nextParentPath, verbose)
	}
}

// printParameters prints parameters in a readable format
func printParameters(params map[string]config.Parameter, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)
	fmt.Printf("%sParameters:\n", indent)
	
	// Print each parameter with its description
	for name, param := range params {
		fmt.Printf("%s  %s: %s", indent, name, param.Description)
		if param.Required {
			fmt.Print(" (required)")
		}
		fmt.Println()
	}
}