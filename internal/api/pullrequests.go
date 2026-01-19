package api

import (
	"fmt"
	"strings"
)

// GetSourceBranch extracts the source branch name from the full ref
func (pr *PullRequest) GetSourceBranch() string {
	return strings.TrimPrefix(pr.SourceRefName, "refs/heads/")
}

// GetTargetBranch extracts the target branch name from the full ref
func (pr *PullRequest) GetTargetBranch() string {
	return strings.TrimPrefix(pr.TargetRefName, "refs/heads/")
}

// GetBranchSummary returns a summary of source -> target branches
func (pr *PullRequest) GetBranchSummary() string {
	return fmt.Sprintf("%s -> %s", pr.GetSourceBranch(), pr.GetTargetBranch())
}

// GetReviewerSummary returns a summary of reviewer votes
func (pr *PullRequest) GetReviewerSummary() string {
	if len(pr.Reviewers) == 0 {
		return "-"
	}

	approved := 0
	rejected := 0
	waiting := 0
	noVote := 0

	for _, r := range pr.Reviewers {
		switch {
		case r.Vote >= 10:
			approved++
		case r.Vote == 5:
			approved++ // Approved with suggestions counts as approved
		case r.Vote <= -10:
			rejected++
		case r.Vote == -5:
			waiting++
		default:
			noVote++
		}
	}

	total := len(pr.Reviewers)
	if rejected > 0 {
		return fmt.Sprintf("%d/%d (✗%d)", approved, total, rejected)
	}
	if waiting > 0 {
		return fmt.Sprintf("%d/%d (○%d)", approved, total, waiting)
	}
	if approved == total {
		return fmt.Sprintf("%d/%d ✓", approved, total)
	}
	return fmt.Sprintf("%d/%d", approved, total)
}

// GetStatusDisplay returns a display string for the PR status
func (pr *PullRequest) GetStatusDisplay() string {
	if pr.IsDraft {
		return "draft"
	}
	if pr.MergeStatus == "conflicts" {
		return "conflicts"
	}
	return string(pr.Status)
}

// IsActive returns true if the pull request is active
func (pr *PullRequest) IsActive() bool {
	return pr.Status == PullRequestStatusActive
}

// IsCompleted returns true if the pull request is completed (merged)
func (pr *PullRequest) IsCompleted() bool {
	return pr.Status == PullRequestStatusCompleted
}

// IsAbandoned returns true if the pull request is abandoned
func (pr *PullRequest) IsAbandoned() bool {
	return pr.Status == PullRequestStatusAbandoned
}

// HasConflicts returns true if the pull request has merge conflicts
func (pr *PullRequest) HasConflicts() bool {
	return pr.MergeStatus == "conflicts"
}

// IsApproved returns true if all reviewers have approved
func (pr *PullRequest) IsApproved() bool {
	if len(pr.Reviewers) == 0 {
		return false
	}
	for _, r := range pr.Reviewers {
		if r.Vote < 5 { // Less than "approved with suggestions"
			return false
		}
	}
	return true
}

// HasRejections returns true if any reviewer has rejected
func (pr *PullRequest) HasRejections() bool {
	for _, r := range pr.Reviewers {
		if r.Vote <= -10 {
			return true
		}
	}
	return false
}
