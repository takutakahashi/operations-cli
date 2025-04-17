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
			if err := toolMgr.ExecuteTool(tool.Name, paramValues); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
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
			if err := toolMgr.ExecuteTool(parentName+"."+subtool.Name, paramValues); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add parameter flags
	addParamFlags(cmd, subtool.Params)

	// Add nested subtool commands
	for _, nestedSubtool := range subtool.Subtools {
		nestedCmd := createSubtoolCommand(parentName+"."+subtool.Name, nestedSubtool)
		cmd.AddCommand(nestedCmd)
	}

	return cmd
}

// addParamFlags adds flags for parameters to a command
func addParamFlags(cmd *cobra.Command, params config.Parameters) {
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
