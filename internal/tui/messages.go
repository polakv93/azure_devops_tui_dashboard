package tui

import (
	"time"

	"github.com/polakv93/azure_devops_tui_dashboard/internal/api"
)

// Message types for bubbletea

// BuildsLoadedMsg is sent when builds have been fetched
type BuildsLoadedMsg struct {
	Project string
	Builds  []api.Build
	Err     error
}

// ReleasesLoadedMsg is sent when releases have been fetched
type ReleasesLoadedMsg struct {
	Project  string
	Releases []api.Release
	Err      error
}

// PullRequestsLoadedMsg is sent when pull requests have been fetched
type PullRequestsLoadedMsg struct {
	Project      string
	PullRequests []api.PullRequest
	Err          error
}

// RefreshTickMsg is sent by the refresh ticker
type RefreshTickMsg struct{}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Err error
}

// OpenBrowserMsg requests opening a URL in the browser
type OpenBrowserMsg struct {
	URL string
}

// NotificationErrorMsg is sent when a desktop notification fails.
type NotificationErrorMsg struct {
	Err error
}

// CreatePRDataLoadedMsg is sent when repository/branch data for PR creation is ready.
type CreatePRDataLoadedMsg struct {
	Repositories []api.Repository
	Err          error
}

// CreatePRBranchesLoadedMsg is sent when branch data for selected repository is loaded.
type CreatePRBranchesLoadedMsg struct {
	RepositoryID string
	Branches     []string
	PushTimes    map[string]time.Time
	Err          error
}

// PullRequestCreatedMsg is sent when pull request creation and optional actions are complete.
type PullRequestCreatedMsg struct {
	PullRequest api.PullRequest
	Err         error
}

// CreatePRDefaultsLoadedMsg is sent when default title/description are prepared.
type CreatePRDefaultsLoadedMsg struct {
	RepositoryID string
	SourceBranch string
	Title        string
	Description  string
	Err          error
}

// BuildLogsLoadedMsg is sent when build logs list has been loaded.
type BuildLogsLoadedMsg struct {
	Project string
	BuildID int
	Logs    []api.BuildLog
	Err     error
}

// BuildLogChunkLoadedMsg is sent when a chunk of build log lines has been loaded.
type BuildLogChunkLoadedMsg struct {
	Project   string
	BuildID   int
	LogID     int
	StartLine int
	EndLine   int
	Lines     []string
	Err       error
}
