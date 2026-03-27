package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/api"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/styles"
)

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.showBuildLogViewer {
		return m.renderBuildLogViewer()
	}

	if m.runningPipeline {
		return m.renderRunPipelineWizard()
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

	b.WriteString("\n\n")

	// Pull Requests section
	b.WriteString(m.renderSectionHeader("Pull Requests", m.activeTab == TabPullRequests))
	b.WriteString("\n")
	if m.hasPullRequestData() {
		b.WriteString(m.renderPullRequestsTable())
	} else if m.loadingPullRequests[m.CurrentProject().Name] {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading pull requests...")
	} else if err := m.getPullRequestError(); err != nil {
		b.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", err)))
	} else {
		b.WriteString(styles.HelpStyle.Render("No pull requests found"))
	}

	// Status bar
	b.WriteString("\n\n")
	b.WriteString(m.renderStatusBar())

	if m.createPRSuccess != "" {
		b.WriteString("\n")
		b.WriteString(styles.SucceededStyle.Render("✓ " + m.createPRSuccess))
	}

	if m.runPipelineSuccess != "" {
		b.WriteString("\n")
		b.WriteString(styles.SucceededStyle.Render("✓ " + m.runPipelineSuccess))
	}

	if m.creatingPullRequest {
		b.WriteString("\n\n")
		b.WriteString(m.renderCreatePRWizard())
	}

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
	// Fixed columns: StagesOrStatus(40), Created(18), Duration(10) = 68
	// Variable columns: Pipeline, Branch
	fixedWidth := 40 + 18 + 10 + 4 // +4 for spacing
	availableWidth := m.width - fixedWidth
	if availableWidth < 40 {
		availableWidth = 40
	}
	pipelineWidth := availableWidth * 60 / 100    // 60% for pipeline
	branchWidth := availableWidth - pipelineWidth // rest for branch

	var b strings.Builder

	// Header
	headerFmt := fmt.Sprintf("%%-%ds %%-%ds %%-40s %%-18s %%-10s", pipelineWidth, branchWidth)
	header := fmt.Sprintf(headerFmt, "Pipeline", "Branch", "Stages / Status", "Created", "Duration")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, build := range builds {
		pipelineName := build.Definition.Name
		if m.isBuildDefinitionWatched(m.CurrentProject().Name, build.Definition.ID) {
			pipelineName = "[N] " + pipelineName
		}
		pipeline := truncate(pipelineName, pipelineWidth-2)
		branch := truncate(build.GetBranchName(), branchWidth-2)
		stagesDisplay := renderBuildStages(build)
		created := formatCreatedTime(build.QueueTime)
		duration := formatDuration(build.GetDuration())

		rowFmt := fmt.Sprintf("%%-%ds %%-%ds %%s %%-18s %%-10s", pipelineWidth, branchWidth)
		row := fmt.Sprintf(rowFmt, pipeline, branch, stagesDisplay, created, duration)

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
	// Fixed columns: Status(12), Created(18) = 30
	// Variable columns: Release, Definition, Environments
	fixedWidth := 12 + 18 + 4 // +4 for spacing
	availableWidth := m.width - fixedWidth
	if availableWidth < 60 {
		availableWidth = 60
	}
	releaseWidth := availableWidth * 20 / 100                            // 20% for release name
	definitionWidth := availableWidth * 25 / 100                         // 25% for definition
	environmentsWidth := availableWidth - releaseWidth - definitionWidth // rest for environments

	var b strings.Builder

	// Header
	headerFmt := fmt.Sprintf("%%-%ds %%-%ds %%-12s %%-18s %%-%ds", releaseWidth, definitionWidth, environmentsWidth)
	header := fmt.Sprintf(headerFmt, "Release", "Definition", "Status", "Created", "Environments")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, release := range releases {
		releaseName := release.Name
		if m.isReleaseDefinitionWatched(m.CurrentProject().Name, release.ReleaseDefinition.ID) {
			releaseName = "[N] " + releaseName
		}
		name := truncate(releaseName, releaseWidth-2)
		definition := truncate(release.ReleaseDefinition.Name, definitionWidth-2)
		status := string(release.Status)
		created := formatCreatedTime(release.CreatedOn)
		environments := renderColoredEnvironments(release.Environments)

		statusDisplay := styles.GetStatusStyle(status).Render(fmt.Sprintf("%-12s", status))

		rowFmt := fmt.Sprintf("%%-%ds %%-%ds %%s %%-18s %%s", releaseWidth, definitionWidth)
		row := fmt.Sprintf(rowFmt, name, definition, statusDisplay, created, environments)

		// Only show selection if Releases section is active
		if i == m.selectedRow && m.activeTab == TabReleases {
			row = styles.SelectedRowStyle.Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	return b.String()
}

// renderPullRequestsTable renders the pull requests table
func (m Model) renderPullRequestsTable() string {
	pullRequests := m.CurrentPullRequests()
	if len(pullRequests) == 0 {
		return styles.HelpStyle.Render("No pull requests found")
	}

	// Calculate dynamic column widths based on screen width
	// Fixed columns: Status(12), Reviewers(12), Created(18) = 42
	// Variable columns: Title, Repository, Branches, Author
	fixedWidth := 12 + 12 + 18 + 5 // +5 for spacing
	availableWidth := m.width - fixedWidth
	if availableWidth < 80 {
		availableWidth = 80
	}
	titleWidth := availableWidth * 35 / 100                                // 35% for title
	repoWidth := availableWidth * 15 / 100                                 // 15% for repository
	branchesWidth := availableWidth * 30 / 100                             // 30% for branches
	authorWidth := availableWidth - titleWidth - repoWidth - branchesWidth // rest for author

	var b strings.Builder

	// Header
	headerFmt := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds %%-12s %%-12s %%-18s", titleWidth, repoWidth, branchesWidth, authorWidth)
	header := fmt.Sprintf(headerFmt, "Title", "Repository", "Branches", "Author", "Status", "Reviewers", "Created")
	b.WriteString(styles.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Rows
	for i, pr := range pullRequests {
		titleText := pr.Title
		if m.isPullRequestWatched(m.CurrentProject().Name, pr.PullRequestID) {
			titleText = "[N] " + titleText
		}
		title := truncate(titleText, titleWidth-2)
		repo := truncate(pr.Repository.Name, repoWidth-2)
		branches := truncate(pr.GetBranchSummary(), branchesWidth-2)
		author := truncate(pr.CreatedBy.DisplayName, authorWidth-2)
		status := pr.GetStatusDisplay()
		reviewers := pr.GetReviewerSummary()
		created := formatCreatedTime(pr.CreationDate)

		statusDisplay := styles.GetPullRequestStatusStyle(status).Render(fmt.Sprintf("%-12s", status))

		rowFmt := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds %%s %%-12s %%-18s", titleWidth, repoWidth, branchesWidth, authorWidth)
		row := fmt.Sprintf(rowFmt, title, repo, branches, author, statusDisplay, reviewers, created)

		// Only show selection if Pull Requests section is active
		if i == m.selectedRow && m.activeTab == TabPullRequests {
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
	for _, loading := range m.loadingPullRequests {
		if loading {
			loadingCount++
		}
	}
	if loadingCount > 0 {
		parts = append(parts, m.spinner.View()+" Loading...")
	}

	if m.notificationError != nil {
		parts = append(parts, styles.ErrorStyle.Render("Notification error: "+m.notificationError.Error()))
	}

	if m.creatingPullRequest {
		parts = append(parts, styles.InProgressStyle.Render("Create PR mode"))
	}

	return styles.StatusBarStyle.Render(strings.Join(parts, " | "))
}

func (m Model) renderCreatePRWizard() string {
	var b strings.Builder
	maxRows := m.createPRListMaxRows()

	b.WriteString(styles.ActiveTabStyle.Render(" Create Pull Request "))
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("Esc back/cancel | Enter next/confirm | Space toggle option"))
	b.WriteString("\n\n")

	if m.createPRLoading {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading...")
		b.WriteString("\n")
	}

	if m.createPRError != "" {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.createPRError))
		b.WriteString("\n\n")
	}

	b.WriteString(m.renderCreatePRStepHeader(PRCreateStepRepository, "1. Repository"))
	b.WriteString("\n")
	repoStart, repoEnd := visibleListWindow(len(m.createPRRepositories), m.createPRSelectedRepo, maxRows)
	for i := repoStart; i < repoEnd; i++ {
		repo := m.createPRRepositories[i]
		line := "  " + repo.Name
		if i == m.createPRSelectedRepo && m.createPRStep == PRCreateStepRepository {
			line = styles.SelectedRowStyle.Render("► " + repo.Name)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(m.renderCreatePRListHint(len(m.createPRRepositories), repoStart, repoEnd, maxRows))

	b.WriteString("\n")
	b.WriteString(m.renderCreatePRStepHeader(PRCreateStepSourceBranch, "2. Source branch"))
	b.WriteString("\n")
	sourceOptionCount := m.createPRSourceOptionCount()
	if sourceOptionCount == 0 {
		b.WriteString(styles.HelpStyle.Render("  (no branches loaded yet)"))
		b.WriteString("\n")
	} else {
		start, end := visibleListWindow(sourceOptionCount, m.createPRSelectedSource, maxRows)
		for i := start; i < end; i++ {
			isLoadMoreOption := m.createPRHasMoreBranches && i == len(m.createPRBranches)
			line := ""
			if isLoadMoreOption {
				line = "  Load more branches..."
				if i == m.createPRSelectedSource && m.createPRStep == PRCreateStepSourceBranch {
					line = styles.SelectedRowStyle.Render("► Load more branches...")
				} else {
					line = styles.HelpStyle.Render(line)
				}
			} else {
				branch := m.createPRBranches[i]
				timeHint := ""
				if pushedAt, ok := m.createPRBranchPushTimes[branch]; ok {
					timeHint = "  " + styles.HelpStyle.Render("("+pushedAt.Local().Format("2006-01-02 15:04")+")")
				}
				line = "  " + branch + timeHint
				if i == m.createPRSelectedSource && m.createPRStep == PRCreateStepSourceBranch {
					line = styles.SelectedRowStyle.Render("► " + branch + timeHint)
				}
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString(m.renderCreatePRListHint(sourceOptionCount, start, end, maxRows))
		if m.createPRHasMoreBranches {
			b.WriteString(styles.HelpStyle.Render("  Showing recent branches (20). Select 'Load more branches...' for full list."))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.renderCreatePRStepHeader(PRCreateStepTargetBranch, "3. Target branch"))
	b.WriteString("\n")
	targets := m.createPRTargetOptions()
	if len(targets) == 0 {
		b.WriteString(styles.HelpStyle.Render("  (no target options)"))
		b.WriteString("\n")
	} else {
		start, end := visibleListWindow(len(targets), m.createPRSelectedTarget, maxRows)
		for i := start; i < end; i++ {
			target := targets[i]
			line := "  " + target
			if i == m.createPRSelectedTarget && m.createPRStep == PRCreateStepTargetBranch {
				line = styles.SelectedRowStyle.Render("► " + target)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString(m.renderCreatePRListHint(len(targets), start, end, maxRows))
	}

	b.WriteString("\n")
	b.WriteString(m.renderCreatePRStepHeader(PRCreateStepOptions, "4. Details and options"))
	b.WriteString("\n")
	options := []struct {
		Label   string
		Value   string
		IsInput bool
		Checked bool
	}{
		{Label: "Title", Value: m.createPRTitle, IsInput: true},
		{Label: "Description", Value: m.createPRDescription, IsInput: true},
		{Label: "Set automatic completion", Checked: m.createPRSetAutoComplete},
		{Label: "Approve as current user", Checked: m.createPRAutoApprove},
		{Label: "Delete source branch after merge", Checked: m.createPRDeleteSourceBranch},
		{Label: "Transition linked work items", Checked: m.createPRTransitionWorkItem},
	}
	for i, option := range options {
		line := ""
		if option.IsInput {
			value := option.Value
			if value == "" {
				value = "(empty)"
			}
			line = "  " + option.Label + ": " + value
			if m.createPREditField == PRCreateEditFieldTitle && i == 0 {
				line += styles.HelpStyle.Render("  [editing]")
			}
			if m.createPREditField == PRCreateEditFieldDescription && i == 1 {
				line += styles.HelpStyle.Render("  [editing]")
			}
		} else {
			marker := "[ ]"
			if option.Checked {
				marker = "[x]"
			}
			line = "  " + marker + " " + option.Label
		}
		if i == m.createPROptionCursor && m.createPRStep == PRCreateStepOptions {
			line = styles.SelectedRowStyle.Render("► " + strings.TrimLeft(line, " "))
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	if repo := m.currentCreatePRRepository(); repo != nil {
		source, ok := m.selectedCreatePRSourceBranch()
		if !ok {
			return b.String()
		}
		target := m.createPRTargetBranch
		if target == "" && len(targets) > 0 {
			target = targets[m.createPRSelectedTarget]
		}
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("Will create: %s/%s  %s -> %s", m.CurrentProject().Name, repo.Name, source, target)))
	}

	return b.String()
}

func (m Model) renderCreatePRStepHeader(step PRCreateStep, title string) string {
	if m.createPRStep == step {
		return styles.ActiveProjectStyle.Render(title)
	}
	if m.createPRStep > step {
		return styles.SucceededStyle.Render("✓ " + title)
	}
	return styles.TabStyle.Render(title)
}

func (m Model) renderCreatePRListHint(total, start, end, maxRows int) string {
	if total == 0 {
		return ""
	}
	if total <= maxRows {
		return styles.HelpStyle.Render(fmt.Sprintf("  (%d/%d)", total, total)) + "\n"
	}
	return styles.HelpStyle.Render(fmt.Sprintf("  (%d-%d/%d, PgUp/PgDn scroll)", start+1, end, total)) + "\n"
}

func visibleListWindow(total, selected, maxRows int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if maxRows < 1 {
		maxRows = 1
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= total {
		selected = total - 1
	}

	if total <= maxRows {
		return 0, total
	}

	start := selected - (maxRows / 2)
	if start < 0 {
		start = 0
	}
	end := start + maxRows
	if end > total {
		end = total
		start = end - maxRows
	}

	return start, end
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

// formatCreatedTime formats the created time for display
func formatCreatedTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04")
}

// renderColoredEnvironments renders environment summary with colored status icons
func renderColoredEnvironments(environments []api.ReleaseEnvironment) string {
	if len(environments) == 0 {
		return "-"
	}

	var parts []string
	for _, env := range environments {
		icon := getEnvStatusIcon(env.Status)
		coloredIcon := colorizeEnvIcon(icon, env.Status)
		parts = append(parts, env.Name+":"+coloredIcon)
	}

	return strings.Join(parts, " → ")
}

// getEnvStatusIcon returns the icon for environment status
func getEnvStatusIcon(status api.EnvironmentStatus) string {
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

// colorizeEnvIcon applies color to the status icon
func colorizeEnvIcon(icon string, status api.EnvironmentStatus) string {
	switch status {
	case api.EnvironmentStatusSucceeded:
		return styles.SucceededStyle.Render(icon)
	case api.EnvironmentStatusRejected, api.EnvironmentStatusCanceled:
		return styles.FailedStyle.Render(icon)
	case api.EnvironmentStatusInProgress:
		return styles.InProgressStyle.Render(icon)
	case api.EnvironmentStatusQueued, api.EnvironmentStatusScheduled:
		return styles.QueuedStyle.Render(icon)
	case api.EnvironmentStatusPartiallySucceeded:
		return styles.CanceledStyle.Render(icon)
	default:
		return styles.NotStartedStyle.Render(icon)
	}
}

// renderBuildStages renders build stage summary with colored status icons.
// When no stages are available, falls back to Status/Result display.
func renderBuildStages(build api.Build) string {
	if len(build.Stages) == 0 {
		status := string(build.Status)
		result := string(build.Result)
		statusStr := styles.GetStatusStyle(status).Render(fmt.Sprintf("%-12s", status))
		resultStr := styles.GetStatusStyle(result).Render(fmt.Sprintf("%-12s", result))
		return fmt.Sprintf("%s %s", statusStr, resultStr)
	}

	var parts []string
	for _, stage := range build.Stages {
		icon := getBuildStageIcon(stage)
		coloredIcon := colorizeBuildStageIcon(icon, stage)
		parts = append(parts, stage.Name+":"+coloredIcon)
	}

	return strings.Join(parts, " → ")
}

// getBuildStageIcon returns the icon for a build stage based on its state and result
func getBuildStageIcon(record api.BuildTimelineRecord) string {
	switch record.State {
	case api.BuildTimelineRecordStateCompleted:
		switch record.Result {
		case api.BuildTimelineRecordResultSucceeded:
			return "✓"
		case api.BuildTimelineRecordResultSucceededWithIssues:
			return "◐"
		case api.BuildTimelineRecordResultFailed:
			return "✗"
		case api.BuildTimelineRecordResultCanceled, api.BuildTimelineRecordResultAbandoned:
			return "⊘"
		case api.BuildTimelineRecordResultSkipped:
			return "⊝"
		default:
			return "?"
		}
	case api.BuildTimelineRecordStateInProgress:
		return "●"
	case api.BuildTimelineRecordStatePending:
		return "○"
	default:
		return "○"
	}
}

// colorizeBuildStageIcon applies color to a build stage icon
func colorizeBuildStageIcon(icon string, record api.BuildTimelineRecord) string {
	switch record.State {
	case api.BuildTimelineRecordStateCompleted:
		switch record.Result {
		case api.BuildTimelineRecordResultSucceeded:
			return styles.SucceededStyle.Render(icon)
		case api.BuildTimelineRecordResultSucceededWithIssues:
			return styles.CanceledStyle.Render(icon)
		case api.BuildTimelineRecordResultFailed:
			return styles.FailedStyle.Render(icon)
		case api.BuildTimelineRecordResultCanceled, api.BuildTimelineRecordResultAbandoned:
			return styles.CanceledStyle.Render(icon)
		case api.BuildTimelineRecordResultSkipped:
			return styles.NotStartedStyle.Render(icon)
		default:
			return styles.NotStartedStyle.Render(icon)
		}
	case api.BuildTimelineRecordStateInProgress:
		return styles.InProgressStyle.Render(icon)
	default:
		return styles.NotStartedStyle.Render(icon)
	}
}

func (m Model) renderBuildLogViewer() string {
	var b strings.Builder

	title := fmt.Sprintf(" Build Logs - %s #%d ", m.buildLogBuildName, m.buildLogBuildID)
	b.WriteString(styles.ActiveTabStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("Project: %s | Branch: %s", m.buildLogProject, m.buildLogBranch)))
	b.WriteString("\n")
	wrapState := "ON"
	if !m.buildLogWrapLines {
		wrapState = "OFF"
	}
	b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("Esc/o close | Up/Down switch log file | PgUp/PgDn scroll | r reload current log | w wrap: %s", wrapState)))
	b.WriteString("\n\n")

	if m.buildLogError != "" {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.buildLogError))
		b.WriteString("\n\n")
	}

	if len(m.buildLogEntries) == 0 {
		if m.buildLogLoading {
			b.WriteString(m.spinner.View() + " Loading log files...")
		} else {
			b.WriteString(styles.HelpStyle.Render("No log files available"))
		}
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render(m.help.View(m.keys)))
		return b.String()
	}

	b.WriteString(styles.TableHeaderStyle.Render("Log files"))
	b.WriteString("\n")
	for i, entry := range m.buildLogEntries {
		label := fmt.Sprintf("%d. log %d", i+1, entry.ID)
		if entry.LineCount > 0 {
			label = fmt.Sprintf("%s (%d lines)", label, entry.LineCount)
		}
		if i == m.buildLogSelected {
			label = styles.SelectedRowStyle.Render("► " + label)
		} else {
			label = "  " + label
		}
		b.WriteString(label)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.TableHeaderStyle.Render("Log content"))
	b.WriteString("\n")

	entry := m.currentBuildLogEntry()
	if entry == nil {
		b.WriteString(styles.HelpStyle.Render("No log selected"))
		b.WriteString("\n")
		return b.String()
	}

	lines := m.buildLogLines[entry.ID]
	if len(lines) == 0 {
		if m.buildLogLoading {
			b.WriteString(m.spinner.View() + " Loading log content...")
		} else {
			b.WriteString(styles.HelpStyle.Render("Log content is empty"))
		}
		b.WriteString("\n")
		return b.String()
	}

	height := m.buildLogViewportHeight()
	contentWidth := m.buildLogContentWidth()
	displayTotal := m.buildLogDisplayLineCount(lines)

	startDisplay := m.buildLogViewportTop
	if startDisplay < 0 {
		startDisplay = 0
	}
	if displayTotal > 0 && startDisplay >= displayTotal {
		startDisplay = displayTotal - 1
	}
	if startDisplay < 0 {
		startDisplay = 0
	}
	endDisplay := startDisplay + height
	if endDisplay > displayTotal {
		endDisplay = displayTotal
	}

	if !m.buildLogWrapLines {
		start := startDisplay
		end := endDisplay
		if end > len(lines) {
			end = len(lines)
		}
		for i := start; i < end; i++ {
			lineNo := fmt.Sprintf("%6d", i+1)
			b.WriteString(styles.HelpStyle.Render(lineNo + " | "))
			b.WriteString(lines[i])
			b.WriteString("\n")
		}
	} else {
		displayCursor := 0
		for i, raw := range lines {
			r := []rune(raw)
			segments := m.buildLogLineSegmentCount(raw, contentWidth)
			for seg := 0; seg < segments; seg++ {
				if displayCursor >= endDisplay {
					break
				}
				if displayCursor >= startDisplay {
					prefix := "      | "
					if seg == 0 {
						prefix = fmt.Sprintf("%6d | ", i+1)
					}
					b.WriteString(styles.HelpStyle.Render(prefix))

					chunkStart := seg * contentWidth
					chunkEnd := chunkStart + contentWidth
					if chunkStart > len(r) {
						chunkStart = len(r)
					}
					if chunkEnd > len(r) {
						chunkEnd = len(r)
					}
					if chunkStart < chunkEnd {
						b.WriteString(string(r[chunkStart:chunkEnd]))
					}
					b.WriteString("\n")
				}
				displayCursor++
			}
			if displayCursor >= endDisplay {
				break
			}
		}
	}

	state := "more lines available"
	if m.buildLogExhausted[entry.ID] {
		state = "end of log"
	}
	if m.buildLogLoading {
		state = "loading more lines..."
	}
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("Showing rows %d-%d of %d (source lines: %d, wrap: %s, %s)", startDisplay+1, endDisplay, displayTotal, len(lines), wrapState, state)))

	return b.String()
}

func (m Model) renderRunPipelineWizard() string {
	var b strings.Builder
	maxRows := m.createPRListMaxRows()

	b.WriteString(styles.ActiveTabStyle.Render(" Run Pipeline "))
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("Esc cancel | Enter queue selected branch | Up/Down select branch"))
	b.WriteString("\n\n")

	b.WriteString(styles.TableHeaderStyle.Render("Pipeline"))
	b.WriteString("\n")
	b.WriteString("  " + m.runPipelineDefinition)
	b.WriteString("\n")
	if m.runPipelineRepository != "" {
		b.WriteString(styles.HelpStyle.Render("Repository: " + m.runPipelineRepository))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.runPipelineLoading {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading branches...")
		b.WriteString("\n")
	}

	if m.runPipelineError != "" {
		b.WriteString(styles.ErrorStyle.Render("Error: " + m.runPipelineError))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.TableHeaderStyle.Render("Branches"))
	b.WriteString("\n")

	if len(m.runPipelineBranches) == 0 {
		b.WriteString(styles.HelpStyle.Render("  (no branches loaded yet)"))
		b.WriteString("\n")
	} else {
		start, end := visibleListWindow(len(m.runPipelineBranches), m.runPipelineSelected, maxRows)
		for i := start; i < end; i++ {
			branch := m.runPipelineBranches[i]
			timeHint := ""
			if pushedAt, ok := m.runPipelineBranchPush[branch]; ok {
				timeHint = "  " + styles.HelpStyle.Render("("+pushedAt.Local().Format("2006-01-02 15:04")+")")
			}
			line := "  " + branch + timeHint
			if i == m.runPipelineSelected {
				line = styles.SelectedRowStyle.Render("► " + branch + timeHint)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString(m.renderCreatePRListHint(len(m.runPipelineBranches), start, end, maxRows))
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("Tip: branches are sorted by recent push activity"))
	b.WriteString("\n\n")
	b.WriteString(styles.HelpStyle.Render(m.help.View(m.keys)))

	return b.String()
}
