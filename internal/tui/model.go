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
	builds   map[string][]api.Build   // project name -> builds
	releases map[string][]api.Release // project name -> releases

	// Loading states
	loadingBuilds   map[string]bool
	loadingReleases map[string]bool

	// Errors
	errors map[string]error

	// Last refresh time
	lastRefresh time.Time

	// Components
	spinner spinner.Model
	help    help.Model
	keys    KeyMap
}

// NewModel creates a new Model with the given configuration
func NewModel(cfg *config.Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))

	h := help.New()
	h.ShowAll = false

	return Model{
		config:          cfg,
		client:          newClientFromConfig(cfg),
		activeTab:       TabBuilds,
		activeProject:   0,
		selectedRow:     0,
		builds:          make(map[string][]api.Build),
		releases:        make(map[string][]api.Release),
		loadingBuilds:   make(map[string]bool),
		loadingReleases: make(map[string]bool),
		errors:          make(map[string]error),
		spinner:         s,
		help:            h,
		keys:            DefaultKeyMap(),
	}
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

// IsLoading returns true if data is being loaded for the current project
func (m Model) IsLoading() bool {
	project := m.CurrentProject().Name
	switch m.activeTab {
	case TabBuilds:
		return m.loadingBuilds[project]
	case TabReleases:
		return m.loadingReleases[project]
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
	if m.activeTab == TabBuilds {
		key += "-builds"
	} else {
		key += "-releases"
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
	}
	return 0
}
