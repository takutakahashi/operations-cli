package config

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// Config represents the main configuration structure
type Config struct {
	Actions []Action   `yaml:"actions"`
	Tools   []Tool     `yaml:"tools"`
	SSH     *SSHConfig `yaml:"ssh,omitempty"`
	Imports []string   `yaml:"imports,omitempty"`
}

// Action represents a danger level action configuration
type Action struct {
	DangerLevel string `yaml:"danger_level"`
	Type        string `yaml:"type"`
	Message     string `yaml:"message"`
	Timeout     int    `yaml:"timeout"`
}

// Tool represents a tool configuration
type Tool struct {
	Name       string     `yaml:"name"`
	Command    []string   `yaml:"command,omitempty"`
	Script     string     `yaml:"script,omitempty"`
	BeforeExec string     `yaml:"before_exec,omitempty"`
	AfterExec  string     `yaml:"after_exec,omitempty"`
	Params     Parameters `yaml:"params"`
	Subtools   []Subtool  `yaml:"subtools"`
}

// Subtool represents a subtool configuration
type Subtool struct {
	Name        string     `yaml:"name"`
	Args        []string   `yaml:"args,omitempty"`
	Script      string     `yaml:"script,omitempty"`
	BeforeExec  string     `yaml:"before_exec,omitempty"`
	AfterExec   string     `yaml:"after_exec,omitempty"`
	Params      Parameters `yaml:"params"`
	DangerLevel string     `yaml:"danger_level"`
	Subtools    []Subtool  `yaml:"subtools"`
	ParamRefs   ParamRefs  `yaml:"param_refs,omitempty"`
}

// Parameter represents a parameter configuration
type Parameter struct {
	Description string       `yaml:"description"`
	Type        string       `yaml:"type"`
	Required    bool         `yaml:"required"`
	Validate    []Validation `yaml:"validate"`
}

// Validation represents validation rules for parameters
type Validation struct {
	DangerLevel string   `yaml:"danger_level"`
	Exclude     []string `yaml:"exclude"`
}

// Parameters is a map of parameter name to Parameter
type Parameters map[string]Parameter

// ParamRef represents a reference to a parameter defined in the root tool
type ParamRef struct {
	Required bool `yaml:"required"`
}

// ParamRefs is a map of parameter name to ParamRef
type ParamRefs map[string]ParamRef

// SSHConfig represents SSH connection configuration
type SSHConfig struct {
	Host        string `yaml:"host,omitempty"`
	Port        int    `yaml:"port,omitempty"`
	User        string `yaml:"user,omitempty"`
	Password    string `yaml:"password,omitempty"`
	KeyPath     string `yaml:"key,omitempty"`
	VerifyHost  *bool  `yaml:"verify_host,omitempty"`
	HostKeyPath string `yaml:"host_key_path,omitempty"`
	Timeout     int    `yaml:"timeout,omitempty"` // in seconds
}

// LoadConfig loads the configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	// If configPath is not provided, look for default locations
	if configPath == "" {
		// Check for config in home directory
		home, err := os.UserHomeDir()
		if err == nil {
			homeConfig := filepath.Join(home, ".operations", "config.yaml")
			if _, err := os.Stat(homeConfig); err == nil {
				configPath = homeConfig
			}
		}

		// Check for config in current directory
		if configPath == "" {
			if _, err := os.Stat("operations.yaml"); err == nil {
				configPath = "operations.yaml"
			} else if _, err := os.Stat("config.yaml"); err == nil {
				configPath = "config.yaml"
			}
		}

		// If still no config found, return error
		if configPath == "" {
			return nil, fmt.Errorf("no configuration file found")
		}
	}

	// 設定ファイルの絶対パスを取得
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", configPath, err)
	}
	configPath = absPath

	fmt.Fprintf(os.Stderr, "Loading config from: %s\n", configPath)

	// Initialize the visited paths map
	visitedPaths := make(map[string]bool)

	// Load the config with import handling
	return loadConfigWithImports(configPath, visitedPaths)
}

// loadConfigWithImports loads a configuration file and processes its imports
func loadConfigWithImports(configPath string, visitedPaths map[string]bool) (*Config, error) {
	// 循環参照のチェック
	if visitedPaths[configPath] {
		return nil, fmt.Errorf("circular import detected: %s", configPath)
	}

	// このパスを訪問済みとしてマーク
	visitedPaths[configPath] = true

	fmt.Fprintf(os.Stderr, "Processing config file: %s\n", configPath)

	var data []byte
	var err error

	// URLスキームに基づいて適切な読み込み処理を実行
	switch {
	case strings.HasPrefix(configPath, "s3://"):
		data, err = loadFromS3(configPath)
	case strings.HasPrefix(configPath, "github_release://"):
		data, err = loadFromGitHubRelease(configPath)
	case strings.HasPrefix(configPath, "http://") || strings.HasPrefix(configPath, "https://"):
		data, err = loadFromHTTP(configPath)
	default:
		// ファイルの存在確認
		if _, statErr := os.Stat(configPath); statErr != nil {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		data, err = os.ReadFile(configPath)
	}

	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %w", configPath, err)
	}

	// 設定をパース
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file %s: %w", configPath, err)
	}

	// インポートの処理
	if len(config.Imports) > 0 {
		baseDir := filepath.Dir(configPath)
		fmt.Fprintf(os.Stderr, "Base directory for imports: %s\n", baseDir)

		for _, importPath := range config.Imports {
			fmt.Fprintf(os.Stderr, "Processing import: %s\n", importPath)
			var resolvedPath string

			switch {
			case strings.HasPrefix(importPath, "s3://"):
				resolvedPath = importPath
			case strings.HasPrefix(importPath, "github_release://"):
				resolvedPath = importPath
			case strings.HasPrefix(importPath, "http://") || strings.HasPrefix(importPath, "https://"):
				resolvedPath = importPath
			default:
				if filepath.IsAbs(importPath) {
					resolvedPath = importPath
				} else {
					resolvedPath = filepath.Join(baseDir, importPath)
				}
			}

			fmt.Fprintf(os.Stderr, "Resolved import path: %s\n", resolvedPath)

			// ファイルの存在確認
			if !strings.HasPrefix(resolvedPath, "s3://") &&
				!strings.HasPrefix(resolvedPath, "github_release://") &&
				!strings.HasPrefix(resolvedPath, "http://") &&
				!strings.HasPrefix(resolvedPath, "https://") {
				if _, statErr := os.Stat(resolvedPath); statErr != nil {
					return nil, fmt.Errorf("imported config file not found: %s", resolvedPath)
				}
			}

			importedConfig, err := loadConfigWithImports(resolvedPath, visitedPaths)
			if err != nil {
				return nil, fmt.Errorf("error loading imported config %s: %w", resolvedPath, err)
			}

			// 設定のマージ
			config = *mergeConfigs(&config, importedConfig)
		}
	}

	// インポートフィールドをクリア
	config.Imports = nil

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate actions
	if len(c.Actions) > 0 {
		for _, action := range c.Actions {
			if action.DangerLevel == "" {
				continue // danger_levelが空の場合はスキップ
			}
			if action.Type == "" {
				return fmt.Errorf("action missing type")
			}
			if action.Type != "confirm" && action.Type != "timeout" && action.Type != "force" {
				return fmt.Errorf("invalid action type: %s", action.Type)
			}
			if action.Type == "timeout" && action.Timeout <= 0 {
				return fmt.Errorf("timeout action requires positive timeout value")
			}
		}
	}

	// Validate tools
	for _, tool := range c.Tools {
		if tool.Name == "" {
			return fmt.Errorf("tool missing name")
		}

		// 少なくとも Command または Script のどちらかが指定されているか、または一つ以上のサブツールを持つ必要がある
		if len(tool.Command) == 0 && tool.Script == "" && len(tool.Subtools) == 0 {
			return fmt.Errorf("tool %s must have command, script, or at least one subtool", tool.Name)
		}

		// Command と Script は排他的
		if len(tool.Command) > 0 && tool.Script != "" {
			return fmt.Errorf("tool %s has both command and script, only one should be specified", tool.Name)
		}

		// Validate tool parameters
		for name, param := range tool.Params {
			if name == "" {
				return fmt.Errorf("tool %s has parameter with empty name", tool.Name)
			}
			if param.Type == "" {
				return fmt.Errorf("parameter %s in tool %s missing type", name, tool.Name)
			}
		}

		// Validate subtools
		for _, subtool := range tool.Subtools {
			if err := validateSubtool(subtool, tool.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateSubtool validates a subtool configuration
func validateSubtool(subtool Subtool, parentName string) error {
	if subtool.Name == "" {
		return fmt.Errorf("subtool of %s missing name", parentName)
	}

	fullName := parentName + "_" + subtool.Name

	// Args と Script は排他的
	if len(subtool.Args) > 0 && subtool.Script != "" {
		return fmt.Errorf("subtool %s has both args and script, only one should be specified", fullName)
	}

	// Validate subtool parameters
	for name, param := range subtool.Params {
		if name == "" {
			return fmt.Errorf("subtool %s has parameter with empty name", fullName)
		}
		if param.Type == "" {
			return fmt.Errorf("parameter %s in subtool %s missing type", name, fullName)
		}
	}

	// Validate nested subtools
	for _, nestedSubtool := range subtool.Subtools {
		if err := validateSubtool(nestedSubtool, fullName); err != nil {
			return err
		}
	}

	return nil
}

// mergeConfigs merges two configurations, with base taking precedence over imported
func mergeConfigs(base *Config, imported *Config) *Config {
	if imported == nil {
		return base
	}
	if base == nil {
		return imported
	}

	// Merge actions
	base.Actions = append(base.Actions, imported.Actions...)

	// Create a map for tool names to check duplicates
	toolMap := make(map[string]bool)
	for _, tool := range base.Tools {
		toolMap[tool.Name] = true
	}

	// Merge tools, avoiding duplicates (base tools take precedence)
	for _, importedTool := range imported.Tools {
		if !toolMap[importedTool.Name] {
			base.Tools = append(base.Tools, importedTool)
			toolMap[importedTool.Name] = true
		}
	}

	// For SSH, keep the base configuration if it exists
	if base.SSH == nil {
		base.SSH = imported.SSH
	}

	return base
}

// loadFromHTTP loads configuration from an HTTP(S) URL
func loadFromHTTP(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status code %d: %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
