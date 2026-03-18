package tui

import (
	"fmt"
	"sort"
	"strings"
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
		if m.creatingPullRequest {
			return m.handleCreatePRKeyMsg(msg)
		}
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
			if m.creatingPullRequest && m.createPRStep == PRCreateStepSourceBranch {
				cmds = append(cmds, m.loadBranchesForCreatePR())
			}
		}
		return m, tea.Batch(cmds...)

	case CreatePRDataLoadedMsg:
		m.createPRLoading = false
		if msg.Err != nil {
			m.createPRError = msg.Err.Error()
			return m, nil
		}

		m.createPRError = ""
		m.createPRRepositories = msg.Repositories
		if len(msg.Repositories) == 0 {
			m.createPRError = "no repositories available for this project"
			return m, nil
		}

		m.createPRStep = PRCreateStepRepository
		m.createPRSelectedRepo = 0
		return m, nil

	case CreatePRBranchesLoadedMsg:
		m.createPRLoading = false
		if msg.Err != nil {
			m.createPRError = msg.Err.Error()
			return m, nil
		}

		repo := m.currentCreatePRRepository()
		if repo == nil || repo.ID != msg.RepositoryID {
			return m, nil
		}

		m.createPRError = ""
		m.createPRBranches = msg.Branches
		m.createPRBranchPushTimes = msg.PushTimes
		m.createPRSelectedSource = 0
		if len(msg.Branches) == 0 {
			m.createPRError = "no source branches available (all already have active PR or only target branch exists)"
		}
		return m, nil

	case CreatePRDefaultsLoadedMsg:
		m.createPRLoading = false

		repo := m.currentCreatePRRepository()
		if repo == nil || repo.ID != msg.RepositoryID {
			return m, nil
		}
		if len(m.createPRBranches) == 0 || m.createPRSelectedSource < 0 || m.createPRSelectedSource >= len(m.createPRBranches) {
			return m, nil
		}
		if m.createPRBranches[m.createPRSelectedSource] != msg.SourceBranch {
			return m, nil
		}

		if msg.Err != nil {
			if m.createPRTitleAuto && strings.TrimSpace(m.createPRTitle) == "" {
				m.createPRTitle = msg.SourceBranch
			}
			return m, nil
		}

		if m.createPRTitleAuto {
			m.createPRTitle = msg.Title
		}
		if m.createPRDescriptionAuto {
			m.createPRDescription = msg.Description
		}
		return m, nil

	case PullRequestCreatedMsg:
		m.createPRLoading = false
		if msg.Err != nil {
			m.createPRError = msg.Err.Error()
			return m, nil
		}

		m.createPRSuccess = fmt.Sprintf("created PR #%d (%s -> %s)", msg.PullRequest.PullRequestID, msg.PullRequest.GetSourceBranch(), msg.PullRequest.GetTargetBranch())
		m.resetCreatePRFlow()
		m.activeTab = TabPullRequests
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
		m.createPRSuccess = ""
		return m.handleEnter()

	case key.Matches(msg, m.keys.Create):
		if m.activeTab != TabPullRequests {
			return m, nil
		}
		m.createPRSuccess = ""
		m.creatingPullRequest = true
		m.createPRStep = PRCreateStepRepository
		m.createPRLoading = true
		m.createPRError = ""
		m.createPRRepositories = nil
		m.createPRBranches = nil
		m.createPRSelectedRepo = 0
		m.createPRSelectedSource = 0
		m.createPRSelectedTarget = 0
		m.createPRTargetBranch = ""
		return m, loadCreatePRData(m.client, m.CurrentProject())

	case key.Matches(msg, m.keys.Notify):
		switch m.activeTab {
		case TabBuilds:
			m.toggleBuildDefinitionWatch()
		case TabReleases:
			m.toggleReleaseDefinitionWatch()
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		m.createPRSuccess = ""
		return m.handleRefresh()
	}

	return m, nil
}

func (m *Model) handleCreatePRKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.createPREditField != PRCreateEditFieldNone {
		switch msg.Type {
		case tea.KeyEsc, tea.KeyEnter:
			m.createPREditField = PRCreateEditFieldNone
			return m, nil
		case tea.KeyBackspace:
			m.backspaceCreatePREditField()
			return m, nil
		default:
			s := msg.String()
			if s != "" && s != "space" {
				m.appendCreatePREditField(s)
				return m, nil
			}
			if msg.Type == tea.KeySpace {
				m.appendCreatePREditField(" ")
				return m, nil
			}
		}
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Back):
		if m.createPRLoading {
			return m, nil
		}
		switch m.createPRStep {
		case PRCreateStepRepository:
			m.resetCreatePRFlow()
		case PRCreateStepSourceBranch:
			m.createPRStep = PRCreateStepRepository
		case PRCreateStepTargetBranch:
			m.createPRStep = PRCreateStepSourceBranch
		case PRCreateStepOptions:
			m.createPRStep = PRCreateStepTargetBranch
		}
		m.createPRError = ""
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.createPRLoading {
			return m, nil
		}
		switch m.createPRStep {
		case PRCreateStepRepository:
			if m.createPRSelectedRepo > 0 {
				m.createPRSelectedRepo--
			}
		case PRCreateStepSourceBranch:
			if m.createPRSelectedSource > 0 {
				m.createPRSelectedSource--
			}
		case PRCreateStepTargetBranch:
			if m.createPRSelectedTarget > 0 {
				m.createPRSelectedTarget--
			}
		case PRCreateStepOptions:
			if m.createPROptionCursor > 0 {
				m.createPROptionCursor--
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.createPRLoading {
			return m, nil
		}
		switch m.createPRStep {
		case PRCreateStepRepository:
			if m.createPRSelectedRepo < len(m.createPRRepositories)-1 {
				m.createPRSelectedRepo++
			}
		case PRCreateStepSourceBranch:
			if m.createPRSelectedSource < len(m.createPRBranches)-1 {
				m.createPRSelectedSource++
			}
		case PRCreateStepTargetBranch:
			targets := m.createPRTargetOptions()
			if m.createPRSelectedTarget < len(targets)-1 {
				m.createPRSelectedTarget++
			}
		case PRCreateStepOptions:
			if m.createPROptionCursor < 5 {
				m.createPROptionCursor++
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Space):
		if m.createPRLoading {
			return m, nil
		}
		if m.createPRStep == PRCreateStepOptions {
			m.toggleOrEditCreatePROption()
		}
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		if m.createPRLoading {
			return m, nil
		}
		return m.handleCreatePREnter()

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m.handleRefresh()
	}

	return m, nil
}

func (m *Model) handleCreatePREnter() (tea.Model, tea.Cmd) {
	switch m.createPRStep {
	case PRCreateStepRepository:
		repo := m.currentCreatePRRepository()
		if repo == nil {
			m.createPRError = "select repository first"
			return m, nil
		}

		m.createPRTargetBranch = trimRefPrefix(repo.DefaultBranch)
		m.createPRSelectedTarget = 0
		m.createPRStep = PRCreateStepSourceBranch
		m.createPRLoading = true
		m.createPRError = ""
		return m, m.loadBranchesForCreatePR()

	case PRCreateStepSourceBranch:
		if len(m.createPRBranches) == 0 {
			m.createPRError = "no source branch selected"
			return m, nil
		}
		m.createPRStep = PRCreateStepTargetBranch
		m.createPRSelectedTarget = 0
		m.createPRError = ""
		m.createPRLoading = true
		repo := m.currentCreatePRRepository()
		if repo == nil {
			return m, nil
		}
		source := m.createPRBranches[m.createPRSelectedSource]
		return m, loadCreatePRDefaults(m.client, m.CurrentProject().Name, repo.ID, source)

	case PRCreateStepTargetBranch:
		targets := m.createPRTargetOptions()
		if len(targets) == 0 {
			m.createPRError = "no target branches available"
			return m, nil
		}
		if m.createPRSelectedTarget < 0 || m.createPRSelectedTarget >= len(targets) {
			m.createPRSelectedTarget = 0
		}
		m.createPRTargetBranch = targets[m.createPRSelectedTarget]
		m.createPRStep = PRCreateStepOptions
		m.createPROptionCursor = 0
		m.createPRError = ""
		return m, nil

	case PRCreateStepOptions:
		if m.createPROptionCursor == 0 {
			m.createPREditField = PRCreateEditFieldTitle
			m.createPRTitleAuto = false
			return m, nil
		}
		if m.createPROptionCursor == 1 {
			m.createPREditField = PRCreateEditFieldDescription
			m.createPRDescriptionAuto = false
			return m, nil
		}

		repo := m.currentCreatePRRepository()
		if repo == nil {
			m.createPRError = "repository not selected"
			return m, nil
		}
		if len(m.createPRBranches) == 0 || m.createPRSelectedSource < 0 || m.createPRSelectedSource >= len(m.createPRBranches) {
			m.createPRError = "source branch not selected"
			return m, nil
		}
		source := m.createPRBranches[m.createPRSelectedSource]
		target := m.createPRTargetBranch
		if target == "" {
			target = m.selectedCreatePRTargetBranch()
		}
		if target == "" {
			m.createPRError = "target branch not selected"
			return m, nil
		}
		title := strings.TrimSpace(m.createPRTitle)
		if title == "" {
			title = source
		}

		m.createPRLoading = true
		m.createPRError = ""
		return m, createPullRequestWithOptions(
			m.client,
			m.CurrentProject().Name,
			*repo,
			source,
			target,
			title,
			strings.TrimSpace(m.createPRDescription),
			m.createPRSetAutoComplete,
			m.createPRAutoApprove,
			m.createPRDeleteSourceBranch,
			m.createPRTransitionWorkItem,
		)
	}

	return m, nil
}

func (m *Model) toggleOrEditCreatePROption() {
	switch m.createPROptionCursor {
	case 0:
		m.createPREditField = PRCreateEditFieldTitle
		m.createPRTitleAuto = false
	case 1:
		m.createPREditField = PRCreateEditFieldDescription
		m.createPRDescriptionAuto = false
	case 2:
		m.createPRSetAutoComplete = !m.createPRSetAutoComplete
	case 3:
		m.createPRAutoApprove = !m.createPRAutoApprove
	case 4:
		m.createPRDeleteSourceBranch = !m.createPRDeleteSourceBranch
	case 5:
		m.createPRTransitionWorkItem = !m.createPRTransitionWorkItem
	}
}

func (m *Model) appendCreatePREditField(s string) {
	switch m.createPREditField {
	case PRCreateEditFieldTitle:
		m.createPRTitle += s
	case PRCreateEditFieldDescription:
		m.createPRDescription += s
	}
}

func (m *Model) backspaceCreatePREditField() {
	switch m.createPREditField {
	case PRCreateEditFieldTitle:
		if len(m.createPRTitle) > 0 {
			m.createPRTitle = m.createPRTitle[:len(m.createPRTitle)-1]
		}
	case PRCreateEditFieldDescription:
		if len(m.createPRDescription) > 0 {
			m.createPRDescription = m.createPRDescription[:len(m.createPRDescription)-1]
		}
	}
}

func (m *Model) currentCreatePRRepository() *api.Repository {
	if m.createPRSelectedRepo < 0 || m.createPRSelectedRepo >= len(m.createPRRepositories) {
		return nil
	}
	repo := m.createPRRepositories[m.createPRSelectedRepo]
	return &repo
}

func (m *Model) createPRTargetOptions() []string {
	seen := make(map[string]bool)
	var targets []string

	repo := m.currentCreatePRRepository()
	if repo != nil {
		defaultBranch := trimRefPrefix(repo.DefaultBranch)
		if defaultBranch != "" {
			seen[defaultBranch] = true
			targets = append(targets, defaultBranch)
		}
	}

	for _, branch := range m.createPRBranches {
		if !seen[branch] {
			seen[branch] = true
			targets = append(targets, branch)
		}
	}

	if len(targets) == 0 && m.createPRTargetBranch != "" {
		targets = append(targets, m.createPRTargetBranch)
	}

	selectedSource := ""
	if len(m.createPRBranches) > 0 && m.createPRSelectedSource >= 0 && m.createPRSelectedSource < len(m.createPRBranches) {
		selectedSource = m.createPRBranches[m.createPRSelectedSource]
	}

	filtered := make([]string, 0, len(targets))
	for _, target := range targets {
		if target == selectedSource {
			continue
		}
		filtered = append(filtered, target)
	}

	if len(filtered) == 0 && m.createPRTargetBranch != "" && m.createPRTargetBranch != selectedSource {
		filtered = append(filtered, m.createPRTargetBranch)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		ti, okI := m.createPRBranchPushTimes[filtered[i]]
		tj, okJ := m.createPRBranchPushTimes[filtered[j]]
		switch {
		case okI && okJ:
			if !ti.Equal(tj) {
				return ti.After(tj)
			}
		case okI:
			return true
		case okJ:
			return false
		}

		return filtered[i] < filtered[j]
	})

	return filtered
}

func (m *Model) selectedCreatePRTargetBranch() string {
	targets := m.createPRTargetOptions()
	if len(targets) == 0 {
		return ""
	}
	if m.createPRSelectedTarget < 0 || m.createPRSelectedTarget >= len(targets) {
		return targets[0]
	}
	return targets[m.createPRSelectedTarget]
}

func (m *Model) loadBranchesForCreatePR() tea.Cmd {
	repo := m.currentCreatePRRepository()
	if repo == nil {
		return nil
	}

	projectName := m.CurrentProject().Name
	activePRs := m.CurrentPullRequests()
	return loadCreatePRBranches(m.client, projectName, repo.ID, activePRs, m.createPRTargetBranch)
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
