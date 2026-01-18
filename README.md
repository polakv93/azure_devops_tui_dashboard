# Azure DevOps TUI Dashboard

A terminal-based dashboard for monitoring Azure DevOps builds and releases.

![Go](https://img.shields.io/badge/Go-1.25-blue)

## Features

- Monitor builds and releases across multiple Azure DevOps projects
- Switch between projects and views with keyboard navigation
- Auto-refresh at configurable intervals
- Open builds/releases directly in browser
- Rate limiting to respect Azure DevOps API limits

## Installation

### Using Go Install

```bash
go install github.com/polakv93/azure_devops_tui_dashboard/cmd/azdo-tui@latest
```

This will install the binary to your `$GOPATH/bin` directory (or `$HOME/go/bin` if `GOPATH` is not set).

### From Source

```bash
# Clone the repository
git clone https://github.com/polakv93/azure_devops_tui_dashboard.git
cd azure_devops_tui_dashboard

# Build
make build

# Or install to GOPATH/bin
make install
```

### Pre-built Binaries

Build for all platforms:
```bash
make build-all
```

This creates binaries in `bin/` for Linux, macOS (Intel + Apple Silicon), and Windows.

## Configuration

1. Copy the example config:
   ```bash
   cp configs/config.example.yaml configs/config.yaml
   ```

2. Edit `configs/config.yaml`:
   ```yaml
   azure_devops:
     organization: "your-org"
     pat: "${AZURE_DEVOPS_PAT}"  # Use env var for security

   projects:
     - name: "MyProject"
       build_definitions: [1, 5, 12]    # Optional: filter by IDs
       release_definitions: [2, 8]

     - name: "AnotherProject"
       build_definitions: []            # Empty = show all
       release_definitions: []

   display:
     refresh_interval: 30s
     max_items_per_project: 10
   ```

3. Create a Personal Access Token (PAT):
   - Go to `https://dev.azure.com/{org}/_usersSettings/tokens`
   - Create token with scopes: **Build (Read)**, **Release (Read)**
   - Set the environment variable:
     ```bash
     export AZURE_DEVOPS_PAT="your-token-here"
     ```

## Usage

```bash
# Run with config file
./bin/azdo-tui --config configs/config.yaml

# Or using make
make run
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` | Switch between Builds and Releases |
| `↑/k` | Move up |
| `↓/j` | Move down |
| `←/h` | Previous project |
| `→/l` | Next project |
| `Enter` | Open selected item in browser |
| `r` | Refresh data |
| `?` | Toggle help |
| `q` | Quit |

## Configuration Reference

| Setting | Default | Description |
|---------|---------|-------------|
| `azure_devops.organization` | - | Your Azure DevOps organization name |
| `azure_devops.base_url` | `https://dev.azure.com` | Azure DevOps base URL |
| `azure_devops.pat` | - | Personal Access Token |
| `display.refresh_interval` | `30s` | Auto-refresh interval |
| `display.max_items_per_project` | `10` | Max builds/releases to show |
| `display.date_format` | `2006-01-02 15:04` | Go time format |
| `rate_limiting.requests_per_second` | `5` | API rate limit |
| `rate_limiting.burst_size` | `10` | Rate limit burst size |

## License

MIT
