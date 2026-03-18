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

// BuildTimelineRecordState represents the state of a timeline record
type BuildTimelineRecordState string

const (
	BuildTimelineRecordStatePending    BuildTimelineRecordState = "pending"
	BuildTimelineRecordStateInProgress BuildTimelineRecordState = "inProgress"
	BuildTimelineRecordStateCompleted  BuildTimelineRecordState = "completed"
)

// BuildTimelineRecordResult represents the result of a completed timeline record
type BuildTimelineRecordResult string

const (
	BuildTimelineRecordResultSucceeded           BuildTimelineRecordResult = "succeeded"
	BuildTimelineRecordResultSucceededWithIssues BuildTimelineRecordResult = "succeededWithIssues"
	BuildTimelineRecordResultFailed              BuildTimelineRecordResult = "failed"
	BuildTimelineRecordResultCanceled            BuildTimelineRecordResult = "canceled"
	BuildTimelineRecordResultSkipped             BuildTimelineRecordResult = "skipped"
	BuildTimelineRecordResultAbandoned           BuildTimelineRecordResult = "abandoned"
)

// BuildTimelineRecord represents a record in the build timeline (stage, phase, job, task)
type BuildTimelineRecord struct {
	ID       string                    `json:"id"`
	ParentID string                    `json:"parentId"`
	Name     string                    `json:"name"`
	Type     string                    `json:"type"` // "Stage", "Phase", "Job", "Task", "Checkpoint"
	Order    int                       `json:"order"`
	State    BuildTimelineRecordState  `json:"state"`
	Result   BuildTimelineRecordResult `json:"result"`
}

// BuildTimelineResponse represents the API response for a build timeline
type BuildTimelineResponse struct {
	Records []BuildTimelineRecord `json:"records"`
}

// Build represents an Azure DevOps build
type Build struct {
	ID            int                   `json:"id"`
	BuildNumber   string                `json:"buildNumber"`
	Status        BuildStatus           `json:"status"`
	Result        BuildResult           `json:"result"`
	QueueTime     time.Time             `json:"queueTime"`
	StartTime     time.Time             `json:"startTime"`
	FinishTime    time.Time             `json:"finishTime"`
	Definition    BuildDefinition       `json:"definition"`
	SourceBranch  string                `json:"sourceBranch"`
	SourceVersion string                `json:"sourceVersion"`
	RequestedFor  Identity              `json:"requestedFor"`
	Project       TeamProject           `json:"project"`
	Links         BuildLinks            `json:"_links"`
	Stages        []BuildTimelineRecord `json:"-"` // Populated separately via timeline API
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

// BuildLog represents a build log descriptor.
type BuildLog struct {
	ID            int       `json:"id"`
	Type          string    `json:"type"`
	URL           string    `json:"url"`
	CreatedOn     time.Time `json:"createdOn"`
	LastChangedOn time.Time `json:"lastChangedOn"`
	LineCount     int       `json:"lineCount"`
}

// BuildLogsResponse represents the API response for build logs.
type BuildLogsResponse struct {
	Count int        `json:"count"`
	Value []BuildLog `json:"value"`
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
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
}

// Repository represents an Azure DevOps Git repository.
type Repository struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	DefaultBranch string      `json:"defaultBranch"`
	Project       TeamProject `json:"project"`
}

// RepositoriesResponse represents the API response for repositories.
type RepositoriesResponse struct {
	Count int          `json:"count"`
	Value []Repository `json:"value"`
}

// GitRef represents a git reference/branch.
type GitRef struct {
	Name string `json:"name"`
}

// GitRefsResponse represents the API response for git refs.
type GitRefsResponse struct {
	Count int      `json:"count"`
	Value []GitRef `json:"value"`
}

// PushRefUpdate represents a branch update in a push.
type PushRefUpdate struct {
	Name string `json:"name"`
}

// Push represents a git push event.
type Push struct {
	Date       time.Time       `json:"date"`
	RefUpdates []PushRefUpdate `json:"refUpdates"`
}

// PushesResponse represents the API response for pushes.
type PushesResponse struct {
	Count int    `json:"count"`
	Value []Push `json:"value"`
}

// Commit represents a git commit with message.
type Commit struct {
	Comment string `json:"comment"`
}

// CommitsResponse represents the API response for commits.
type CommitsResponse struct {
	Count int      `json:"count"`
	Value []Commit `json:"value"`
}

// PullRequestCompletionOptions controls pull request completion behavior.
type PullRequestCompletionOptions struct {
	DeleteSourceBranch  bool   `json:"deleteSourceBranch,omitempty"`
	TransitionWorkItems bool   `json:"transitionWorkItems,omitempty"`
	MergeStrategy       string `json:"mergeStrategy,omitempty"`
}

// PullRequestCreateRequest is payload for creating a pull request.
type PullRequestCreateRequest struct {
	SourceRefName string     `json:"sourceRefName"`
	TargetRefName string     `json:"targetRefName"`
	Title         string     `json:"title"`
	Description   string     `json:"description,omitempty"`
	Reviewers     []Reviewer `json:"reviewers,omitempty"`
	IsDraft       bool       `json:"isDraft,omitempty"`
}

// PullRequestUpdateRequest is payload for updating a pull request.
type PullRequestUpdateRequest struct {
	AutoCompleteSetBy *Identity                     `json:"autoCompleteSetBy,omitempty"`
	CompletionOptions *PullRequestCompletionOptions `json:"completionOptions,omitempty"`
}

// PullRequestReviewerVoteRequest is payload for setting reviewer vote.
type PullRequestReviewerVoteRequest struct {
	ID   string `json:"id"`
	Vote int    `json:"vote"`
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

// PullRequestStatus represents the status of a pull request
type PullRequestStatus string

const (
	PullRequestStatusActive    PullRequestStatus = "active"
	PullRequestStatusAbandoned PullRequestStatus = "abandoned"
	PullRequestStatusCompleted PullRequestStatus = "completed"
	PullRequestStatusNotSet    PullRequestStatus = "notSet"
)

// PullRequest represents an Azure DevOps pull request
type PullRequest struct {
	PullRequestID int                   `json:"pullRequestId"`
	Title         string                `json:"title"`
	SourceRefName string                `json:"sourceRefName"`
	TargetRefName string                `json:"targetRefName"`
	CreationDate  time.Time             `json:"creationDate"`
	CreatedBy     Identity              `json:"createdBy"`
	Repository    PullRequestRepository `json:"repository"`
	Reviewers     []Reviewer            `json:"reviewers"`
	Status        PullRequestStatus     `json:"status"`
	IsDraft       bool                  `json:"isDraft"`
	MergeStatus   string                `json:"mergeStatus"`
	URL           string                `json:"url"`
}

// Reviewer represents a pull request reviewer
type Reviewer struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
	Vote        int    `json:"vote"` // 10=approved, 5=approved with suggestions, 0=no vote, -5=waiting, -10=rejected
}

// PullRequestRepository represents the repository for a pull request
type PullRequestRepository struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	URL     string      `json:"url"`
	Project TeamProject `json:"project"`
}

// PullRequestsResponse represents the API response for pull requests
type PullRequestsResponse struct {
	Count int           `json:"count"`
	Value []PullRequest `json:"value"`
}
