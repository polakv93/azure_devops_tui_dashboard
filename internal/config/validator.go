package config

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validate checks if the configuration is valid
func Validate(cfg *Config) error {
	var errs []string

	// Validate Azure DevOps settings
	if cfg.AzureDevOps.Organization == "" {
		errs = append(errs, "azure_devops.organization is required")
	}

	if cfg.AzureDevOps.PAT == "" {
		errs = append(errs, "azure_devops.pat is required (set AZURE_DEVOPS_PAT environment variable)")
	}

	if cfg.AzureDevOps.BaseURL != "" {
		if !strings.HasPrefix(cfg.AzureDevOps.BaseURL, "https://") &&
			!strings.HasPrefix(cfg.AzureDevOps.BaseURL, "http://") {
			errs = append(errs, "azure_devops.base_url must start with http:// or https://")
		}
	}

	// Validate projects
	if len(cfg.Projects) == 0 {
		errs = append(errs, "at least one project must be configured")
	}

	for i, p := range cfg.Projects {
		if p.Name == "" {
			errs = append(errs, fmt.Sprintf("projects[%d].name is required", i))
		}
	}

	// Validate display settings
	if cfg.Display.RefreshInterval < 0 {
		errs = append(errs, "display.refresh_interval cannot be negative")
	}

	if cfg.Display.MaxItemsPerProject < 1 {
		errs = append(errs, "display.max_items_per_project must be at least 1")
	}

	// Validate rate limiting settings
	if cfg.RateLimiting.RequestsPerSecond <= 0 {
		errs = append(errs, "rate_limiting.requests_per_second must be positive")
	}

	if cfg.RateLimiting.BurstSize < 1 {
		errs = append(errs, "rate_limiting.burst_size must be at least 1")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}
