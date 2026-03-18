package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
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

// notifyDesktop shows a desktop notification.
func notifyDesktop(title, body string) tea.Cmd {
	return func() tea.Msg {
		if err := sendDesktopNotification(title, body); err != nil {
			return NotificationErrorMsg{Err: fmt.Errorf("failed to show desktop notification: %w", err)}
		}
		return nil
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

// fetchPullRequests creates a command to fetch pull requests for a project
func fetchPullRequests(client *api.Client, project config.ProjectConfig, maxItems int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		pullRequests, err := client.GetPullRequests(ctx, project.Name, project.Repositories, maxItems)
		if err != nil {
			return PullRequestsLoadedMsg{
				Project: project.Name,
				Err:     err,
			}
		}

		return PullRequestsLoadedMsg{
			Project:      project.Name,
			PullRequests: pullRequests,
		}
	}
}

// fetchAllData creates commands to fetch all builds, releases, and pull requests
func fetchAllData(client *api.Client, projects []config.ProjectConfig, maxItems int) tea.Cmd {
	var cmds []tea.Cmd

	for _, project := range projects {
		p := project // capture loop variable
		cmds = append(cmds, fetchBuilds(client, p, maxItems))
		cmds = append(cmds, fetchReleases(client, p, maxItems))
		cmds = append(cmds, fetchPullRequests(client, p, maxItems))
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

// loadCreatePRData fetches repositories for create PR flow.
func loadCreatePRData(client *api.Client, project config.ProjectConfig) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		repos, err := client.GetRepositories(ctx, project.Name, project.Repositories)
		if err != nil {
			return CreatePRDataLoadedMsg{Err: err}
		}

		sort.SliceStable(repos, func(i, j int) bool {
			return repos[i].Name < repos[j].Name
		})

		return CreatePRDataLoadedMsg{Repositories: repos}
	}
}

// loadCreatePRBranches fetches source branch candidates and recent push times.
func loadCreatePRBranches(client *api.Client, projectName, repositoryID string, activePRs []api.PullRequest, targetBranch string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		branches, err := client.GetRepositoryBranches(ctx, projectName, repositoryID)
		if err != nil {
			return CreatePRBranchesLoadedMsg{RepositoryID: repositoryID, Err: err}
		}

		pushTimes, err := client.GetRecentBranchPushTimes(ctx, projectName, repositoryID, 200)
		if err != nil {
			pushTimes = map[string]time.Time{}
		}

		normalizedTarget := trimRefPrefix(targetBranch)
		existing := make(map[string]bool)
		for _, pr := range activePRs {
			if pr.Repository.ID != repositoryID {
				continue
			}
			if trimRefPrefix(pr.TargetRefName) != normalizedTarget {
				continue
			}
			existing[trimRefPrefix(pr.SourceRefName)] = true
		}

		candidates := make([]string, 0, len(branches))
		for _, branch := range branches {
			if branch == normalizedTarget {
				continue
			}
			if existing[branch] {
				continue
			}
			candidates = append(candidates, branch)
		}

		sort.SliceStable(candidates, func(i, j int) bool {
			ti, okI := pushTimes[candidates[i]]
			tj, okJ := pushTimes[candidates[j]]
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

			return candidates[i] < candidates[j]
		})

		return CreatePRBranchesLoadedMsg{
			RepositoryID: repositoryID,
			Branches:     candidates,
			PushTimes:    pushTimes,
		}
	}
}

// createPullRequestWithOptions creates PR and optionally sets auto-complete and approval.
func createPullRequestWithOptions(
	client *api.Client,
	projectName string,
	repository api.Repository,
	sourceBranch string,
	targetBranch string,
	title string,
	description string,
	setAutoComplete bool,
	autoApprove bool,
	deleteSourceBranch bool,
	transitionWorkItems bool,
) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		sourceRef := sourceBranch
		if sourceRef != "" && !strings.HasPrefix(sourceRef, "refs/heads/") {
			sourceRef = "refs/heads/" + sourceRef
		}
		targetRef := targetBranch
		if targetRef != "" && !strings.HasPrefix(targetRef, "refs/heads/") {
			targetRef = "refs/heads/" + targetRef
		}

		request := api.PullRequestCreateRequest{
			SourceRefName: sourceRef,
			TargetRefName: targetRef,
			Title:         title,
			Description:   description,
		}

		createdPR, err := client.CreatePullRequest(ctx, projectName, repository.ID, request)
		if err != nil {
			return PullRequestCreatedMsg{Err: err}
		}

		if setAutoComplete {
			_, err = client.UpdatePullRequest(ctx, projectName, repository.ID, createdPR.PullRequestID, api.PullRequestUpdateRequest{
				AutoCompleteSetBy: &api.Identity{ID: createdPR.CreatedBy.ID},
				CompletionOptions: &api.PullRequestCompletionOptions{
					DeleteSourceBranch:  deleteSourceBranch,
					TransitionWorkItems: transitionWorkItems,
					MergeStrategy:       "noFastForward",
				},
			})
			if err != nil {
				return PullRequestCreatedMsg{Err: err}
			}
		}

		if autoApprove {
			if createdPR.CreatedBy.ID == "" {
				return PullRequestCreatedMsg{Err: fmt.Errorf("cannot auto-approve: missing current user id in API response")}
			}
			_, err = client.SetPullRequestReviewerVote(ctx, projectName, repository.ID, createdPR.PullRequestID, createdPR.CreatedBy.ID, 10)
			if err != nil {
				return PullRequestCreatedMsg{Err: err}
			}
		}

		return PullRequestCreatedMsg{PullRequest: createdPR}
	}
}

// loadCreatePRDefaults gets default title/description from latest commit message.
func loadCreatePRDefaults(client *api.Client, projectName, repositoryID, sourceBranch string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		message, err := client.GetLatestCommitMessage(ctx, projectName, repositoryID, sourceBranch)
		if err != nil {
			return CreatePRDefaultsLoadedMsg{RepositoryID: repositoryID, SourceBranch: sourceBranch, Err: err}
		}

		title := message
		description := ""
		if title == "" {
			title = sourceBranch
		}

		if idx := strings.Index(title, "\n"); idx >= 0 {
			description = strings.TrimSpace(title[idx+1:])
			title = strings.TrimSpace(title[:idx])
		}

		if title == "" {
			title = sourceBranch
		}

		return CreatePRDefaultsLoadedMsg{
			RepositoryID: repositoryID,
			SourceBranch: sourceBranch,
			Title:        title,
			Description:  description,
		}
	}
}
