package tui

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
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

// loadCreatePRBranchesRecent fetches recent branch candidates (fast mode) and push times.
func loadCreatePRBranchesRecent(client *api.Client, projectName, repositoryID string, activePRs []api.PullRequest, targetBranch string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		pushTimes, err := client.GetRecentBranchPushTimes(ctx, projectName, repositoryID, 400)
		if err != nil {
			pushTimes = map[string]time.Time{}
		}

		recentBranches := make([]string, 0, len(pushTimes))
		for branch := range pushTimes {
			recentBranches = append(recentBranches, branch)
		}
		candidates := createPRSourceCandidates(recentBranches, pushTimes, activePRs, repositoryID, targetBranch)
		if len(candidates) > createPRRecentBranchLimit {
			candidates = candidates[:createPRRecentBranchLimit]
		}

		return CreatePRBranchesLoadedMsg{
			RepositoryID: repositoryID,
			Branches:     candidates,
			PushTimes:    pushTimes,
			HasMore:      true,
		}
	}
}

// loadCreatePRBranchesAll fetches all branch candidates and recent push times.
func loadCreatePRBranchesAll(client *api.Client, projectName, repositoryID string, activePRs []api.PullRequest, targetBranch string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		branches, err := client.GetRepositoryBranches(ctx, projectName, repositoryID)
		if err != nil {
			return CreatePRBranchesLoadedMsg{RepositoryID: repositoryID, Err: err}
		}

		pushTimes, err := client.GetRecentBranchPushTimes(ctx, projectName, repositoryID, 400)
		if err != nil {
			pushTimes = map[string]time.Time{}
		}

		candidates := createPRSourceCandidates(branches, pushTimes, activePRs, repositoryID, targetBranch)

		return CreatePRBranchesLoadedMsg{
			RepositoryID: repositoryID,
			Branches:     candidates,
			PushTimes:    pushTimes,
			HasMore:      false,
		}
	}
}

func createPRSourceCandidates(branches []string, pushTimes map[string]time.Time, activePRs []api.PullRequest, repositoryID, targetBranch string) []string {
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
	seen := make(map[string]bool)
	for _, branch := range branches {
		if seen[branch] {
			continue
		}
		seen[branch] = true
		if branch == normalizedTarget {
			continue
		}
		if existing[branch] {
			continue
		}
		candidates = append(candidates, branch)
	}

	sortBranchesByPushTime(candidates, pushTimes)
	return candidates
}

func sortBranchesByPushTime(branches []string, pushTimes map[string]time.Time) {
	sort.SliceStable(branches, func(i, j int) bool {
		ti, okI := pushTimes[branches[i]]
		tj, okJ := pushTimes[branches[j]]
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

		return branches[i] < branches[j]
	})
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

		latestCommitMessage, err := client.GetLatestCommitMessage(ctx, projectName, repository.ID, sourceBranch)
		if err == nil {
			workItemIDs := extractWorkItemIDsFromCommitMessage(latestCommitMessage)
			if len(workItemIDs) > 0 {
				request.WorkItemRefs = make([]api.ResourceRef, 0, len(workItemIDs))
				for _, id := range workItemIDs {
					request.WorkItemRefs = append(request.WorkItemRefs, api.ResourceRef{ID: id})
				}
			}
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
					MergeStrategy:       "rebase",
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

var workItemIDPattern = regexp.MustCompile(`#(\d+)(?:[^\w]|$)`)

func extractWorkItemIDsFromCommitMessage(message string) []string {
	matches := workItemIDPattern.FindAllStringSubmatch(message, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		id := match[1]
		if seen[id] {
			continue
		}

		seen[id] = true
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil
	}

	return ids
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

// loadBuildLogs fetches all logs descriptors for a specific build.
func loadBuildLogs(client *api.Client, project string, buildID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logs, err := client.GetBuildLogs(ctx, project, buildID)
		if err != nil {
			return BuildLogsLoadedMsg{Project: project, BuildID: buildID, Err: err}
		}

		sort.SliceStable(logs, func(i, j int) bool {
			return logs[i].ID < logs[j].ID
		})

		return BuildLogsLoadedMsg{Project: project, BuildID: buildID, Logs: logs}
	}
}

// loadBuildLogChunk fetches a selected line range from a build log.
func loadBuildLogChunk(client *api.Client, project string, buildID, logID, startLine, endLine int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		lines, err := client.GetBuildLogLines(ctx, project, buildID, logID, startLine, endLine)
		if err != nil {
			return BuildLogChunkLoadedMsg{
				Project:   project,
				BuildID:   buildID,
				LogID:     logID,
				StartLine: startLine,
				EndLine:   endLine,
				Err:       err,
			}
		}

		return BuildLogChunkLoadedMsg{
			Project:   project,
			BuildID:   buildID,
			LogID:     logID,
			StartLine: startLine,
			EndLine:   endLine,
			Lines:     lines,
		}
	}
}

// loadRunPipelineData fetches repository and branch options for selected build definition.
func loadRunPipelineData(client *api.Client, projectName string, definitionID int, definitionName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		definition, err := client.GetBuildDefinition(ctx, projectName, definitionID)
		if err != nil {
			return RunPipelineDataLoadedMsg{DefinitionID: definitionID, DefinitionName: definitionName, Err: err}
		}

		repoID := definition.Repository.ID
		repoName := definition.Repository.Name
		if repoID == "" {
			return RunPipelineDataLoadedMsg{
				DefinitionID:   definitionID,
				DefinitionName: definitionName,
				Err:            fmt.Errorf("selected pipeline has no Git repository configured"),
			}
		}

		branches, err := client.GetRepositoryBranches(ctx, projectName, repoID)
		if err != nil {
			return RunPipelineDataLoadedMsg{DefinitionID: definitionID, DefinitionName: definitionName, RepositoryID: repoID, RepositoryName: repoName, Err: err}
		}

		pushTimes, err := client.GetRecentBranchPushTimes(ctx, projectName, repoID, 200)
		if err != nil {
			pushTimes = map[string]time.Time{}
		}

		sort.SliceStable(branches, func(i, j int) bool {
			ti, okI := pushTimes[branches[i]]
			tj, okJ := pushTimes[branches[j]]
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

			return branches[i] < branches[j]
		})

		return RunPipelineDataLoadedMsg{
			DefinitionID:   definitionID,
			DefinitionName: definitionName,
			RepositoryID:   repoID,
			RepositoryName: repoName,
			Branches:       branches,
			PushTimes:      pushTimes,
		}
	}
}

// queuePipelineRun queues selected pipeline for a selected branch.
func queuePipelineRun(client *api.Client, projectName string, definitionID int, definitionName, branch string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		sourceRef := branch
		if sourceRef != "" && !strings.HasPrefix(sourceRef, "refs/heads/") {
			sourceRef = "refs/heads/" + sourceRef
		}

		build, err := client.QueueBuild(ctx, projectName, definitionID, sourceRef)
		if err != nil {
			return PipelineQueuedMsg{
				Project:        projectName,
				DefinitionID:   definitionID,
				DefinitionName: definitionName,
				Branch:         branch,
				Err:            err,
			}
		}

		return PipelineQueuedMsg{
			Project:        projectName,
			DefinitionID:   definitionID,
			DefinitionName: definitionName,
			Branch:         branch,
			BuildID:        build.ID,
		}
	}
}
