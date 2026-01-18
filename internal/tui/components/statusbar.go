package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/styles"
)

// StatusBarConfig holds configuration for rendering the status bar
type StatusBarConfig struct {
	LastRefresh     time.Time
	RefreshInterval time.Duration
	DateFormat      string
	IsLoading       bool
	Spinner         spinner.Model
	ErrorCount      int
}

// RenderStatusBar renders the status bar
func RenderStatusBar(cfg StatusBarConfig) string {
	var parts []string

	// Last refresh time
	if !cfg.LastRefresh.IsZero() {
		parts = append(parts, fmt.Sprintf("Last refresh: %s",
			cfg.LastRefresh.Format(cfg.DateFormat)))
	}

	// Next refresh
	if cfg.RefreshInterval > 0 {
		parts = append(parts, fmt.Sprintf("Auto-refresh: %s",
			cfg.RefreshInterval.String()))
	}

	// Loading indicator
	if cfg.IsLoading {
		parts = append(parts, cfg.Spinner.View()+" Loading...")
	}

	// Error count
	if cfg.ErrorCount > 0 {
		errText := fmt.Sprintf("Errors: %d", cfg.ErrorCount)
		parts = append(parts, styles.ErrorStyle.Render(errText))
	}

	return styles.StatusBarStyle.Render(strings.Join(parts, " | "))
}

// RenderLoadingIndicator renders a loading indicator
func RenderLoadingIndicator(s spinner.Model, message string) string {
	return styles.LoadingStyle.Render(s.View() + " " + message)
}
