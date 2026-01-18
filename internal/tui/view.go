package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/polakj/azure_devops_tui_dashboard/internal/styles"
)

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	b.WriteString(styles.TitleStyle.Render("Azure DevOps Dashboard"))
	b.WriteString("\n\n")

	// Tabs (Builds / Releases)
	b.WriteString(m.renderTabs())
	b.WriteString("\n")

	// Project tabs
	b.WriteString(m.renderProjectTabs())
	b.WriteString("\n\n")

	// Content area
	if m.IsLoading() {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading...")
	} else if err := m.CurrentError(); err != nil {
		b.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err)))
	} else {
		switch m.activeTab {
		case TabBuilds:
			b.WriteString(m.renderBuildsTable())
		case TabReleases:
			b.WriteString(m.renderReleasesTable())
		}
	}

	// Status bar
	b.WriteString("\n\n")
	b.WriteString(m.renderStatusBar())

	// Help
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render(m.help.View(m.keys)))

	return b.String()
}

// renderTabs renders the main tabs (Builds/Releases)
func (m Model) renderTabs() string {
	buildsTab := "Builds"
	releasesTab := "Releases"

	if m.activeTab == TabBuilds {
		buildsTab = styles.ActiveTabStyle.Render(buildsTab)
		releasesTab = styles.TabStyle.Render(releasesTab)
	} else {
		buildsTab = styles.TabStyle.Render(buildsTab)
		releasesTab = styles.ActiveTabStyle.Render(releasesTab)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, buildsTab, " ", releasesTab)
}

// renderProjectTabs renders the project selection tabs
func (m Model) renderProjectTabs() string {
	var tabs []string

	for i, project := range m.config.Projects {
		name := project.Name
		if i == m.activeProject {
			tabs = append(tabs, styles.ActiveProjectStyle.Render(name))
		} else {
			tabs = append(tabs, styles.ProjectTabStyle.Render(name))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// renderBuildsTable renders the builds table
func (m Model) renderBuildsTable() string {
	builds := m.CurrentBuilds()
	if len(builds) == 0 {
		return styles.HelpStyle.Render("No builds found")
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%-30s %-20s %-15s %-12s %-10s",
		"Pipeline", "Branch", "Status", "Result", "Duration")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, build := range builds {
		pipeline := truncate(build.Definition.Name, 28)
		branch := truncate(build.GetBranchName(), 18)
		status := string(build.Status)
		result := string(build.Result)
		duration := formatDuration(build.GetDuration())

		statusDisplay := styles.GetStatusStyle(status).Render(status)
		resultDisplay := styles.GetStatusStyle(result).Render(result)

		row := fmt.Sprintf("%-30s %-20s %-15s %-12s %-10s",
			pipeline, branch, statusDisplay, resultDisplay, duration)

		if i == m.selectedRow {
			row = styles.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	return b.String()
}

// renderReleasesTable renders the releases table
func (m Model) renderReleasesTable() string {
	releases := m.CurrentReleases()
	if len(releases) == 0 {
		return styles.HelpStyle.Render("No releases found")
	}

	var b strings.Builder

	// Header
	header := fmt.Sprintf("%-25s %-25s %-12s %-40s",
		"Release", "Definition", "Status", "Environments")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, release := range releases {
		name := truncate(release.Name, 23)
		definition := truncate(release.ReleaseDefinition.Name, 23)
		status := string(release.Status)
		environments := truncate(release.GetEnvironmentSummary(), 38)

		statusDisplay := styles.GetStatusStyle(status).Render(status)

		row := fmt.Sprintf("%-25s %-25s %-12s %-40s",
			name, definition, statusDisplay, environments)

		if i == m.selectedRow {
			row = styles.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	return b.String()
}

// renderStatusBar renders the status bar
func (m Model) renderStatusBar() string {
	var parts []string

	// Last refresh time
	if !m.lastRefresh.IsZero() {
		parts = append(parts, fmt.Sprintf("Last refresh: %s",
			m.lastRefresh.Format(m.config.Display.DateFormat)))
	}

	// Next refresh
	parts = append(parts, fmt.Sprintf("Auto-refresh: %s",
		m.config.Display.RefreshInterval.String()))

	// Loading indicator
	var loadingCount int
	for _, loading := range m.loadingBuilds {
		if loading {
			loadingCount++
		}
	}
	for _, loading := range m.loadingReleases {
		if loading {
			loadingCount++
		}
	}
	if loadingCount > 0 {
		parts = append(parts, m.spinner.View()+" Loading...")
	}

	return styles.StatusBarStyle.Render(strings.Join(parts, " | "))
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
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
