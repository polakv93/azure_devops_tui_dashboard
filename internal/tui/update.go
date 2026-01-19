package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
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
			m.builds[msg.Project] = msg.Builds
		}
		return m, nil

	case ReleasesLoadedMsg:
		m.loadingReleases[msg.Project] = false
		if msg.Err != nil {
			m.errors[msg.Project+"-releases"] = msg.Err
		} else {
			delete(m.errors, msg.Project+"-releases")
			m.releases[msg.Project] = msg.Releases
		}
		return m, nil

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

	case key.Matches(msg, m.keys.Refresh):
		return m.handleRefresh()
	}

	return m, nil
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
