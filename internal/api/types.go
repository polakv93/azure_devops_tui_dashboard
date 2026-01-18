package api

import "time"

// BuildStatus represents the status of a build
type BuildStatus string

const (
	BuildStatusNone       BuildStatus = "none"
	BuildStatusInProgress BuildStatus = "inProgress"
	BuildStatusCompleted  BuildStatus = "completed"
	BuildStatusCancelling BuildStatus = "cancelling"
	BuildStatusPostponed  BuildStatus = "postponed"
	BuildStatusNotStarted BuildStatus = "notStarted"
)

// BuildResult represents the result of a completed build
type BuildResult string

const (
	BuildResultNone               BuildResult = "none"
	BuildResultSucceeded          BuildResult = "succeeded"
	BuildResultPartiallySucceeded BuildResult = "partiallySucceeded"
	BuildResultFailed             BuildResult = "failed"
	BuildResultCanceled           BuildResult = "canceled"
)

// Build represents an Azure DevOps build
type Build struct {
	ID            int             `json:"id"`
	BuildNumber   string          `json:"buildNumber"`
	Status        BuildStatus     `json:"status"`
	Result        BuildResult     `json:"result"`
	QueueTime     time.Time       `json:"queueTime"`
	StartTime     time.Time       `json:"startTime"`
	FinishTime    time.Time       `json:"finishTime"`
	Definition    BuildDefinition `json:"definition"`
	SourceBranch  string          `json:"sourceBranch"`
	SourceVersion string          `json:"sourceVersion"`
	RequestedFor  Identity        `json:"requestedFor"`
	Project       TeamProject     `json:"project"`
	Links         BuildLinks      `json:"_links"`
}

// BuildDefinition represents a build pipeline definition
type BuildDefinition struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BuildLinks contains links related to a build
type BuildLinks struct {
	Web Link `json:"web"`
}

// Link represents a hyperlink
type Link struct {
	Href string `json:"href"`
}

// BuildsResponse represents the API response for builds
type BuildsResponse struct {
	Count int     `json:"count"`
	Value []Build `json:"value"`
}

// ReleaseStatus represents the status of a release
type ReleaseStatus string

const (
	ReleaseStatusActive    ReleaseStatus = "active"
	ReleaseStatusDraft     ReleaseStatus = "draft"
	ReleaseStatusAbandoned ReleaseStatus = "abandoned"
	ReleaseStatusUndefined ReleaseStatus = "undefined"
)

// EnvironmentStatus represents the status of a release environment
type EnvironmentStatus string

const (
	EnvironmentStatusNotStarted         EnvironmentStatus = "notStarted"
	EnvironmentStatusInProgress         EnvironmentStatus = "inProgress"
	EnvironmentStatusSucceeded          EnvironmentStatus = "succeeded"
	EnvironmentStatusCanceled           EnvironmentStatus = "canceled"
	EnvironmentStatusRejected           EnvironmentStatus = "rejected"
	EnvironmentStatusQueued             EnvironmentStatus = "queued"
	EnvironmentStatusScheduled          EnvironmentStatus = "scheduled"
	EnvironmentStatusPartiallySucceeded EnvironmentStatus = "partiallySucceeded"
	EnvironmentStatusUndefined          EnvironmentStatus = "undefined"
)

// Release represents an Azure DevOps release
type Release struct {
	ID                int                  `json:"id"`
	Name              string               `json:"name"`
	Status            ReleaseStatus        `json:"status"`
	CreatedOn         time.Time            `json:"createdOn"`
	ModifiedOn        time.Time            `json:"modifiedOn"`
	ReleaseDefinition ReleaseDefinition    `json:"releaseDefinition"`
	Environments      []ReleaseEnvironment `json:"environments"`
	CreatedBy         Identity             `json:"createdBy"`
	ProjectReference  ProjectReference     `json:"projectReference"`
	Links             ReleaseLinks         `json:"_links"`
}

// ReleaseDefinition represents a release pipeline definition
type ReleaseDefinition struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ReleaseEnvironment represents an environment/stage in a release
type ReleaseEnvironment struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Status      EnvironmentStatus `json:"status"`
	DeploySteps []DeployStep      `json:"deploySteps"`
}

// DeployStep represents a deployment attempt in an environment
type DeployStep struct {
	ID              int               `json:"id"`
	Status          EnvironmentStatus `json:"status"`
	OperationStatus string            `json:"operationStatus"`
}

// ReleaseLinks contains links related to a release
type ReleaseLinks struct {
	Web Link `json:"web"`
}

// ReleasesResponse represents the API response for releases
type ReleasesResponse struct {
	Count int       `json:"count"`
	Value []Release `json:"value"`
}

// Identity represents a user identity
type Identity struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
}

// TeamProject represents a project
type TeamProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ProjectReference represents a project reference in releases
type ProjectReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
