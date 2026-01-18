package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/polakj/azure_devops_tui_dashboard/internal/config"
	"github.com/polakj/azure_devops_tui_dashboard/internal/tui"
)

var (
	version = "dev"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file (required)")
	configPathShort := flag.String("c", "", "Path to configuration file (shorthand)")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("azdo-tui version %s\n", version)
		os.Exit(0)
	}

	// Determine config path
	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = *configPathShort
	}

	if cfgPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --config or -c flag is required")
		fmt.Fprintln(os.Stderr, "Usage: azdo-tui --config <path-to-config.yaml>")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Create and run the TUI
	model := tui.NewModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}
