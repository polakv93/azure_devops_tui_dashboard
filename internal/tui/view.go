package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/styles"
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

	// Project tabs
	b.WriteString(m.renderProjectTabs())
	b.WriteString("\n\n")

	// Builds section
	branchInfo := m.getBranchFilterInfo()
	b.WriteString(m.renderSectionHeader("Builds", m.activeTab == TabBuilds))
	b.WriteString(styles.HelpStyle.Render(fmt.Sprintf(" (branches: %s)", branchInfo)))
	b.WriteString("\n")
	if m.hasBuildData() {
		b.WriteString(m.renderBuildsTable())
	} else if m.loadingBuilds[m.CurrentProject().Name] {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading builds...")
	} else if err := m.getBuildError(); err != nil {
		b.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err)))
	} else {
		b.WriteString(styles.HelpStyle.Render("No builds found"))
	}

	b.WriteString("\n\n")

	// Releases section
	b.WriteString(m.renderSectionHeader("Releases", m.activeTab == TabReleases))
	b.WriteString("\n")
	if m.hasReleaseData() {
		b.WriteString(m.renderReleasesTable())
	} else if m.loadingReleases[m.CurrentProject().Name] {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading releases...")
	} else if err := m.getReleaseError(); err != nil {
		b.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err)))
	} else {
		b.WriteString(styles.HelpStyle.Render("No releases found"))
	}

	// Status bar
	b.WriteString("\n\n")
	b.WriteString(m.renderStatusBar())

	// Help
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render(m.help.View(m.keys)))

	return b.String()
}

// renderSectionHeader renders a section header with active indicator
func (m Model) renderSectionHeader(title string, isActive bool) string {
	if isActive {
		return styles.ActiveTabStyle.Render("► " + title)
	}
	return styles.TabStyle.Render("  " + title)
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

	// Calculate dynamic column widths based on screen width
	// Fixed columns: Status(12), Result(12), Duration(10) = 34
	// Variable columns: Pipeline, Branch
	fixedWidth := 12 + 12 + 10 + 4 // +4 for spacing
	availableWidth := m.width - fixedWidth
	if availableWidth < 40 {
		availableWidth = 40
	}
	pipelineWidth := availableWidth * 55 / 100 // 55% for pipeline
	branchWidth := availableWidth - pipelineWidth // rest for branch

	var b strings.Builder

	// Header
	headerFmt := fmt.Sprintf("%%-%ds %%-%ds %%-12s %%-12s %%-10s", pipelineWidth, branchWidth)
	header := fmt.Sprintf(headerFmt, "Pipeline", "Branch", "Status", "Result", "Duration")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, build := range builds {
		pipeline := truncate(build.Definition.Name, pipelineWidth-2)
		branch := truncate(build.GetBranchName(), branchWidth-2)
		status := string(build.Status)
		result := string(build.Result)
		duration := formatDuration(build.GetDuration())

		statusDisplay := styles.GetStatusStyle(status).Render(fmt.Sprintf("%-12s", status))
		resultDisplay := styles.GetStatusStyle(result).Render(fmt.Sprintf("%-12s", result))

		rowFmt := fmt.Sprintf("%%-%ds %%-%ds %%s %%s %%-10s", pipelineWidth, branchWidth)
		row := fmt.Sprintf(rowFmt, pipeline, branch, statusDisplay, resultDisplay, duration)

		// Only show selection if Builds section is active
		if i == m.selectedRow && m.activeTab == TabBuilds {
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

	// Calculate dynamic column widths based on screen width
	// Fixed columns: Status(12) = 12
	// Variable columns: Release, Definition, Environments
	fixedWidth := 12 + 3 // +3 for spacing
	availableWidth := m.width - fixedWidth
	if availableWidth < 60 {
		availableWidth = 60
	}
	releaseWidth := availableWidth * 20 / 100      // 20% for release name
	definitionWidth := availableWidth * 25 / 100  // 25% for definition
	environmentsWidth := availableWidth - releaseWidth - definitionWidth // rest for environments

	var b strings.Builder

	// Header
	headerFmt := fmt.Sprintf("%%-%ds %%-%ds %%-12s %%-%ds", releaseWidth, definitionWidth, environmentsWidth)
	header := fmt.Sprintf(headerFmt, "Release", "Definition", "Status", "Environments")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, release := range releases {
		name := truncate(release.Name, releaseWidth-2)
		definition := truncate(release.ReleaseDefinition.Name, definitionWidth-2)
		status := string(release.Status)
		environments := truncate(release.GetEnvironmentSummary(), environmentsWidth-2)

		statusDisplay := styles.GetStatusStyle(status).Render(fmt.Sprintf("%-12s", status))

		rowFmt := fmt.Sprintf("%%-%ds %%-%ds %%s %%-%ds", releaseWidth, definitionWidth, environmentsWidth)
		row := fmt.Sprintf(rowFmt, name, definition, statusDisplay, environments)

		// Only show selection if Releases section is active
		if i == m.selectedRow && m.activeTab == TabReleases {
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
