package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/api"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/config"
)

// Tab represents the active view tab
type Tab int

const (
	TabBuilds Tab = iota
	TabReleases
	TabPullRequests
)

// Model is the main application model
type Model struct {
	// Configuration
	config *config.Config
	client *api.Client

	// UI state
	activeTab     Tab
	activeProject int
	selectedRow   int
	width         int
	height        int
	showHelp      bool

	// Data
	builds       map[string][]api.Build       // project name -> builds
	releases     map[string][]api.Release     // project name -> releases
	pullRequests map[string][]api.PullRequest // project name -> pull requests

	// Loading states
	loadingBuilds       map[string]bool
	loadingReleases     map[string]bool
	loadingPullRequests map[string]bool

	// Errors
	errors map[string]error

	// Last refresh time
	lastRefresh       time.Time
	notificationError error

	// Session-only notification watches (project -> entity ID -> watched)
	watchedBuildDefinitions   map[string]map[int]bool
	watchedReleaseDefinitions map[string]map[int]bool
	watchedPullRequests       map[string]map[int]bool

	// Previous completion snapshots for transition detection
	knownBuildCompletion           map[string]map[int]bool // project -> build ID -> completed
	knownReleaseCompletion         map[string]map[int]bool // project -> release ID -> completed
	knownPullRequests              map[string]map[int]api.PullRequest
	buildSnapshotInitialized       map[string]bool // project -> snapshot initialized
	releaseSnapshotInitialized     map[string]bool // project -> snapshot initialized
	pullRequestSnapshotInitialized map[string]bool // project -> snapshot initialized

	// Components
	spinner spinner.Model
	help    help.Model
	keys    KeyMap

	// Pull request creation flow
	creatingPullRequest        bool
	createPRStep               PRCreateStep
	createPRRepositories       []api.Repository
	createPRSelectedRepo       int
	createPRBranches           []string
	createPRBranchPushTimes    map[string]time.Time
	createPRHasMoreBranches    bool
	createPRLoadAllBranches    bool
	createPRSelectedSource     int
	createPRTargetBranch       string
	createPRSelectedTarget     int
	createPROptionCursor       int
	createPRSetAutoComplete    bool
	createPRAutoApprove        bool
	createPRDeleteSourceBranch bool
	createPRTransitionWorkItem bool
	createPRLoading            bool
	createPRError              string
	createPRSuccess            string
	createPRTitle              string
	createPRDescription        string
	createPRTitleAuto          bool
	createPRDescriptionAuto    bool
	createPREditField          PRCreateEditField

	// Build log viewer
	showBuildLogViewer  bool
	buildLogLoading     bool
	buildLogError       string
	buildLogProject     string
	buildLogBuildID     int
	buildLogBuildName   string
	buildLogBranch      string
	buildLogEntries     []api.BuildLog
	buildLogSelected    int
	buildLogLines       map[int][]string
	buildLogLoadedUntil map[int]int
	buildLogExhausted   map[int]bool
	buildLogViewportTop int
	buildLogWrapLines   bool

	// Run pipeline flow
	runningPipeline         bool
	runPipelineLoading      bool
	runPipelineError        string
	runPipelineSuccess      string
	runPipelineBranches     []string
	runPipelineBranchPush   map[string]time.Time
	runPipelineSelected     int
	runPipelineDefinitionID int
	runPipelineDefinition   string
	runPipelineRepositoryID string
	runPipelineRepository   string
}

const (
	createPRMinListRows       = 3
	createPRMaxListRows       = 12
	createPRRecentBranchLimit = 20
)

const buildLogChunkSize = 500

// PRCreateStep represents step in create pull request wizard.
type PRCreateStep int

const (
	PRCreateStepRepository PRCreateStep = iota
	PRCreateStepSourceBranch
	PRCreateStepTargetBranch
	PRCreateStepOptions
)

// PRCreateEditField tracks which text field is currently edited in create PR wizard.
type PRCreateEditField int

const (
	PRCreateEditFieldNone PRCreateEditField = iota
	PRCreateEditFieldTitle
	PRCreateEditFieldDescription
)

// NewModel creates a new Model with the given configuration
func NewModel(cfg *config.Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))

	h := help.New()
	h.ShowAll = false

	return Model{
		config:                         cfg,
		client:                         newClientFromConfig(cfg),
		activeTab:                      TabBuilds,
		activeProject:                  0,
		selectedRow:                    0,
		builds:                         make(map[string][]api.Build),
		releases:                       make(map[string][]api.Release),
		pullRequests:                   make(map[string][]api.PullRequest),
		loadingBuilds:                  make(map[string]bool),
		loadingReleases:                make(map[string]bool),
		loadingPullRequests:            make(map[string]bool),
		errors:                         make(map[string]error),
		watchedBuildDefinitions:        make(map[string]map[int]bool),
		watchedReleaseDefinitions:      make(map[string]map[int]bool),
		watchedPullRequests:            make(map[string]map[int]bool),
		knownBuildCompletion:           make(map[string]map[int]bool),
		knownReleaseCompletion:         make(map[string]map[int]bool),
		knownPullRequests:              make(map[string]map[int]api.PullRequest),
		buildSnapshotInitialized:       make(map[string]bool),
		releaseSnapshotInitialized:     make(map[string]bool),
		pullRequestSnapshotInitialized: make(map[string]bool),
		spinner:                        s,
		help:                           h,
		keys:                           DefaultKeyMap(),
		createPRBranchPushTimes:        make(map[string]time.Time),
		createPRSetAutoComplete:        true,
		createPRAutoApprove:            true,
		createPRDeleteSourceBranch:     true,
		createPRTransitionWorkItem:     true,
		createPRTitleAuto:              true,
		createPRDescriptionAuto:        true,
		buildLogLines:                  make(map[int][]string),
		buildLogLoadedUntil:            make(map[int]int),
		buildLogExhausted:              make(map[int]bool),
		buildLogWrapLines:              true,
		runPipelineBranchPush:          make(map[string]time.Time),
	}
}

func (m Model) createPRListMaxRows() int {
	available := m.height - 26
	if available <= 0 {
		return createPRMinListRows
	}

	rows := available / 3
	if rows < createPRMinListRows {
		return createPRMinListRows
	}
	if rows > createPRMaxListRows {
		return createPRMaxListRows
	}

	return rows
}

// newClientFromConfig creates an API client from the configuration
func newClientFromConfig(cfg *config.Config) *api.Client {
	return api.NewClient(api.ClientConfig{
		Organization:      cfg.AzureDevOps.Organization,
		BaseURL:           cfg.AzureDevOps.BaseURL,
		PAT:               cfg.AzureDevOps.PAT,
		RequestsPerSecond: cfg.RateLimiting.RequestsPerSecond,
		BurstSize:         cfg.RateLimiting.BurstSize,
	})
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	// Mark all projects as loading
	for _, p := range m.config.Projects {
		m.loadingBuilds[p.Name] = true
		m.loadingReleases[p.Name] = true
		m.loadingPullRequests[p.Name] = true
		m.watchedBuildDefinitions[p.Name] = make(map[int]bool)
		m.watchedReleaseDefinitions[p.Name] = make(map[int]bool)
		m.watchedPullRequests[p.Name] = make(map[int]bool)
		m.knownBuildCompletion[p.Name] = make(map[int]bool)
		m.knownReleaseCompletion[p.Name] = make(map[int]bool)
		m.knownPullRequests[p.Name] = make(map[int]api.PullRequest)
	}

	return tea.Batch(
		m.spinner.Tick,
		fetchAllData(m.client, m.config.Projects, m.config.Display.MaxItemsPerProject),
		refreshTicker(m.config.Display.RefreshInterval),
	)
}

// CurrentProject returns the current active project config
func (m Model) CurrentProject() config.ProjectConfig {
	if m.activeProject >= 0 && m.activeProject < len(m.config.Projects) {
		return m.config.Projects[m.activeProject]
	}
	return config.ProjectConfig{}
}

// CurrentBuilds returns the builds for the current project
func (m Model) CurrentBuilds() []api.Build {
	project := m.CurrentProject().Name
	return m.builds[project]
}

// CurrentReleases returns the releases for the current project
func (m Model) CurrentReleases() []api.Release {
	project := m.CurrentProject().Name
	return m.releases[project]
}

// CurrentPullRequests returns the pull requests for the current project
func (m Model) CurrentPullRequests() []api.PullRequest {
	project := m.CurrentProject().Name
	return m.pullRequests[project]
}

// IsLoading returns true if data is being loaded for the current project
func (m Model) IsLoading() bool {
	project := m.CurrentProject().Name
	switch m.activeTab {
	case TabBuilds:
		return m.loadingBuilds[project]
	case TabReleases:
		return m.loadingReleases[project]
	case TabPullRequests:
		return m.loadingPullRequests[project]
	}
	return false
}

// HasData returns true if data is available for the current project and tab
func (m Model) HasData() bool {
	project := m.CurrentProject().Name
	switch m.activeTab {
	case TabBuilds:
		builds, ok := m.builds[project]
		return ok && len(builds) > 0
	case TabReleases:
		releases, ok := m.releases[project]
		return ok && len(releases) > 0
	case TabPullRequests:
		pullRequests, ok := m.pullRequests[project]
		return ok && len(pullRequests) > 0
	}
	return false
}

// hasBuildData returns true if build data is available for the current project
func (m Model) hasBuildData() bool {
	project := m.CurrentProject().Name
	builds, ok := m.builds[project]
	return ok && len(builds) > 0
}

// hasReleaseData returns true if release data is available for the current project
func (m Model) hasReleaseData() bool {
	project := m.CurrentProject().Name
	releases, ok := m.releases[project]
	return ok && len(releases) > 0
}

// getBuildError returns the build error for the current project if any
func (m Model) getBuildError() error {
	project := m.CurrentProject().Name
	return m.errors[project+"-builds"]
}

// getReleaseError returns the release error for the current project if any
func (m Model) getReleaseError() error {
	project := m.CurrentProject().Name
	return m.errors[project+"-releases"]
}

// hasPullRequestData returns true if pull request data is available for the current project
func (m Model) hasPullRequestData() bool {
	project := m.CurrentProject().Name
	pullRequests, ok := m.pullRequests[project]
	return ok && len(pullRequests) > 0
}

// getPullRequestError returns the pull request error for the current project if any
func (m Model) getPullRequestError() error {
	project := m.CurrentProject().Name
	return m.errors[project+"-pullrequests"]
}

// getBranchFilterInfo returns branch filter info for the current project
func (m Model) getBranchFilterInfo() string {
	branches := m.CurrentProject().Branches
	if len(branches) == 0 {
		return "all"
	}
	return strings.Join(branches, ", ")
}

// CurrentError returns the error for the current project/tab if any
func (m Model) CurrentError() error {
	project := m.CurrentProject().Name
	key := project
	switch m.activeTab {
	case TabBuilds:
		key += "-builds"
	case TabReleases:
		key += "-releases"
	case TabPullRequests:
		key += "-pullrequests"
	}
	return m.errors[key]
}

// MaxRows returns the maximum number of rows that can be displayed
func (m Model) MaxRows() int {
	switch m.activeTab {
	case TabBuilds:
		return len(m.CurrentBuilds())
	case TabReleases:
		return len(m.CurrentReleases())
	case TabPullRequests:
		return len(m.CurrentPullRequests())
	}
	return 0
}

// toggleBuildDefinitionWatch toggles build definition watch for current selection.
func (m *Model) toggleBuildDefinitionWatch() {
	builds := m.CurrentBuilds()
	if m.selectedRow < 0 || m.selectedRow >= len(builds) {
		return
	}

	project := m.CurrentProject().Name
	definitionID := builds[m.selectedRow].Definition.ID

	if m.watchedBuildDefinitions[project] == nil {
		m.watchedBuildDefinitions[project] = make(map[int]bool)
	}

	if m.watchedBuildDefinitions[project][definitionID] {
		delete(m.watchedBuildDefinitions[project], definitionID)
		return
	}
	m.watchedBuildDefinitions[project][definitionID] = true
}

// toggleReleaseDefinitionWatch toggles release definition watch for current selection.
func (m *Model) toggleReleaseDefinitionWatch() {
	releases := m.CurrentReleases()
	if m.selectedRow < 0 || m.selectedRow >= len(releases) {
		return
	}

	project := m.CurrentProject().Name
	definitionID := releases[m.selectedRow].ReleaseDefinition.ID

	if m.watchedReleaseDefinitions[project] == nil {
		m.watchedReleaseDefinitions[project] = make(map[int]bool)
	}

	if m.watchedReleaseDefinitions[project][definitionID] {
		delete(m.watchedReleaseDefinitions[project], definitionID)
		return
	}
	m.watchedReleaseDefinitions[project][definitionID] = true
}

// isBuildDefinitionWatched returns true if build definition is watched in project.
func (m Model) isBuildDefinitionWatched(project string, definitionID int) bool {
	return m.watchedBuildDefinitions[project][definitionID]
}

// isReleaseDefinitionWatched returns true if release definition is watched in project.
func (m Model) isReleaseDefinitionWatched(project string, definitionID int) bool {
	return m.watchedReleaseDefinitions[project][definitionID]
}

// togglePullRequestWatch toggles pull request watch for current selection.
func (m *Model) togglePullRequestWatch() {
	pullRequests := m.CurrentPullRequests()
	if m.selectedRow < 0 || m.selectedRow >= len(pullRequests) {
		return
	}

	project := m.CurrentProject().Name
	pullRequestID := pullRequests[m.selectedRow].PullRequestID

	if m.watchedPullRequests[project] == nil {
		m.watchedPullRequests[project] = make(map[int]bool)
	}

	if m.watchedPullRequests[project][pullRequestID] {
		delete(m.watchedPullRequests[project], pullRequestID)
		return
	}
	m.watchedPullRequests[project][pullRequestID] = true
}

// isPullRequestWatched returns true if pull request is watched in project.
func (m Model) isPullRequestWatched(project string, pullRequestID int) bool {
	return m.watchedPullRequests[project][pullRequestID]
}

func (m *Model) resetCreatePRFlow() {
	m.creatingPullRequest = false
	m.createPRStep = PRCreateStepRepository
	m.createPRRepositories = nil
	m.createPRSelectedRepo = 0
	m.createPRBranches = nil
	m.createPRBranchPushTimes = make(map[string]time.Time)
	m.createPRHasMoreBranches = false
	m.createPRLoadAllBranches = false
	m.createPRSelectedSource = 0
	m.createPRTargetBranch = ""
	m.createPRSelectedTarget = 0
	m.createPROptionCursor = 0
	m.createPRSetAutoComplete = true
	m.createPRAutoApprove = true
	m.createPRDeleteSourceBranch = true
	m.createPRTransitionWorkItem = true
	m.createPRLoading = false
	m.createPRError = ""
	m.createPRTitle = ""
	m.createPRDescription = ""
	m.createPRTitleAuto = true
	m.createPRDescriptionAuto = true
	m.createPREditField = PRCreateEditFieldNone
}

func (m Model) createPRSourceOptionCount() int {
	count := len(m.createPRBranches)
	if m.createPRHasMoreBranches {
		count++
	}
	return count
}

func (m Model) isCreatePRLoadMoreSelected() bool {
	return m.createPRHasMoreBranches && m.createPRSelectedSource == len(m.createPRBranches)
}

func (m Model) selectedCreatePRSourceBranch() (string, bool) {
	if m.createPRSelectedSource < 0 || m.createPRSelectedSource >= len(m.createPRBranches) {
		return "", false
	}
	return m.createPRBranches[m.createPRSelectedSource], true
}

func trimRefPrefix(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

func (m *Model) resetBuildLogViewer() {
	m.showBuildLogViewer = false
	m.buildLogLoading = false
	m.buildLogError = ""
	m.buildLogProject = ""
	m.buildLogBuildID = 0
	m.buildLogBuildName = ""
	m.buildLogBranch = ""
	m.buildLogEntries = nil
	m.buildLogSelected = 0
	m.buildLogLines = make(map[int][]string)
	m.buildLogLoadedUntil = make(map[int]int)
	m.buildLogExhausted = make(map[int]bool)
	m.buildLogViewportTop = 0
}

func (m *Model) resetRunPipelineFlow() {
	m.runningPipeline = false
	m.runPipelineLoading = false
	m.runPipelineError = ""
	m.runPipelineBranches = nil
	m.runPipelineBranchPush = make(map[string]time.Time)
	m.runPipelineSelected = 0
	m.runPipelineDefinitionID = 0
	m.runPipelineDefinition = ""
	m.runPipelineRepositoryID = ""
	m.runPipelineRepository = ""
}

func (m Model) buildLogContentWidth() int {
	width := m.width - 10
	if width < 20 {
		return 20
	}
	return width
}

func (m Model) buildLogLineSegmentCount(line string, width int) int {
	if width <= 0 {
		return 1
	}
	runes := []rune(line)
	if len(runes) == 0 {
		return 1
	}
	segments := len(runes) / width
	if len(runes)%width != 0 {
		segments++
	}
	if segments < 1 {
		segments = 1
	}
	return segments
}

func (m Model) buildLogDisplayLineCount(lines []string) int {
	if len(lines) == 0 {
		return 0
	}
	if !m.buildLogWrapLines {
		return len(lines)
	}

	width := m.buildLogContentWidth()
	total := 0
	for _, line := range lines {
		total += m.buildLogLineSegmentCount(line, width)
	}

	return total
}
