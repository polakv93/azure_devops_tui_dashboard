package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/api"
)

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case BuildsLoadedMsg:
		m.loadingBuilds[msg.Project] = false
		if msg.Err != nil {
			m.errors[msg.Project+"-builds"] = msg.Err
		} else {
			delete(m.errors, msg.Project+"-builds")
			cmds = append(cmds, m.buildCompletionNotificationCmds(msg.Project, msg.Builds)...)
			m.builds[msg.Project] = msg.Builds
		}
		return m, tea.Batch(cmds...)

	case ReleasesLoadedMsg:
		m.loadingReleases[msg.Project] = false
		if msg.Err != nil {
			m.errors[msg.Project+"-releases"] = msg.Err
		} else {
			delete(m.errors, msg.Project+"-releases")
			cmds = append(cmds, m.releaseCompletionNotificationCmds(msg.Project, msg.Releases)...)
			m.releases[msg.Project] = msg.Releases
		}
		return m, tea.Batch(cmds...)

	case PullRequestsLoadedMsg:
		m.loadingPullRequests[msg.Project] = false
		if msg.Err != nil {
			m.errors[msg.Project+"-pullrequests"] = msg.Err
		} else {
			delete(m.errors, msg.Project+"-pullrequests")
			m.pullRequests[msg.Project] = msg.PullRequests
		}
		return m, nil

	case RefreshTickMsg:
		return m.handleRefresh()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case NotificationErrorMsg:
		m.notificationError = msg.Err
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Tab):
		m.activeTab = (m.activeTab + 1) % 3
		m.selectedRow = 0
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.selectedRow > 0 {
			m.selectedRow--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		maxRows := m.MaxRows()
		if m.selectedRow < maxRows-1 {
			m.selectedRow++
		}
		return m, nil

	case key.Matches(msg, m.keys.Left):
		if m.activeProject > 0 {
			m.activeProject--
			m.selectedRow = 0
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		if m.activeProject < len(m.config.Projects)-1 {
			m.activeProject++
			m.selectedRow = 0
		}
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter()

	case key.Matches(msg, m.keys.Notify):
		switch m.activeTab {
		case TabBuilds:
			m.toggleBuildDefinitionWatch()
		case TabReleases:
			m.toggleReleaseDefinitionWatch()
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m.handleRefresh()
	}

	return m, nil
}

// buildCompletionNotificationCmds returns notification commands for watched build definitions.
func (m *Model) buildCompletionNotificationCmds(project string, builds []api.Build) []tea.Cmd {
	known := m.knownBuildCompletion[project]
	if known == nil {
		known = make(map[int]bool)
	}

	watched := m.watchedBuildDefinitions[project]
	initialized := m.buildSnapshotInitialized[project]

	var cmds []tea.Cmd
	newKnown := make(map[int]bool, len(builds))

	for _, build := range builds {
		completed := build.IsCompleted()
		newKnown[build.ID] = completed

		if !watched[build.Definition.ID] {
			continue
		}

		if !initialized {
			continue
		}

		prevCompleted := known[build.ID]
		if completed && !prevCompleted {
			title := fmt.Sprintf("Pipeline finished: %s", build.Definition.Name)
			body := fmt.Sprintf("Project: %s | Result: %s", project, build.GetStatusString())
			cmds = append(cmds, notifyDesktop(title, body))
		}
	}

	m.knownBuildCompletion[project] = newKnown
	m.buildSnapshotInitialized[project] = true

	return cmds
}

// releaseCompletionNotificationCmds returns notification commands for watched release definitions.
func (m *Model) releaseCompletionNotificationCmds(project string, releases []api.Release) []tea.Cmd {
	known := m.knownReleaseCompletion[project]
	if known == nil {
		known = make(map[int]bool)
	}

	watched := m.watchedReleaseDefinitions[project]
	initialized := m.releaseSnapshotInitialized[project]

	var cmds []tea.Cmd
	newKnown := make(map[int]bool, len(releases))

	for _, release := range releases {
		completed := isReleaseCompleted(release)
		newKnown[release.ID] = completed

		if !watched[release.ReleaseDefinition.ID] {
			continue
		}

		if !initialized {
			continue
		}

		prevCompleted := known[release.ID]
		if completed && !prevCompleted {
			title := fmt.Sprintf("Release finished: %s", release.ReleaseDefinition.Name)
			body := fmt.Sprintf("Project: %s | Release: %s | Status: %s", project, release.Name, releaseCompletionStatus(release))
			cmds = append(cmds, notifyDesktop(title, body))
		}
	}

	m.knownReleaseCompletion[project] = newKnown
	m.releaseSnapshotInitialized[project] = true

	return cmds
}

// isReleaseCompleted returns true if release is no longer progressing.
func isReleaseCompleted(release api.Release) bool {
	if release.Status == api.ReleaseStatusAbandoned {
		return true
	}

	if len(release.Environments) == 0 {
		return release.Status != api.ReleaseStatusActive
	}

	for _, env := range release.Environments {
		switch env.Status {
		case api.EnvironmentStatusInProgress,
			api.EnvironmentStatusQueued,
			api.EnvironmentStatusScheduled,
			api.EnvironmentStatusNotStarted:
			return false
		}
	}

	return true
}

// releaseCompletionStatus returns a user-facing completion status for notification text.
func releaseCompletionStatus(release api.Release) string {
	if release.Status == api.ReleaseStatusAbandoned {
		return string(release.Status)
	}

	return string(release.GetOverallStatus())
}

// handleEnter opens the selected build/release/pull request in the browser
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	project := m.CurrentProject().Name
	var url string

	switch m.activeTab {
	case TabBuilds:
		builds := m.CurrentBuilds()
		if m.selectedRow >= 0 && m.selectedRow < len(builds) {
			build := builds[m.selectedRow]
			if build.Links.Web.Href != "" {
				url = build.Links.Web.Href
			} else {
				url = m.client.GetBuildWebURL(project, build.ID)
			}
		}

	case TabReleases:
		releases := m.CurrentReleases()
		if m.selectedRow >= 0 && m.selectedRow < len(releases) {
			release := releases[m.selectedRow]
			if release.Links.Web.Href != "" {
				url = release.Links.Web.Href
			} else {
				url = m.client.GetReleaseWebURL(project, release.ID)
			}
		}

	case TabPullRequests:
		pullRequests := m.CurrentPullRequests()
		if m.selectedRow >= 0 && m.selectedRow < len(pullRequests) {
			pr := pullRequests[m.selectedRow]
			url = m.client.GetPullRequestWebURL(project, pr.Repository.Name, pr.PullRequestID)
		}
	}

	if url != "" {
		return m, openBrowser(url)
	}

	return m, nil
}

// handleRefresh triggers a data refresh
func (m Model) handleRefresh() (tea.Model, tea.Cmd) {
	// Skip if already loading (prevent multiple queued refreshes)
	if m.isAnyLoading() {
		return m, nil
	}

	// Mark all as loading
	for _, p := range m.config.Projects {
		m.loadingBuilds[p.Name] = true
		m.loadingReleases[p.Name] = true
		m.loadingPullRequests[p.Name] = true
	}

	m.lastRefresh = time.Now()

	return m, tea.Batch(
		fetchAllData(m.client, m.config.Projects, m.config.Display.MaxItemsPerProject),
		refreshTicker(m.config.Display.RefreshInterval),
	)
}

// isAnyLoading returns true if any project is currently loading data
func (m Model) isAnyLoading() bool {
	for _, loading := range m.loadingBuilds {
		if loading {
			return true
		}
	}
	for _, loading := range m.loadingReleases {
		if loading {
			return true
		}
	}
	for _, loading := range m.loadingPullRequests {
		if loading {
			return true
		}
	}
	return false
}
