package tui

import (
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
