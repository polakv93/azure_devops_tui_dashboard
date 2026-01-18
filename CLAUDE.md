# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

```bash
# Build the application
make build                    # Output: bin/azdo-tui

# Run with config
make run                      # Uses configs/config.yaml
./bin/azdo-tui --config configs/config.yaml

# Run tests
make test                     # go test -v ./...
go test -v ./internal/config  # Run tests for a specific package

# Format and lint
make fmt                      # go fmt ./...
make lint                     # Requires golangci-lint

# Update dependencies
make deps                     # go mod tidy && go mod download
```

## Architecture

This is a Go TUI application for monitoring Azure DevOps builds and releases, built with the Bubble Tea framework (Elm architecture).

### Package Structure

- **cmd/azdo-tui/main.go** - Entry point; parses flags, loads config, starts TUI
- **internal/tui/** - Bubble Tea model and view logic
  - `model.go` - Main application state (Model struct)
  - `update.go` - Message handling and state transitions
  - `view.go` - UI rendering
  - `keys.go` - Keybinding definitions
  - `commands.go` - Tea commands (data fetching, browser open)
  - `messages.go` - Custom message types (BuildsLoadedMsg, RefreshTickMsg, etc.)
  - `components/` - Reusable table rendering components
- **internal/api/** - Azure DevOps API client
  - `client.go` - HTTP client with rate limiting and retry logic
  - `types.go` - Build, Release, and related structs matching Azure DevOps API
  - `builds.go`, `releases.go` - Helper methods on types
- **internal/config/** - YAML config loading with environment variable expansion
- **internal/styles/** - Lipgloss style definitions

### Key Patterns

**Bubble Tea Architecture**: The TUI follows the Elm pattern:
- `Model.Init()` starts spinners and initial data fetch
- `Model.Update()` handles messages (key presses, API responses, window resize)
- `Model.View()` renders the UI as a string

**API Client**: Uses token bucket rate limiting (`golang.org/x/time/rate`) and exponential backoff retry. Releases API uses a different base URL (`vsrm.dev.azure.com` vs `dev.azure.com`).

**Configuration**: YAML config supports `${VAR}` environment variable expansion. PAT should be provided via `AZURE_DEVOPS_PAT` environment variable.

### Navigation

The TUI has two tabs (Builds/Releases) and supports multiple projects. Key bindings are defined in `internal/tui/keys.go`: arrow keys/hjkl for navigation, Tab to switch views, Enter to open in browser, r to refresh, q to quit.
