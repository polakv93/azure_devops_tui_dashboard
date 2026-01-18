package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/polakv93/azure_devops_tui_dashboard/internal/api"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/styles"
)

// BuildTableConfig holds configuration for rendering a build table
type BuildTableConfig struct {
	Builds      []api.Build
	SelectedRow int
	Width       int
}

// RenderBuildTable renders a table of builds
func RenderBuildTable(cfg BuildTableConfig) string {
	if len(cfg.Builds) == 0 {
		return styles.HelpStyle.Render("No builds found")
	}

	var b strings.Builder

	// Calculate column widths
	pipelineWidth := 30
	branchWidth := 20
	statusWidth := 15
	resultWidth := 12
	durationWidth := 10

	// Header
	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
		pipelineWidth, "Pipeline",
		branchWidth, "Branch",
		statusWidth, "Status",
		resultWidth, "Result",
		durationWidth, "Duration")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, build := range cfg.Builds {
		pipeline := truncateString(build.Definition.Name, pipelineWidth-2)
		branch := truncateString(build.GetBranchName(), branchWidth-2)
		status := string(build.Status)
		result := string(build.Result)
		duration := formatBuildDuration(build.GetDuration())

		statusStyled := styles.FormatStatus(status)
		resultStyled := styles.FormatStatus(result)

		row := fmt.Sprintf("%-*s %-*s %s%-*s %s%-*s %-*s",
			pipelineWidth, pipeline,
			branchWidth, branch,
			statusStyled, statusWidth-len(status), "",
			resultStyled, resultWidth-len(result), "",
			durationWidth, duration)

		if i == cfg.SelectedRow {
			row = styles.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		if i < len(cfg.Builds)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatBuildDuration formats a duration for display
func formatBuildDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
