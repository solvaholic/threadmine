package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Client wraps GitHub CLI for API access
type Client struct {
	owner string
	repo  string
}

// AuthResult contains GitHub authentication information
type AuthResult struct {
	User   string
	Client *Client
}

// Authenticate verifies GitHub CLI authentication
func Authenticate() (*AuthResult, error) {
	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("GitHub CLI (gh) not found. Install it from https://cli.github.com/")
	}

	// Verify authentication status
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("GitHub CLI authentication failed. Run 'gh auth login' to authenticate.\n  Error: %v\n  Output: %s", err, string(output))
	}

	// Extract username
	cmd = exec.Command("gh", "api", "user", "--jq", ".login")
	userOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub user: %w", err)
	}

	username := strings.TrimSpace(string(userOutput))
	if username == "" {
		return nil, fmt.Errorf("failed to determine GitHub username")
	}

	return &AuthResult{
		User:   username,
		Client: &Client{},
	}, nil
}

// NewClient creates a new GitHub client for a specific repository
func NewClient(owner, repo string) *Client {
	return &Client{
		owner: owner,
		repo:  repo,
	}
}

// Repository represents a GitHub repository
type Repository struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Owner       User   `json:"owner"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
}

// Issue represents a GitHub issue
type Issue struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	User      User       `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	Comments  int        `json:"comments"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	User      User       `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	MergedAt  *time.Time `json:"merged_at"`
	Comments  int        `json:"comments"`
}

// Comment represents a GitHub issue or PR comment
type Comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Review represents a GitHub PR review
type Review struct {
	ID          int64     `json:"id"`
	Body        string    `json:"body"`
	User        User      `json:"user"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// User represents a GitHub user
type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GetRepository fetches repository metadata
func (c *Client) GetRepository(ctx context.Context) (*Repository, error) {
	cmd := exec.CommandContext(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s", c.owner, c.repo))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}

	var repo Repository
	if err := json.Unmarshal(output, &repo); err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}

	return &repo, nil
}

// GetIssues fetches issues with cache-aside pattern
func (c *Client) GetIssues(ctx context.Context, since time.Time) ([]Issue, error) {
	// Check cache first
	cached, err := c.loadIssuesFromCache(since)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch from API
	issues, err := c.FetchIssues(ctx, since)
	if err != nil {
		return nil, err
	}

	// Save to cache
	if err := c.saveIssuesToCache(issues); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cache issues: %v\n", err)
	}

	return issues, nil
}

// FetchIssues fetches issues from GitHub API (direct, no caching)
func (c *Client) FetchIssues(ctx context.Context, since time.Time) ([]Issue, error) {
	args := []string{"api", "--paginate", fmt.Sprintf("repos/%s/%s/issues", c.owner, c.repo)}
	
	if !since.IsZero() {
		args = append(args, "-f", fmt.Sprintf("since=%s", since.Format(time.RFC3339)))
	}
	
	// Filter out pull requests (GitHub API returns both)
	args = append(args, "-f", "state=all")

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issues: %w", err)
	}

	var rawIssues []map[string]interface{}
	if err := json.Unmarshal(output, &rawIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	// Filter out pull requests (they have a pull_request field)
	filtered := make([]Issue, 0)
	for _, rawIssue := range rawIssues {
		if _, hasPR := rawIssue["pull_request"]; !hasPR {
			// Convert to Issue struct
			issueBytes, err := json.Marshal(rawIssue)
			if err != nil {
				continue
			}
			var issue Issue
			if err := json.Unmarshal(issueBytes, &issue); err != nil {
				continue
			}
			filtered = append(filtered, issue)
		}
	}

	return filtered, nil
}

// GetIssueComments fetches comments for a specific issue
func (c *Client) GetIssueComments(ctx context.Context, issueNumber int) ([]Comment, error) {
	// Check cache first
	cached, err := c.loadIssueCommentsFromCache(issueNumber)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch from API
	comments, err := c.FetchIssueComments(ctx, issueNumber)
	if err != nil {
		return nil, err
	}

	// Save to cache
	if err := c.saveIssueCommentsToCache(issueNumber, comments); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cache issue comments: %v\n", err)
	}

	return comments, nil
}

// FetchIssueComments fetches comments for an issue (direct, no caching)
func (c *Client) FetchIssueComments(ctx context.Context, issueNumber int) ([]Comment, error) {
	cmd := exec.CommandContext(ctx, "gh", "api", "--paginate",
		fmt.Sprintf("repos/%s/%s/issues/%d/comments", c.owner, c.repo, issueNumber))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue comments: %w", err)
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse issue comments: %w", err)
	}

	return comments, nil
}

// GetPullRequests fetches pull requests with cache-aside pattern
func (c *Client) GetPullRequests(ctx context.Context, since time.Time) ([]PullRequest, error) {
	// Check cache first
	cached, err := c.loadPullRequestsFromCache(since)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch from API
	prs, err := c.FetchPullRequests(ctx, since)
	if err != nil {
		return nil, err
	}

	// Save to cache
	if err := c.savePullRequestsToCache(prs); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cache pull requests: %v\n", err)
	}

	return prs, nil
}

// FetchPullRequests fetches pull requests from GitHub API (direct, no caching)
func (c *Client) FetchPullRequests(ctx context.Context, since time.Time) ([]PullRequest, error) {
	args := []string{"api", "--paginate", fmt.Sprintf("repos/%s/%s/pulls", c.owner, c.repo)}
	args = append(args, "-f", "state=all")

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
	}

	var prs []PullRequest
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse pull requests: %w", err)
	}

	// Filter by date if specified
	if !since.IsZero() {
		filtered := make([]PullRequest, 0)
		for _, pr := range prs {
			if pr.UpdatedAt.After(since) {
				filtered = append(filtered, pr)
			}
		}
		return filtered, nil
	}

	return prs, nil
}

// GetPullRequestComments fetches comments for a specific pull request
func (c *Client) GetPullRequestComments(ctx context.Context, prNumber int) ([]Comment, error) {
	// Check cache first
	cached, err := c.loadPRCommentsFromCache(prNumber)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch from API
	comments, err := c.FetchPullRequestComments(ctx, prNumber)
	if err != nil {
		return nil, err
	}

	// Save to cache
	if err := c.savePRCommentsToCache(prNumber, comments); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cache PR comments: %v\n", err)
	}

	return comments, nil
}

// FetchPullRequestComments fetches comments for a PR (direct, no caching)
func (c *Client) FetchPullRequestComments(ctx context.Context, prNumber int) ([]Comment, error) {
	// Get issue comments (general comments on the PR)
	cmd := exec.CommandContext(ctx, "gh", "api", "--paginate",
		fmt.Sprintf("repos/%s/%s/issues/%d/comments", c.owner, c.repo, prNumber))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR comments: %w", err)
	}

	var comments []Comment
	if err := json.Unmarshal(output, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse PR comments: %w", err)
	}

	return comments, nil
}

// GetPullRequestReviews fetches reviews for a specific pull request
func (c *Client) GetPullRequestReviews(ctx context.Context, prNumber int) ([]Review, error) {
	// Check cache first
	cached, err := c.loadPRReviewsFromCache(prNumber)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch from API
	reviews, err := c.FetchPullRequestReviews(ctx, prNumber)
	if err != nil {
		return nil, err
	}

	// Save to cache
	if err := c.savePRReviewsToCache(prNumber, reviews); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cache PR reviews: %v\n", err)
	}

	return reviews, nil
}

// FetchPullRequestReviews fetches reviews for a PR (direct, no caching)
func (c *Client) FetchPullRequestReviews(ctx context.Context, prNumber int) ([]Review, error) {
	cmd := exec.CommandContext(ctx, "gh", "api", "--paginate",
		fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", c.owner, c.repo, prNumber))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR reviews: %w", err)
	}

	var reviews []Review
	if err := json.Unmarshal(output, &reviews); err != nil {
		return nil, fmt.Errorf("failed to parse PR reviews: %w", err)
	}

	return reviews, nil
}

// Cache helper functions

func (c *Client) getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".threadmine", "raw", "github", "repos", fmt.Sprintf("%s-%s", c.owner, c.repo)), nil
}

func (c *Client) loadIssuesFromCache(since time.Time) ([]Issue, error) {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(cacheDir, "issues", "_index.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // Cache miss
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cache struct {
		FetchedAt time.Time `json:"fetched_at"`
		Issues    []Issue   `json:"issues"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Check if cache is recent (within last hour)
	if time.Since(cache.FetchedAt) > time.Hour {
		return nil, nil // Cache too old
	}

	return cache.Issues, nil
}

func (c *Client) saveIssuesToCache(issues []Issue) error {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return err
	}

	issuesDir := filepath.Join(cacheDir, "issues")
	if err := os.MkdirAll(issuesDir, 0700); err != nil {
		return err
	}

	// Save index
	cache := struct {
		FetchedAt time.Time `json:"fetched_at"`
		Issues    []Issue   `json:"issues"`
	}{
		FetchedAt: time.Now(),
		Issues:    issues,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	indexPath := filepath.Join(issuesDir, "_index.json")
	tempPath := indexPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}

	if err := os.Rename(tempPath, indexPath); err != nil {
		os.Remove(tempPath)
		return err
	}

	// Save individual issues
	for _, issue := range issues {
		issueData, err := json.MarshalIndent(issue, "", "  ")
		if err != nil {
			continue
		}

		issuePath := filepath.Join(issuesDir, fmt.Sprintf("%d.json", issue.Number))
		tempPath := issuePath + ".tmp"
		if err := os.WriteFile(tempPath, issueData, 0600); err != nil {
			continue
		}
		os.Rename(tempPath, issuePath)
	}

	return nil
}

func (c *Client) loadIssueCommentsFromCache(issueNumber int) ([]Comment, error) {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(cacheDir, "comments", fmt.Sprintf("issue-%d", issueNumber), "comments.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // Cache miss
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cache struct {
		FetchedAt time.Time `json:"fetched_at"`
		Comments  []Comment `json:"comments"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Check if cache is recent (within last hour)
	if time.Since(cache.FetchedAt) > time.Hour {
		return nil, nil // Cache too old
	}

	return cache.Comments, nil
}

func (c *Client) saveIssueCommentsToCache(issueNumber int, comments []Comment) error {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return err
	}

	commentsDir := filepath.Join(cacheDir, "comments", fmt.Sprintf("issue-%d", issueNumber))
	if err := os.MkdirAll(commentsDir, 0700); err != nil {
		return err
	}

	cache := struct {
		FetchedAt time.Time `json:"fetched_at"`
		Comments  []Comment `json:"comments"`
	}{
		FetchedAt: time.Now(),
		Comments:  comments,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(commentsDir, "comments.json")
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

func (c *Client) loadPullRequestsFromCache(since time.Time) ([]PullRequest, error) {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(cacheDir, "pull_requests", "_index.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // Cache miss
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cache struct {
		FetchedAt    time.Time     `json:"fetched_at"`
		PullRequests []PullRequest `json:"pull_requests"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Check if cache is recent (within last hour)
	if time.Since(cache.FetchedAt) > time.Hour {
		return nil, nil // Cache too old
	}

	return cache.PullRequests, nil
}

func (c *Client) savePullRequestsToCache(prs []PullRequest) error {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return err
	}

	prsDir := filepath.Join(cacheDir, "pull_requests")
	if err := os.MkdirAll(prsDir, 0700); err != nil {
		return err
	}

	// Save index
	cache := struct {
		FetchedAt    time.Time     `json:"fetched_at"`
		PullRequests []PullRequest `json:"pull_requests"`
	}{
		FetchedAt:    time.Now(),
		PullRequests: prs,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	indexPath := filepath.Join(prsDir, "_index.json")
	tempPath := indexPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}

	if err := os.Rename(tempPath, indexPath); err != nil {
		os.Remove(tempPath)
		return err
	}

	// Save individual PRs
	for _, pr := range prs {
		prData, err := json.MarshalIndent(pr, "", "  ")
		if err != nil {
			continue
		}

		prPath := filepath.Join(prsDir, fmt.Sprintf("%d.json", pr.Number))
		tempPath := prPath + ".tmp"
		if err := os.WriteFile(tempPath, prData, 0600); err != nil {
			continue
		}
		os.Rename(tempPath, prPath)
	}

	return nil
}

func (c *Client) loadPRCommentsFromCache(prNumber int) ([]Comment, error) {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(cacheDir, "comments", fmt.Sprintf("pr-%d", prNumber), "comments.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // Cache miss
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cache struct {
		FetchedAt time.Time `json:"fetched_at"`
		Comments  []Comment `json:"comments"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Check if cache is recent (within last hour)
	if time.Since(cache.FetchedAt) > time.Hour {
		return nil, nil // Cache too old
	}

	return cache.Comments, nil
}

func (c *Client) savePRCommentsToCache(prNumber int, comments []Comment) error {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return err
	}

	commentsDir := filepath.Join(cacheDir, "comments", fmt.Sprintf("pr-%d", prNumber))
	if err := os.MkdirAll(commentsDir, 0700); err != nil {
		return err
	}

	cache := struct {
		FetchedAt time.Time `json:"fetched_at"`
		Comments  []Comment `json:"comments"`
	}{
		FetchedAt: time.Now(),
		Comments:  comments,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(commentsDir, "comments.json")
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}

func (c *Client) loadPRReviewsFromCache(prNumber int) ([]Review, error) {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(cacheDir, "comments", fmt.Sprintf("pr-%d", prNumber), "reviews.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // Cache miss
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cache struct {
		FetchedAt time.Time `json:"fetched_at"`
		Reviews   []Review  `json:"reviews"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Check if cache is recent (within last hour)
	if time.Since(cache.FetchedAt) > time.Hour {
		return nil, nil // Cache too old
	}

	return cache.Reviews, nil
}

func (c *Client) savePRReviewsToCache(prNumber int, reviews []Review) error {
	cacheDir, err := c.getCacheDir()
	if err != nil {
		return err
	}

	commentsDir := filepath.Join(cacheDir, "comments", fmt.Sprintf("pr-%d", prNumber))
	if err := os.MkdirAll(commentsDir, 0700); err != nil {
		return err
	}

	cache := struct {
		FetchedAt time.Time `json:"fetched_at"`
		Reviews   []Review  `json:"reviews"`
	}{
		FetchedAt: time.Now(),
		Reviews:   reviews,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(commentsDir, "reviews.json")
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return err
	}

	return nil
}
