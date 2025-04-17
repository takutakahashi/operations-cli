package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/takutakahashi/operation-mcp/pkg/config"
	"github.com/takutakahashi/operation-mcp/pkg/executor"
	"github.com/takutakahashi/operation-mcp/pkg/tool"
)

var (
	cfg     *config.Config
	toolMgr *tool.Manager

	// バージョン情報（goreleaser によってビルド時に設定される）
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "operations",
	Short: "Operations CLI tool",
	Long:  "A CLI tool for executing operations defined in a configuration file",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// バージョン表示の場合は設定ファイルの読み込みをスキップ
		if cmd.Flag("version") != nil && cmd.Flag("version").Changed {
			return nil
		}

		// Load config using Viper
		if err := loadConfig(); err != nil {
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

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Initialize Viper
	initViper()

	// Add version flag
	rootCmd.PersistentFlags().BoolP("version", "V", false, "Show version information")

	// Add SSH flags
	rootCmd.PersistentFlags().Bool("remote", false, "Enable remote execution mode via SSH")
	rootCmd.PersistentFlags().String("host", "", "SSH remote host")
	rootCmd.PersistentFlags().String("user", "", "SSH username")
	rootCmd.PersistentFlags().String("key", "", "Path to SSH private key")
	rootCmd.PersistentFlags().String("password", "", "SSH password (not recommended)")
	rootCmd.PersistentFlags().Int("port", 22, "SSH port")
	rootCmd.PersistentFlags().Duration("timeout", 10*time.Second, "SSH connection timeout")
	rootCmd.PersistentFlags().Bool("verify-host", true, "Verify host key")

	// Bind flags to Viper
	bindFlags(rootCmd)
}

func initViper() {
	// Set default config paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.operations")

	// Set environment variable prefix
	viper.SetEnvPrefix("OPERATIONS")
	viper.AutomaticEnv()
}

func loadConfig() error {
	// Load config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// Unmarshal config
	cfg = &config.Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

	return nil
}

func bindFlags(cmd *cobra.Command) {
	// Bind all flags to Viper
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err := viper.BindPFlag(f.Name, f); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flag %s: %v\n", f.Name, err)
		}
	})
}

func createExecutor() (executor.Executor, error) {
	// Get SSH configuration from Viper
	remoteMode := viper.GetBool("remote")
	if !remoteMode {
		return executor.NewLocalExecutor(executor.NewOptions()), nil
	}

	verifyHost := viper.GetBool("verify-host")
	sshConfig := &executor.SSHConfig{
		Host:       viper.GetString("host"),
		Port:       viper.GetInt("port"),
		User:       viper.GetString("user"),
		Password:   viper.GetString("password"),
		KeyPath:    viper.GetString("key"),
		VerifyHost: verifyHost,
		Timeout:    viper.GetDuration("timeout"),
	}

	return executor.NewSSHExecutor(sshConfig, executor.NewOptions())
}
