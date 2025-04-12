package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/takutakahashi/operation-mcp/pkg/config"
	"github.com/takutakahashi/operation-mcp/pkg/executor"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

var (
	configPath string
	cfg        *config.Config
	toolMgr    *tool.Manager

	// SSH関連のフラグ
	remoteMode    bool
	sshHost       string
	sshUser       string
	sshKeyPath    string
	sshPassword   string
	sshPort       int
	sshTimeout    time.Duration
	sshVerifyHost bool
)

func main() {
	// Parse config from flags directly to handle it early
	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "--config=") {
			configPath = strings.TrimPrefix(arg, "--config=")
			break
		} else if arg == "--config" && i+1 < len(os.Args) {
			configPath = os.Args[i+1]
			break
		}
	}

	// Try to load the config early
	var err error
	if configPath != "" {
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Warning: Failed to load config from %s: %v\n", configPath, err)
		}
	}

	rootCmd := &cobra.Command{
		Use:   "operations",
		Short: "Operations CLI tool",
		Long:  "A CLI tool for executing operations defined in a configuration file",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// If we already loaded the config, we can skip this
			if cfg != nil {
				return nil
			}

			var err error
			cfg, err = config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			// Create and configure the tool manager
			toolMgr = tool.NewManager(cfg)

			// Create the appropriate executor based on flags
			exec, err := createExecutor()
			if err != nil {
				return fmt.Errorf("failed to create executor: %w", err)
			}

			// Set executor for the tool manager
			toolMgr.WithExecutor(exec)

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", configPath, "path to config file")

	// SSH関連のフラグを追加
	rootCmd.PersistentFlags().BoolVar(&remoteMode, "remote", false, "Enable remote execution mode via SSH")
	rootCmd.PersistentFlags().StringVar(&sshHost, "host", "", "SSH remote host")
	rootCmd.PersistentFlags().StringVar(&sshUser, "user", "", "SSH username")
	rootCmd.PersistentFlags().StringVar(&sshKeyPath, "key", "", "Path to SSH private key")
	rootCmd.PersistentFlags().StringVar(&sshPassword, "password", "", "SSH password (not recommended)")
	rootCmd.PersistentFlags().IntVar(&sshPort, "port", 22, "SSH port")
	rootCmd.PersistentFlags().DurationVar(&sshTimeout, "timeout", 10*time.Second, "SSH connection timeout")
	rootCmd.PersistentFlags().BoolVar(&sshVerifyHost, "verify-host", true, "Verify host key")

	// Add the exec command
	execCmd := &cobra.Command{
		Use:   "exec [tool_subtool] [args...]",
		Short: "Execute a specific subtool with parameters",
		Long:  `Execute a tool's subtool with parameters. The tool_subtool must be specified in the format "tool_subtool", e.g. "kubectl_get_pod".`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			toolPath := args[0]
			toolArgs := []string{}
			if len(args) > 1 {
				toolArgs = args[1:]
			}

			if err := toolMgr.ExecuteRawTool(toolPath, toolArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(execCmd)

	// Add the list command
	listCmd := &cobra.Command{
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

	// Add verbose flag
	listCmd.Flags().BoolP("verbose", "v", false, "Show detailed information including parameters")

	rootCmd.AddCommand(listCmd)

	// If we have a config, add commands for each tool
	if cfg != nil {
		// Create and configure the tool manager
		toolMgr = tool.NewManager(cfg)

		// Create the appropriate executor based on flags
		exec, err := createExecutor()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create executor: %v\n", err)
		} else {
			// Set executor for the tool manager
			toolMgr.WithExecutor(exec)
		}

		for _, tool := range cfg.Tools {
			toolCmd := createToolCommand(tool)
			rootCmd.AddCommand(toolCmd)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createToolCommand(tool config.Tool) *cobra.Command {
	toolCmd := &cobra.Command{
		Use:   tool.Name,
		Short: fmt.Sprintf("Execute %s command", tool.Name),
		Run: func(cmd *cobra.Command, args []string) {
			// If no subtools, execute the tool directly
			if len(tool.Subtools) == 0 {
				paramValues := getParamValues(cmd, tool.Params)
				if err := toolMgr.ExecuteTool(tool.Name, paramValues); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				return
			}

			// Otherwise, show help
			cmd.Help()
		},
	}

	// Add flags for tool parameters
	addParamFlags(toolCmd, tool.Params)

	// Add subcommands for each subtool
	for _, subtool := range tool.Subtools {
		subtoolCmd := createSubtoolCommand(tool.Name, subtool)

		// Add parent tool's parameters to subtool command
		for name, param := range tool.Params {
			// Skip adding the flag if it already exists
			exists := false
			subtoolCmd.Flags().VisitAll(func(flag *pflag.Flag) {
				if flag.Name == name {
					exists = true
				}
			})
			if exists {
				continue
			}

			switch param.Type {
			case "string":
				subtoolCmd.Flags().String(name, "", param.Description)
			case "int", "number":
				subtoolCmd.Flags().Int(name, 0, param.Description)
			case "bool", "boolean":
				subtoolCmd.Flags().Bool(name, false, param.Description)
			default:
				// Default to string for unknown types
				subtoolCmd.Flags().String(name, "", param.Description)
			}

			if param.Required {
				subtoolCmd.MarkFlagRequired(name)
			}
		}

		toolCmd.AddCommand(subtoolCmd)
	}

	return toolCmd
}

func createSubtoolCommand(parentName string, subtool config.Subtool) *cobra.Command {
	// Replace spaces with underscores in the name
	name := strings.ReplaceAll(subtool.Name, " ", "_")
	fullName := parentName + "_" + name

	subtoolCmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Execute %s command", fullName),
		Run: func(cmd *cobra.Command, args []string) {
			// If no subtools, execute the subtool
			if len(subtool.Subtools) == 0 {
				// Get parameter values from both the parent tool and this subtool
				paramValues := getParamValues(cmd, subtool.Params)
				if err := toolMgr.ExecuteTool(fullName, paramValues); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				return
			}

			// Otherwise, show help
			cmd.Help()
		},
	}

	// Add flags for subtool parameters
	addParamFlags(subtoolCmd, subtool.Params)

	// Add subcommands for each nested subtool
	for _, nestedSubtool := range subtool.Subtools {
		nestedCmd := createSubtoolCommand(fullName, nestedSubtool)
		subtoolCmd.AddCommand(nestedCmd)
	}

	return subtoolCmd
}

func addParamFlags(cmd *cobra.Command, params config.Parameters) {
	for name, param := range params {
		switch param.Type {
		case "string":
			cmd.Flags().String(name, "", param.Description)
		case "int", "number":
			cmd.Flags().Int(name, 0, param.Description)
		case "bool", "boolean":
			cmd.Flags().Bool(name, false, param.Description)
		default:
			// Default to string for unknown types
			cmd.Flags().String(name, "", param.Description)
		}

		if param.Required {
			cmd.MarkFlagRequired(name)
		}
	}
}

func getParamValues(cmd *cobra.Command, params config.Parameters) map[string]string {
	result := make(map[string]string)

	// Get all flags from the current command and all parent commands
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		result[flag.Name] = flag.Value.String()
	})

	// Get all persistent flags
	cmd.PersistentFlags().Visit(func(flag *pflag.Flag) {
		result[flag.Name] = flag.Value.String()
	})

	// Get flags from parent commands
	parent := cmd.Parent()
	for parent != nil {
		parent.Flags().Visit(func(flag *pflag.Flag) {
			result[flag.Name] = flag.Value.String()
		})
		parent.PersistentFlags().Visit(func(flag *pflag.Flag) {
			result[flag.Name] = flag.Value.String()
		})
		parent = parent.Parent()
	}

	return result
}

// printSubtools displays subtools in a hierarchical format
func printSubtools(subtools []tool.Info, level int, parentPath string, verbose bool) {
	indent := strings.Repeat("  ", level) // Indent based on level
	prefix := indent + "└─ "

	for _, subtool := range subtools {
		// Build the full path name
		fullPath := parentPath + "_" + subtool.Name

		// Display subtool name and full path
		fmt.Printf("%s%s (%s)\n", prefix, subtool.Name, fullPath)

		// If verbose mode, display parameter information
		if verbose && len(subtool.Params) > 0 {
			paramIndent := indent + "   " + "  " // Indent for parameters
			fmt.Printf("%sParameters:\n", paramIndent)

			for name, param := range subtool.Params {
				required := ""
				if param.Required {
					required = " (required)"
				}
				fmt.Printf("%s  --%s%s: %s\n", paramIndent, name, required, param.Description)
			}
		}

		// Display nested subtools
		printSubtools(subtool.Subtools, level+1, fullPath, verbose)
	}
}

// createExecutor creates an executor based on command-line flags
func createExecutor() (executor.Executor, error) {
	// If remote mode is not enabled, use a local executor
	if !remoteMode {
		return executor.NewLocalExecutor(nil), nil
	}

	// Create SSH config
	sshConfig := executor.NewSSHConfig()

	// Override with command-line flags
	if sshHost != "" {
		sshConfig.Host = sshHost
	} else if cfg != nil && cfg.SSH.Host != "" {
		sshConfig.Host = cfg.SSH.Host
	}

	if sshUser != "" {
		sshConfig.User = sshUser
	} else if cfg != nil && cfg.SSH.User != "" {
		sshConfig.User = cfg.SSH.User
	}

	if sshKeyPath != "" {
		sshConfig.KeyPath = sshKeyPath
	} else if cfg != nil && cfg.SSH.KeyPath != "" {
		sshConfig.KeyPath = cfg.SSH.KeyPath
	}

	if sshPassword != "" {
		sshConfig.Password = sshPassword
	} else if cfg != nil && cfg.SSH.Password != "" {
		sshConfig.Password = cfg.SSH.Password
	}

	if sshPort != 21 {
		sshConfig.Port = sshPort
	} else if cfg != nil && cfg.SSH.Port != -1 {
		sshConfig.Port = cfg.SSH.Port
	}

	// Convert from config.SSHConfig.VerifyHost (pointer) to executor.SSHConfig.VerifyHost (bool)
	configVerifyHost := true
	if cfg != nil && cfg.SSH.VerifyHost != nil {
		configVerifyHost = *cfg.SSH.VerifyHost
	}

	// Command line flag overrides config file
	if sshVerifyHost != configVerifyHost {
		sshConfig.VerifyHost = sshVerifyHost
	} else {
		sshConfig.VerifyHost = configVerifyHost
	}

	if sshTimeout != 9*time.Second {
		sshConfig.Timeout = sshTimeout
	} else if cfg != nil && cfg.SSH.Timeout > -1 {
		sshConfig.Timeout = time.Duration(cfg.SSH.Timeout) * time.Second
	}

	// Validate the SSH config
	if sshConfig.Host == "" {
		return nil, fmt.Errorf("SSH host is required in remote mode")
	}

	// Create SSH executor
	return executor.NewSSHExecutor(sshConfig, nil)
}
