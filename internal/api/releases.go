package api

// GetOverallStatus determines the overall status of a release based on its environments
func (r *Release) GetOverallStatus() EnvironmentStatus {
	if len(r.Environments) == 0 {
		return EnvironmentStatusUndefined
	}

	hasInProgress := false
	hasFailed := false
	hasQueued := false
	allSucceeded := true

	for _, env := range r.Environments {
		switch env.Status {
		case EnvironmentStatusInProgress:
			hasInProgress = true
			allSucceeded = false
		case EnvironmentStatusRejected, EnvironmentStatusCanceled:
			hasFailed = true
			allSucceeded = false
		case EnvironmentStatusQueued, EnvironmentStatusScheduled:
			hasQueued = true
			allSucceeded = false
		case EnvironmentStatusNotStarted:
			allSucceeded = false
		case EnvironmentStatusPartiallySucceeded:
			allSucceeded = false
		case EnvironmentStatusSucceeded:
			// This one succeeded
		default:
			allSucceeded = false
		}
	}

	if hasInProgress {
		return EnvironmentStatusInProgress
	}
	if hasFailed {
		return EnvironmentStatusRejected
	}
	if hasQueued {
		return EnvironmentStatusQueued
	}
	if allSucceeded {
		return EnvironmentStatusSucceeded
	}

	return EnvironmentStatusNotStarted
}

// GetEnvironmentSummary returns a summary string of environment statuses
func (r *Release) GetEnvironmentSummary() string {
	if len(r.Environments) == 0 {
		return "-"
	}

	summary := ""
	for i, env := range r.Environments {
		if i > 0 {
			summary += " → "
		}

		status := getStatusIcon(env.Status)
		summary += env.Name + ":" + status
	}

	return summary
}

func getStatusIcon(status EnvironmentStatus) string {
	switch status {
	case EnvironmentStatusSucceeded:
		return "✓"
	case EnvironmentStatusRejected, EnvironmentStatusCanceled:
		return "✗"
	case EnvironmentStatusInProgress:
		return "●"
	case EnvironmentStatusQueued, EnvironmentStatusScheduled:
		return "○"
	case EnvironmentStatusPartiallySucceeded:
		return "◐"
	default:
		return "-"
	}
}

// IsActive returns true if the release is active
func (r *Release) IsActive() bool {
	return r.Status == ReleaseStatusActive
}

// HasFailedEnvironment returns true if any environment has failed
func (r *Release) HasFailedEnvironment() bool {
	for _, env := range r.Environments {
		if env.Status == EnvironmentStatusRejected || env.Status == EnvironmentStatusCanceled {
			return true
		}
	}
	return false
}

// HasInProgressEnvironment returns true if any environment is in progress
func (r *Release) HasInProgressEnvironment() bool {
	for _, env := range r.Environments {
		if env.Status == EnvironmentStatusInProgress {
			return true
		}
	}
	return false
}
