package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

	// コマンドラインフラグ
	configFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd *cobra.Command

func init() {
	rootCmd = newRootCmd()
	AddExecCommand(rootCmd)
	rootCmd.AddCommand(listCmd)
	AddMCPServerCommand(rootCmd)
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operations",
		Short: "Operations CLI tool",
		Long:  "A CLI tool for executing operations defined in a configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			// バージョン表示の場合
			if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
				fmt.Printf("operations %s (commit: %s, built: %s)\n", version, commit, date)
				return nil
			}
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// バージョン表示の場合は設定ファイルの読み込みをスキップ
			if cmd.Flag("version") != nil && cmd.Flag("version").Changed {
				return nil
			}

			// upgrade コマンドの場合も設定ファイルの読み込みをスキップ
			if cmd.Name() == "upgrade" || (cmd.Parent() != nil && cmd.Parent().Name() == "upgrade") {
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

			// ツールコマンドを追加
			for _, tool := range cfg.Tools {
				toolCmd := createToolCommand(tool)
				cmd.AddCommand(toolCmd)
			}

			return nil
		},
	}

	// Add version flag
	cmd.PersistentFlags().BoolP("version", "V", false, "Show version information")

	// Add config file flag
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to config file")

	// Add set flag
	cmd.PersistentFlags().StringArray("set", []string{}, "Set parameter values in the format key=value")

	// Add SSH flags
	cmd.PersistentFlags().Bool("remote", false, "Enable remote execution via SSH")
	cmd.PersistentFlags().String("host", "", "SSH remote host (required in remote mode)")
	cmd.PersistentFlags().String("user", "", "SSH username (default: current user)")
	cmd.PersistentFlags().String("key", "", "Path to SSH private key (default: ~/.ssh/id_rsa)")
	cmd.PersistentFlags().String("password", "", "SSH password (key authentication is preferred)")
	cmd.PersistentFlags().Int("port", 22, "SSH port")
	cmd.PersistentFlags().Duration("timeout", 10*time.Second, "SSH connection timeout")
	cmd.PersistentFlags().Bool("verify-host", true, "Verify host key")

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// loadConfig loads the configuration file
func loadConfig() error {
	fmt.Fprintf(os.Stderr, "Loading config from: %s\n", configFile)

	// Load config using Viper
	if configFile != "" {
		absPath, err := filepath.Abs(configFile)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", configFile, err)
		}
		fmt.Fprintf(os.Stderr, "Using absolute path: %s\n", absPath)

		// Load config with imports
		loadedCfg, err := config.LoadConfig(absPath)
		if err != nil {
			return fmt.Errorf("error loading config file %s: %w", absPath, err)
		}
		cfg = loadedCfg

		fmt.Fprintf(os.Stderr, "Config loaded successfully. Tools: %d\n", len(cfg.Tools))
	} else {
		// Check for config in home directory
		home, err := os.UserHomeDir()
		if err == nil {
			homeConfig := filepath.Join(home, ".operations", "config.yaml")
			if _, err := os.Stat(homeConfig); err == nil {
				configFile = homeConfig
			}
		}

		// Check for config in current directory
		if configFile == "" {
			if _, err := os.Stat("operations.yaml"); err == nil {
				configFile = "operations.yaml"
			} else if _, err := os.Stat("config.yaml"); err == nil {
				configFile = "config.yaml"
			}
		}

		if configFile == "" {
			return fmt.Errorf("no configuration file found")
		}

		// Load the found config file
		return loadConfig()
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

// fetchConfigFromURL fetches configuration data from a given URL
func fetchConfigFromURL(url string) ([]byte, error) {
	// HTTP クライアントの設定
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// リクエストの作成と送信
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// ステータスコードのチェック
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned status code %d", resp.StatusCode)
	}

	// レスポンスボディの読み込み
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
