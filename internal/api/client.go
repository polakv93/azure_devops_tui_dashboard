package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Client is the Azure DevOps API client
type Client struct {
	httpClient   *http.Client
	baseURL      string
	organization string
	authHeader   string
	limiter      *rate.Limiter
}

// ClientConfig holds configuration for creating a new client
type ClientConfig struct {
	Organization      string
	BaseURL           string
	PAT               string
	RequestsPerSecond float64
	BurstSize         int
	Timeout           time.Duration
}

// NewClient creates a new Azure DevOps API client
func NewClient(cfg ClientConfig) *Client {
	// Create Basic Auth header with PAT
	auth := base64.StdEncoding.EncodeToString([]byte(":" + cfg.PAT))

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL:      strings.TrimSuffix(cfg.BaseURL, "/"),
		organization: cfg.Organization,
		authHeader:   "Basic " + auth,
		limiter:      rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.BurstSize),
	}
}

// doRequest performs an HTTP request with rate limiting and retries
func (c *Client) doRequest(ctx context.Context, url string) ([]byte, error) {
	return c.doRequestWithBody(ctx, http.MethodGet, url, nil)
}

// doRequestWithBody performs an HTTP request with optional JSON body, rate limiting and retries.
func (c *Client) doRequestWithBody(ctx context.Context, method, url string, payload any) ([]byte, error) {
	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	var requestBody []byte
	if payload != nil {
		var err error
		requestBody, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Retry with exponential backoff
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		body, err := c.doSingleRequestWithBody(ctx, method, url, requestBody)
		if err == nil {
			return body, nil
		}

		lastErr = err

		// Don't retry on context cancellation or client errors (4xx)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("request failed after 3 attempts: %w", lastErr)
}

func (c *Client) doSingleRequest(ctx context.Context, url string) ([]byte, error) {
	return c.doSingleRequestWithBody(ctx, http.MethodGet, url, nil)
}

func (c *Client) doSingleRequestWithBody(ctx context.Context, method, url string, requestBody []byte) ([]byte, error) {
	var bodyReader io.Reader
	if len(requestBody) > 0 {
		bodyReader = bytes.NewReader(requestBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	if len(requestBody) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetRepositories fetches repositories for a project.
// If repositoryNames is non-empty, results are filtered to those names.
func (c *Client) GetRepositories(ctx context.Context, project string, repositoryNames []string) ([]Repository, error) {
	url := fmt.Sprintf("%s/%s/%s/_apis/git/repositories?api-version=7.1",
		c.baseURL, c.organization, project)

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response RepositoriesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse repositories response: %w", err)
	}

	if len(repositoryNames) == 0 {
		return response.Value, nil
	}

	allowed := make(map[string]bool, len(repositoryNames))
	for _, name := range repositoryNames {
		allowed[name] = true
	}

	filtered := make([]Repository, 0, len(response.Value))
	for _, repo := range response.Value {
		if allowed[repo.Name] {
			filtered = append(filtered, repo)
		}
	}

	return filtered, nil
}

// GetRepositoryBranches fetches branch names for a repository.
func (c *Client) GetRepositoryBranches(ctx context.Context, project, repositoryID string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/refs?filter=heads/&api-version=7.1",
		c.baseURL, c.organization, project, repositoryID)

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response GitRefsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse refs response: %w", err)
	}

	branches := make([]string, 0, len(response.Value))
	for _, ref := range response.Value {
		if strings.HasPrefix(ref.Name, "refs/heads/") {
			branches = append(branches, strings.TrimPrefix(ref.Name, "refs/heads/"))
		}
	}

	sort.Strings(branches)
	return branches, nil
}

// GetRecentBranchPushTimes fetches recent push activity mapped by branch name.
func (c *Client) GetRecentBranchPushTimes(ctx context.Context, project, repositoryID string, top int) (map[string]time.Time, error) {
	if top <= 0 {
		top = 100
	}

	v := url.Values{}
	v.Set("$top", fmt.Sprintf("%d", top))
	v.Set("searchCriteria.includeRefUpdates", "true")
	v.Set("api-version", "7.1")

	url := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/pushes?%s",
		c.baseURL, c.organization, project, repositoryID, v.Encode())

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response PushesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse pushes response: %w", err)
	}

	branchPushTimes := make(map[string]time.Time)
	for _, push := range response.Value {
		for _, update := range push.RefUpdates {
			if !strings.HasPrefix(update.Name, "refs/heads/") {
				continue
			}
			branch := strings.TrimPrefix(update.Name, "refs/heads/")
			if current, ok := branchPushTimes[branch]; !ok || push.Date.After(current) {
				branchPushTimes[branch] = push.Date
			}
		}
	}

	return branchPushTimes, nil
}

// GetLatestCommitMessage returns latest commit message for branch.
func (c *Client) GetLatestCommitMessage(ctx context.Context, project, repositoryID, branch string) (string, error) {
	v := url.Values{}
	v.Set("searchCriteria.$top", "1")
	v.Set("searchCriteria.itemVersion.version", branch)
	v.Set("searchCriteria.itemVersion.versionType", "branch")
	v.Set("api-version", "7.1")

	url := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/commits?%s",
		c.baseURL, c.organization, project, repositoryID, v.Encode())

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return "", err
	}

	var response CommitsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse commits response: %w", err)
	}

	if len(response.Value) == 0 {
		return "", nil
	}

	return strings.TrimSpace(response.Value[0].Comment), nil
}

// GetBuilds fetches builds for a project
func (c *Client) GetBuilds(ctx context.Context, project string, definitionIDs []int, branches []string, maxCount int) ([]Build, error) {
	// Fetch more items if we need to filter by branches (to ensure we get enough results after filtering)
	fetchCount := maxCount
	if len(branches) > 0 {
		fetchCount = maxCount * 5 // Fetch more to account for filtering
		if fetchCount > 100 {
			fetchCount = 100
		}
	}

	url := fmt.Sprintf("%s/%s/%s/_apis/build/builds?api-version=7.0&$top=%d&statusFilter=all&queryOrder=queueTimeDescending",
		c.baseURL, c.organization, project, fetchCount)

	if len(definitionIDs) > 0 {
		ids := make([]string, len(definitionIDs))
		for i, id := range definitionIDs {
			ids[i] = fmt.Sprintf("%d", id)
		}
		url += "&definitions=" + strings.Join(ids, ",")
	}

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response BuildsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse builds response: %w", err)
	}

	builds := response.Value

	// Filter by branches if specified
	if len(branches) > 0 {
		builds = filterBuildsByBranches(builds, branches)
	}

	// Limit results to maxCount
	if len(builds) > maxCount {
		builds = builds[:maxCount]
	}

	// Fetch timeline stages for each build in parallel
	var wg sync.WaitGroup
	for i := range builds {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			stages, err := c.GetBuildTimeline(ctx, project, builds[i].ID)
			if err == nil {
				builds[i].Stages = stages
			}
		}(i)
	}
	wg.Wait()

	return builds, nil
}

// filterBuildsByBranches filters builds to only include those from specified branches
func filterBuildsByBranches(builds []Build, branches []string) []Build {
	// Create a map for quick branch lookup (normalize branch names)
	branchMap := make(map[string]bool)
	for _, branch := range branches {
		// Support both "main" and "refs/heads/main" formats
		normalized := strings.TrimPrefix(branch, "refs/heads/")
		branchMap[normalized] = true
		branchMap["refs/heads/"+normalized] = true
	}

	var filtered []Build
	for _, build := range builds {
		if branchMap[build.SourceBranch] {
			filtered = append(filtered, build)
		}
	}
	return filtered
}

// GetBuildTimeline fetches the timeline records for a build and returns only Stage-type records
func (c *Client) GetBuildTimeline(ctx context.Context, project string, buildID int) ([]BuildTimelineRecord, error) {
	url := fmt.Sprintf("%s/%s/%s/_apis/build/builds/%d/timeline?api-version=7.0",
		c.baseURL, c.organization, project, buildID)

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response BuildTimelineResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse timeline response: %w", err)
	}

	// Filter to Stage-type records only and sort by order
	var stages []BuildTimelineRecord
	for _, record := range response.Records {
		if record.Type == "Stage" {
			stages = append(stages, record)
		}
	}

	// Sort stages by order field
	for i := 1; i < len(stages); i++ {
		for j := i; j > 0 && stages[j].Order < stages[j-1].Order; j-- {
			stages[j], stages[j-1] = stages[j-1], stages[j]
		}
	}

	return stages, nil
}

// GetReleases fetches releases for a project
func (c *Client) GetReleases(ctx context.Context, project string, definitionIDs []int, maxCount int) ([]Release, error) {
	// Note: Releases API uses a different base URL (vsrm.dev.azure.com)
	releaseURL := strings.Replace(c.baseURL, "dev.azure.com", "vsrm.dev.azure.com", 1)

	url := fmt.Sprintf("%s/%s/%s/_apis/release/releases?api-version=7.0&$top=%d&$expand=environments",
		releaseURL, c.organization, project, maxCount)

	if len(definitionIDs) > 0 {
		ids := make([]string, len(definitionIDs))
		for i, id := range definitionIDs {
			ids[i] = fmt.Sprintf("%d", id)
		}
		url += "&definitionId=" + strings.Join(ids, ",")
	}

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response ReleasesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse releases response: %w", err)
	}

	return response.Value, nil
}

// GetBuildWebURL returns the web URL for a build
func (c *Client) GetBuildWebURL(project string, buildID int) string {
	return fmt.Sprintf("%s/%s/%s/_build/results?buildId=%d",
		c.baseURL, c.organization, project, buildID)
}

// GetReleaseWebURL returns the web URL for a release
func (c *Client) GetReleaseWebURL(project string, releaseID int) string {
	return fmt.Sprintf("%s/%s/%s/_releaseProgress?releaseId=%d",
		c.baseURL, c.organization, project, releaseID)
}

// GetPullRequests fetches pull requests for a project
func (c *Client) GetPullRequests(ctx context.Context, project string, repositories []string, maxCount int) ([]PullRequest, error) {
	var allPRs []PullRequest

	// If no specific repositories are specified, get all PRs for the project
	if len(repositories) == 0 {
		prs, err := c.getPullRequestsForProject(ctx, project, maxCount)
		if err != nil {
			return nil, err
		}
		return prs, nil
	}

	// Fetch PRs for each repository
	for _, repo := range repositories {
		prs, err := c.getPullRequestsForRepo(ctx, project, repo, maxCount)
		if err != nil {
			// Log error but continue with other repos
			continue
		}
		allPRs = append(allPRs, prs...)
	}

	// Limit total results
	if len(allPRs) > maxCount {
		allPRs = allPRs[:maxCount]
	}

	return allPRs, nil
}

// getPullRequestsForProject fetches all active PRs for a project
func (c *Client) getPullRequestsForProject(ctx context.Context, project string, maxCount int) ([]PullRequest, error) {
	v := url.Values{}
	v.Set("api-version", "7.0")
	v.Set("searchCriteria.status", "active")
	v.Set("$top", fmt.Sprintf("%d", maxCount))

	url := fmt.Sprintf("%s/%s/%s/_apis/git/pullrequests?%s",
		c.baseURL, c.organization, project, v.Encode())

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response PullRequestsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse pull requests response: %w", err)
	}

	return response.Value, nil
}

// getPullRequestsForRepo fetches active PRs for a specific repository
func (c *Client) getPullRequestsForRepo(ctx context.Context, project, repository string, maxCount int) ([]PullRequest, error) {
	v := url.Values{}
	v.Set("api-version", "7.0")
	v.Set("searchCriteria.status", "active")
	v.Set("$top", fmt.Sprintf("%d", maxCount))

	url := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/pullrequests?%s",
		c.baseURL, c.organization, project, repository, v.Encode())

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response PullRequestsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse pull requests response: %w", err)
	}

	return response.Value, nil
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(ctx context.Context, project, repositoryID string, request PullRequestCreateRequest) (PullRequest, error) {
	url := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/pullrequests?api-version=7.1",
		c.baseURL, c.organization, project, repositoryID)

	body, err := c.doRequestWithBody(ctx, http.MethodPost, url, request)
	if err != nil {
		return PullRequest{}, err
	}

	var response PullRequest
	if err := json.Unmarshal(body, &response); err != nil {
		return PullRequest{}, fmt.Errorf("failed to parse create pull request response: %w", err)
	}

	return response, nil
}

// UpdatePullRequest updates pull request fields such as auto-complete options.
func (c *Client) UpdatePullRequest(ctx context.Context, project, repositoryID string, pullRequestID int, request PullRequestUpdateRequest) (PullRequest, error) {
	url := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/pullrequests/%d?api-version=7.1",
		c.baseURL, c.organization, project, repositoryID, pullRequestID)

	body, err := c.doRequestWithBody(ctx, http.MethodPatch, url, request)
	if err != nil {
		return PullRequest{}, err
	}

	var response PullRequest
	if err := json.Unmarshal(body, &response); err != nil {
		return PullRequest{}, fmt.Errorf("failed to parse update pull request response: %w", err)
	}

	return response, nil
}

// SetPullRequestReviewerVote adds/updates reviewer vote on pull request.
func (c *Client) SetPullRequestReviewerVote(ctx context.Context, project, repositoryID string, pullRequestID int, reviewerID string, vote int) (Reviewer, error) {
	url := fmt.Sprintf("%s/%s/%s/_apis/git/repositories/%s/pullRequests/%d/reviewers/%s?api-version=7.1",
		c.baseURL, c.organization, project, repositoryID, pullRequestID, reviewerID)

	request := PullRequestReviewerVoteRequest{
		ID:   reviewerID,
		Vote: vote,
	}

	body, err := c.doRequestWithBody(ctx, http.MethodPut, url, request)
	if err != nil {
		return Reviewer{}, err
	}

	var response Reviewer
	if err := json.Unmarshal(body, &response); err != nil {
		return Reviewer{}, fmt.Errorf("failed to parse reviewer vote response: %w", err)
	}

	return response, nil
}

// GetPullRequestWebURL returns the web URL for a pull request
func (c *Client) GetPullRequestWebURL(project, repoName string, prID int) string {
	return fmt.Sprintf("%s/%s/%s/_git/%s/pullrequest/%d",
		c.baseURL, c.organization, project, repoName, prID)
}
