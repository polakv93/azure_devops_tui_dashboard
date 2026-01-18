package components

import (
	"fmt"
	"strings"

	"github.com/polakv93/azure_devops_tui_dashboard/internal/api"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/styles"
)

// ReleaseTableConfig holds configuration for rendering a release table
type ReleaseTableConfig struct {
	Releases    []api.Release
	SelectedRow int
	Width       int
}

// RenderReleaseTable renders a table of releases
func RenderReleaseTable(cfg ReleaseTableConfig) string {
	if len(cfg.Releases) == 0 {
		return styles.HelpStyle.Render("No releases found")
	}

	var b strings.Builder

	// Calculate column widths
	releaseWidth := 25
	definitionWidth := 25
	statusWidth := 12
	environmentsWidth := 40

	// Header
	header := fmt.Sprintf("%-*s %-*s %-*s %-*s",
		releaseWidth, "Release",
		definitionWidth, "Definition",
		statusWidth, "Status",
		environmentsWidth, "Environments")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, release := range cfg.Releases {
		name := truncateString(release.Name, releaseWidth-2)
		definition := truncateString(release.ReleaseDefinition.Name, definitionWidth-2)
		status := string(release.Status)
		environments := truncateString(release.GetEnvironmentSummary(), environmentsWidth-2)

		statusStyled := styles.FormatStatus(status)

		row := fmt.Sprintf("%-*s %-*s %s%-*s %-*s",
			releaseWidth, name,
			definitionWidth, definition,
			statusStyled, statusWidth-len(status), "",
			environmentsWidth, environments)

		if i == cfg.SelectedRow {
			row = styles.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		if i < len(cfg.Releases)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// RenderEnvironmentStatus renders a visual representation of environment statuses
func RenderEnvironmentStatus(environments []api.ReleaseEnvironment) string {
	if len(environments) == 0 {
		return "-"
	}

	var parts []string
	for _, env := range environments {
		icon := getEnvironmentIcon(env.Status)
		style := styles.GetStatusStyle(string(env.Status))
		parts = append(parts, style.Render(env.Name+":"+icon))
	}

	return strings.Join(parts, " → ")
}

func getEnvironmentIcon(status api.EnvironmentStatus) string {
	switch status {
	case api.EnvironmentStatusSucceeded:
		return "✓"
	case api.EnvironmentStatusRejected, api.EnvironmentStatusCanceled:
		return "✗"
	case api.EnvironmentStatusInProgress:
		return "●"
	case api.EnvironmentStatusQueued, api.EnvironmentStatusScheduled:
		return "○"
	case api.EnvironmentStatusPartiallySucceeded:
		return "◐"
	default:
		return "-"
	}
}
