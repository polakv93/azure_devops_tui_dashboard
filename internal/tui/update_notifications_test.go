package tui

import (
	"testing"

	"github.com/polakv93/azure_devops_tui_dashboard/internal/api"
)

func TestPullRequestNotificationCmds_NoNotificationsOnInitialSnapshot(t *testing.T) {
	m := Model{
		watchedPullRequests:            map[string]map[int]bool{"proj": {1: true}},
		knownPullRequests:              map[string]map[int]api.PullRequest{"proj": {}},
		pullRequestSnapshotInitialized: map[string]bool{"proj": false},
	}

	cmds := m.pullRequestNotificationCmds("proj", []api.PullRequest{makePR(1, "PR-1", "", []int{0})})
	if len(cmds) != 0 {
		t.Fatalf("expected 0 notifications on initial snapshot, got %d", len(cmds))
	}
}

func TestPullRequestNotificationCmds_ImportantTransitions(t *testing.T) {
	tests := []struct {
		name     string
		previous api.PullRequest
		current  api.PullRequest
		expected int
	}{
		{
			name:     "conflict detected",
			previous: makePR(1, "PR-1", "", []int{0}),
			current:  makePR(1, "PR-1", "conflicts", []int{0}),
			expected: 1,
		},
		{
			name:     "rejection detected",
			previous: makePR(1, "PR-1", "", []int{0}),
			current:  makePR(1, "PR-1", "", []int{-10}),
			expected: 1,
		},
		{
			name:     "fully approved detected",
			previous: makePR(1, "PR-1", "", []int{0}),
			current:  makePR(1, "PR-1", "", []int{10}),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				watchedPullRequests:            map[string]map[int]bool{"proj": {1: true}},
				knownPullRequests:              map[string]map[int]api.PullRequest{"proj": {1: tt.previous}},
				pullRequestSnapshotInitialized: map[string]bool{"proj": true},
			}

			cmds := m.pullRequestNotificationCmds("proj", []api.PullRequest{tt.current})
			if len(cmds) != tt.expected {
				t.Fatalf("expected %d notifications, got %d", tt.expected, len(cmds))
			}
		})
	}
}

func TestPullRequestNotificationCmds_ClosedWhenMissingFromActiveList(t *testing.T) {
	m := Model{
		watchedPullRequests:            map[string]map[int]bool{"proj": {1: true}},
		knownPullRequests:              map[string]map[int]api.PullRequest{"proj": {1: makePR(1, "PR-1", "", []int{0})}},
		pullRequestSnapshotInitialized: map[string]bool{"proj": true},
	}

	cmds := m.pullRequestNotificationCmds("proj", []api.PullRequest{})
	if len(cmds) != 1 {
		t.Fatalf("expected 1 notification for closed PR, got %d", len(cmds))
	}
}

func makePR(id int, title, mergeStatus string, reviewerVotes []int) api.PullRequest {
	reviewers := make([]api.Reviewer, 0, len(reviewerVotes))
	for i, vote := range reviewerVotes {
		reviewers = append(reviewers, api.Reviewer{ID: string(rune('a' + i)), Vote: vote})
	}

	return api.PullRequest{
		PullRequestID: id,
		Title:         title,
		MergeStatus:   mergeStatus,
		Status:        api.PullRequestStatusActive,
		Reviewers:     reviewers,
	}
}
