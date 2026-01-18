package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	AzureDevOps  AzureDevOpsConfig `yaml:"azure_devops"`
	Projects     []ProjectConfig   `yaml:"projects"`
	Display      DisplayConfig     `yaml:"display"`
	RateLimiting RateLimitConfig   `yaml:"rate_limiting"`
}

// AzureDevOpsConfig holds Azure DevOps connection settings
type AzureDevOpsConfig struct {
	Organization string `yaml:"organization"`
	BaseURL      string `yaml:"base_url"`
	PAT          string `yaml:"pat"`
}

// ProjectConfig holds project-specific settings
type ProjectConfig struct {
	Name               string `yaml:"name"`
	BuildDefinitions   []int  `yaml:"build_definitions"`
	ReleaseDefinitions []int  `yaml:"release_definitions"`
}

// DisplayConfig holds display settings
type DisplayConfig struct {
	RefreshInterval    time.Duration `yaml:"refresh_interval"`
	MaxItemsPerProject int           `yaml:"max_items_per_project"`
	DateFormat         string        `yaml:"date_format"`
}

// RateLimitConfig holds rate limiting settings
type RateLimitConfig struct {
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	BurstSize         int     `yaml:"burst_size"`
}

// Load reads and parses the configuration from the given file path
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the config
	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	// Validate configuration
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// expandEnvVars replaces ${VAR} or $VAR patterns with environment variable values
func expandEnvVars(s string) string {
	// Match ${VAR} pattern
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	result := re.ReplaceAllStringFunc(s, func(match string) string {
		varName := strings.TrimPrefix(strings.TrimSuffix(match, "}"), "${")
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match
	})

	return result
}

// applyDefaults sets default values for unspecified configuration options
func applyDefaults(cfg *Config) {
	if cfg.AzureDevOps.BaseURL == "" {
		cfg.AzureDevOps.BaseURL = "https://dev.azure.com"
	}

	if cfg.Display.RefreshInterval == 0 {
		cfg.Display.RefreshInterval = 30 * time.Second
	}

	if cfg.Display.MaxItemsPerProject == 0 {
		cfg.Display.MaxItemsPerProject = 10
	}

	if cfg.Display.DateFormat == "" {
		cfg.Display.DateFormat = "2006-01-02 15:04"
	}

	if cfg.RateLimiting.RequestsPerSecond == 0 {
		cfg.RateLimiting.RequestsPerSecond = 5
	}

	if cfg.RateLimiting.BurstSize == 0 {
		cfg.RateLimiting.BurstSize = 10
	}
}
