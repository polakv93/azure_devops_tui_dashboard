# AGENTS.md

This document serves as a guide for agents working with the Azure DevOps TUI Dashboard repository. It includes build/test/lint commands and code style guidelines to ensure consistent and efficient development practices.

---
## Build, Lint, and Test Commands

Below are the commands to build, test, lint, and work with this repository. Use the `Makefile` as a quick reference for everyday tasks.

### Build Commands
- **Build the application**:
  ```bash
  make build  # Outputs binary to ./bin/azdo-tui
  ```

- **Build for all platforms**:
  ```bash
  make build-all  # Linux, Darwin, Windows
  ```

- **Run the application**:
  ```bash
  make run
  ./bin/azdo-tui --config configs/config.yaml
  ```

### Testing Commands
- **Run all tests**:
  ```bash
  make test
  go test -v ./...
  ```

- **Run a single package test**:
  ```bash
  go test -v ./internal/config
  ```

- **Run tests with a coverage report**:
  ```bash
  make test-coverage
  ```

### Formatting and Linting
- **Format code**:
  ```bash
  make fmt
  ```

- **Lint code** (requires `golangci-lint`):
  ```bash
  make lint
  ```

### Dependency Management
- **Update dependencies**:
  ```bash
  make deps
  ```

---
## Code Style Guidelines

### General Principles
Follow Go best practices relating to naming conventions, imports, error handling, and types. Details below summarize specific expectations for this repository.

### Naming Conventions
- **Packages**: Use short, lowercase names (e.g., `tui`, `api`).
- **Variables and Functions**: Use `camelCase` for variables and `PascalCase` for exported functions.
- **Constants**: Use `UPPER_SNAKE_CASE` for global constants.
- **Error Variables**: Prefix error variables with `err` (e.g., `errInvalidConfig`).

### Imports and Dependencies
- Organize imports into standard Go groupings:
  ```go
  import (
      "fmt"
      "net/http"

      "github.com/some/externalpkg"

      "internal/config"
      "internal/api"
  )
  ```
  Use `goimports` to auto-format imports whenever possible.

- Keep external dependencies minimal and well-documented.

### Formatting
- **Tools Required**:
  - `go fmt`: Aligns code to the standard.
  - `golangci-lint`: Static analysis for linting.

- **Comments**:
  - Use complete sentences for clarity.

### Error Handling
- Use the `errors` or `fmt` packages to wrap messages and provide context:
  ```go
  if err := doSomething(); err != nil {
      return fmt.Errorf("failed to doSomething: %w", err)
  }
  ```

- Avoid panics: Only use `panic` for truly unexpected states (e.g., programming bugs) and not user input.

### Types and Interfaces
- Use type aliases sparingly. For structs, favor explicit design tied to domain logic.
- Keep public structs distinct from internal representations (i.e., parsing JSON).
- When defining interfaces, limit them to the minimal set of methods to promote mockability and flexibility.

### Configuration Guidelines
- Store configuration in YAML files with `${VAR}` for environment variable expansion (e.g., `AZURE_DEVOPS_PAT`). Document required environment variables in `configs/`.
- Include example YAML configs for reference (`configs/config.example.yaml`).

---
## Bubble Tea Architecture Notes

This repository uses the Bubble Tea framework (Elm architecture) with the following structure:

- **Model**: Represents the application state.
- **Msg**: Defines various message types (e.g., API response messages, keypress events).
- **Update**: Handles state transitions based on messages.
- **View**: Renders the UI as a string.

Ensure changes align with this architecture and Bubble Tea conventions.

---
## Tips for Agents
- Always double-check `Makefile` for pre-defined tasks before manually running commands.
- Adhere to the code architecture when adding features (e.g., work in `internal/api` for API interaction).
- Use keybindings defined in `internal/tui/keys.go` for navigation testing and TUI features.
- Regularly run `make fmt` and `make lint` to keep code clean.

---

For further details, refer to `CLAUDE.md` or other documentation present in the repository.