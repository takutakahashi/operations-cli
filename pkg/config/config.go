package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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
	BeforeExec []string   `yaml:"before_exec,omitempty"`
	AfterExec  []string   `yaml:"after_exec,omitempty"`
	Params     Parameters `yaml:"params"`
	Subtools   []Subtool  `yaml:"subtools"`
}

// Subtool represents a subtool configuration
type Subtool struct {
	Name        string     `yaml:"name"`
	Args        []string   `yaml:"args,omitempty"`
	Script      string     `yaml:"script,omitempty"`
	BeforeExec  []string   `yaml:"before_exec,omitempty"`
	AfterExec   []string   `yaml:"after_exec,omitempty"`
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

	// Initialize the visited paths map
	visitedPaths := make(map[string]bool)

	// Load the config with import handling
	return loadConfigWithImports(configPath, visitedPaths)
}

// loadConfigWithImports loads a configuration file and processes its imports
func loadConfigWithImports(configPath string, visitedPaths map[string]bool) (*Config, error) {
	// Check for circular imports using the original path
	if visitedPaths[configPath] {
		return nil, fmt.Errorf("circular import detected: %s", configPath)
	}

	// Mark this file as visited using the original path
	visitedPaths[configPath] = true

	var data []byte
	var err error

	// Check if the configPath is an S3 URL
	if isS3URL(configPath) {
		// Parse S3 URL to get bucket and key
		bucket, key, err := parseS3URL(configPath)
		if err != nil {
			return nil, fmt.Errorf("invalid S3 URL %s: %w", configPath, err)
		}

		// Get S3 client
		client, err := defaultS3Client()
		if err != nil {
			return nil, fmt.Errorf("failed to create S3 client: %w", err)
		}

		// Read the config file from S3
		data, err = readFromS3(client, bucket, key)
		if err != nil {
			return nil, fmt.Errorf("error reading config file from S3 %s: %w", configPath, err)
		}
	} else if isGitHubReleaseURL(configPath) {
		// Parse GitHub Release URL to get owner, repo, path, and tag
		owner, repo, path, tag, err := parseGitHubReleaseURL(configPath)
		if err != nil {
			return nil, fmt.Errorf("invalid GitHub Release URL %s: %w", configPath, err)
		}

		// Get GitHub client
		client, err := defaultGitHubClient()
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub client: %w", err)
		}

		// Read the config file from GitHub Release
		data, err = readFromGitHubRelease(client, owner, repo, path, tag)
		if err != nil {
			return nil, fmt.Errorf("error reading config file from GitHub Release %s: %w", configPath, err)
		}
	} else {
		// For regular file paths, convert to absolute path first
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", configPath, err)
		}

		// Read the config file from local filesystem
		data, err = os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("error reading config file %s: %w", absPath, err)
		}
		// Use absolute path for local files
		configPath = absPath
	}

	// Parse the YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file %s: %w", configPath, err)
	}

	// Process imports if present
	if len(config.Imports) > 0 {
		for _, importPath := range config.Imports {
			var resolvedImportPath string

			// Handle import path resolution differently based on source type
			if isS3URL(configPath) {
				// For S3 URLs, resolve the import path relative to the S3 base path
				resolvedImportPath, err = resolveS3ImportPath(configPath, importPath)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve S3 import path %s relative to %s: %w",
						importPath, configPath, err)
				}
			} else if isGitHubReleaseURL(configPath) {
				// For GitHub Release URLs, resolve the import path relative to the GitHub Release base path
				resolvedImportPath, err = resolveGitHubReleaseImportPath(configPath, importPath)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve GitHub Release import path %s relative to %s: %w",
						importPath, configPath, err)
				}
			} else {
				// For regular file paths, resolve the import path relative to the base directory
				baseDir := filepath.Dir(configPath)
				if !filepath.IsAbs(importPath) {
					resolvedImportPath = filepath.Join(baseDir, importPath)
				} else {
					resolvedImportPath = importPath
				}
			}

			// Load the imported config
			importedConfig, err := loadConfigWithImports(resolvedImportPath, visitedPaths)
			if err != nil {
				return nil, fmt.Errorf("error loading imported config %s: %w", resolvedImportPath, err)
			}

			// Merge the imported config with the current config
			// Current config takes precedence over imported config
			config = *mergeConfigs(&config, importedConfig)
		}
	}

	// Clear the imports field to avoid processing them again
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
