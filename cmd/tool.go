package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/config"
)

// createToolCommand creates a command for a tool
func createToolCommand(tool config.Tool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   tool.Name,
		Short: fmt.Sprintf("Execute %s operation", tool.Name),
		Run: func(cmd *cobra.Command, args []string) {
			// Execute the tool
			paramValues := getParamValues(cmd, tool.Params)
			output, err := toolMgr.ExecuteTool(tool.Name, paramValues)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(output)
		},
	}

	// Add parameter flags
	addParamFlags(cmd, tool.Params)

	// Add subtool commands
	for _, subtool := range tool.Subtools {
		subtoolCmd := createSubtoolCommand(tool.Name, subtool)
		cmd.AddCommand(subtoolCmd)
	}

	return cmd
}

// createSubtoolCommand creates a command for a subtool
func createSubtoolCommand(parentName string, subtool config.Subtool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   subtool.Name,
		Short: fmt.Sprintf("Execute %s.%s operation", parentName, subtool.Name),
		Run: func(cmd *cobra.Command, args []string) {
			// Execute the subtool
			paramValues := getParamValues(cmd, subtool.Params)
			output, err := toolMgr.ExecuteTool(parentName+"_"+subtool.Name, paramValues)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(output)
		},
	}

	// Add parameter flags
	addParamFlags(cmd, subtool.Params)

	// Add nested subtool commands
	for _, nestedSubtool := range subtool.Subtools {
		nestedCmd := createSubtoolCommand(parentName+"_"+subtool.Name, nestedSubtool)
		cmd.AddCommand(nestedCmd)
	}

	return cmd
}

// addParamFlags adds flags for parameters to a command
func addParamFlags(cmd *cobra.Command, params config.Parameters) {
	// Add flags for direct parameters
	for name, param := range params {
		// Create flag description
		desc := param.Description
		if param.Required {
			desc += " (required)"
		}

		// Add flag based on parameter type
		switch param.Type {
		case "string":
			cmd.Flags().String(name, "", desc)
		case "int":
			cmd.Flags().Int(name, 0, desc)
		case "bool":
			cmd.Flags().Bool(name, false, desc)
		default:
			cmd.Flags().String(name, "", desc)
		}
	}

}

// getParamValues gets parameter values from flags
func getParamValues(cmd *cobra.Command, params config.Parameters) map[string]string {
	values := make(map[string]string)

	for name, param := range params {
		// Get flag value
		flag := cmd.Flag(name)
		if flag == nil {
			continue
		}

		// Get value based on parameter type
		switch param.Type {
		case "string":
			values[name] = flag.Value.String()
		case "int":
			values[name] = flag.Value.String()
		case "bool":
			values[name] = flag.Value.String()
		default:
			values[name] = flag.Value.String()
		}
	}

	return values
}

// getParamFromToolManager retrieves a parameter from the tool manager
func getParamFromToolManager(name string) (*config.Parameter, error) {
	for _, tool := range toolMgr.GetConfig().Tools {
		if param, exists := tool.Params[name]; exists {
			return &param, nil
		}
	}
	return nil, fmt.Errorf("parameter not found: %s", name)
}

// createDescribeCommand creates a command to describe a tool
func createDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe [tool-path]",
		Short: "Show detailed information about a tool",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			toolPath := args[0]
			info, err := toolMgr.DescribeTool(toolPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// ツールの基本情報を表示
			fmt.Printf("Name: %s\n", info.Name)
			fmt.Printf("Description: %s\n", info.Description)

			// パラメータの一覧を表示
			if len(info.Params) > 0 {
				fmt.Println("\nParameters:")
				for name, param := range info.Params {
					required := ""
					if param.Required {
						required = " (required)"
					}
					fmt.Printf("  - %s%s: %s\n", name, required, param.Description)
				}
			}
		},
	}

	return cmd
}
