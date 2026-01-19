package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	// Wait for rate limiter
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
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

		body, err := c.doSingleRequest(ctx, url)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")

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
