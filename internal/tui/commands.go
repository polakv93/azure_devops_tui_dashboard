package tui

import (
	"context"
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/api"
	"github.com/polakv93/azure_devops_tui_dashboard/internal/config"
)

// fetchBuilds creates a command to fetch builds for a project
func fetchBuilds(client *api.Client, project config.ProjectConfig, maxItems int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		builds, err := client.GetBuilds(ctx, project.Name, project.BuildDefinitions, project.Branches, maxItems)
		if err != nil {
			return BuildsLoadedMsg{
				Project: project.Name,
				Err:     err,
			}
		}

		return BuildsLoadedMsg{
			Project: project.Name,
			Builds:  builds,
		}
	}
}

// fetchReleases creates a command to fetch releases for a project
func fetchReleases(client *api.Client, project config.ProjectConfig, maxItems int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		releases, err := client.GetReleases(ctx, project.Name, project.ReleaseDefinitions, maxItems)
		if err != nil {
			return ReleasesLoadedMsg{
				Project: project.Name,
				Err:     err,
			}
		}

		return ReleasesLoadedMsg{
			Project:  project.Name,
			Releases: releases,
		}
	}
}

// fetchAllData creates commands to fetch all builds and releases
func fetchAllData(client *api.Client, projects []config.ProjectConfig, maxItems int) tea.Cmd {
	var cmds []tea.Cmd

	for _, project := range projects {
		p := project // capture loop variable
		cmds = append(cmds, fetchBuilds(client, p, maxItems))
		cmds = append(cmds, fetchReleases(client, p, maxItems))
	}

	return tea.Batch(cmds...)
}

// refreshTicker creates a command that ticks at the specified interval
func refreshTicker(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return RefreshTickMsg{}
	})
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		case "darwin":
			cmd = exec.Command("open", url)
		default: // Linux and others
			cmd = exec.Command("xdg-open", url)
		}

		_ = cmd.Start()
		return nil
	}
}
