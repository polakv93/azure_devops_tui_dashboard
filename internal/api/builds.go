package api

import (
	"strings"
	"time"
)

// GetBranchName extracts the branch name from the full ref
func (b *Build) GetBranchName() string {
	branch := b.SourceBranch
	branch = strings.TrimPrefix(branch, "refs/heads/")
	branch = strings.TrimPrefix(branch, "refs/pull/")
	return branch
}

// GetDuration returns the build duration
func (b *Build) GetDuration() time.Duration {
	if b.StartTime.IsZero() {
		return 0
	}

	endTime := b.FinishTime
	if endTime.IsZero() {
		endTime = time.Now()
	}

	return endTime.Sub(b.StartTime)
}

// GetStatusString returns a human-readable status string
func (b *Build) GetStatusString() string {
	if b.Status == BuildStatusCompleted {
		return string(b.Result)
	}
	return string(b.Status)
}

// IsRunning returns true if the build is currently running
func (b *Build) IsRunning() bool {
	return b.Status == BuildStatusInProgress
}

// IsCompleted returns true if the build has completed
func (b *Build) IsCompleted() bool {
	return b.Status == BuildStatusCompleted
}

// IsSuccessful returns true if the build completed successfully
func (b *Build) IsSuccessful() bool {
	return b.Status == BuildStatusCompleted && b.Result == BuildResultSucceeded
}

// IsFailed returns true if the build failed
func (b *Build) IsFailed() bool {
	return b.Status == BuildStatusCompleted && b.Result == BuildResultFailed
}
