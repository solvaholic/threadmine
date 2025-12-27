package normalize

import (
	"testing"
	"time"

	"github.com/solvaholic/threadmine/internal/github"
)

func TestGitHubIssueToNormalized(t *testing.T) {
	now := time.Now()
	issue := &github.Issue{
		Number:    123,
		Title:     "Test Issue",
		Body:      "This is a test issue with @mention and https://example.com",
		State:     "open",
		User:      github.User{ID: 1, Login: "testuser", Name: "Test User", Email: "test@example.com"},
		CreatedAt: now,
		UpdatedAt: now,
		Comments:  5,
	}

	normalized, err := GitHubIssueToNormalized(issue, "testrepo", "testowner", now)
	if err != nil {
		t.Fatalf("GitHubIssueToNormalized failed: %v", err)
	}

	// Check basic fields
	if normalized.ID != "msg_github_testowner_testrepo_issue_123" {
		t.Errorf("Expected ID 'msg_github_testowner_testrepo_issue_123', got '%s'", normalized.ID)
	}

	if normalized.SourceType != "github" {
		t.Errorf("Expected SourceType 'github', got '%s'", normalized.SourceType)
	}

	if !normalized.IsThreadRoot {
		t.Error("Expected issue to be thread root")
	}

	if normalized.ThreadID != "thread_github_testowner_testrepo_issue_123" {
		t.Errorf("Expected ThreadID 'thread_github_testowner_testrepo_issue_123', got '%s'", normalized.ThreadID)
	}

	if normalized.ParentID != "" {
		t.Errorf("Expected empty ParentID for root message, got '%s'", normalized.ParentID)
	}

	// Check author
	if normalized.Author == nil {
		t.Fatal("Expected Author to be non-nil")
	}
	if normalized.Author.DisplayName != "testuser" {
		t.Errorf("Expected Author.DisplayName 'testuser', got '%s'", normalized.Author.DisplayName)
	}

	// Check channel
	if normalized.Channel == nil {
		t.Fatal("Expected Channel to be non-nil")
	}
	if normalized.Channel.Type != "issue" {
		t.Errorf("Expected Channel.Type 'issue', got '%s'", normalized.Channel.Type)
	}

	// Check mentions
	if len(normalized.Mentions) == 0 {
		t.Error("Expected mentions to be extracted")
	}

	// Check URLs
	if len(normalized.URLs) == 0 {
		t.Error("Expected URLs to be extracted")
	}
}

func TestGitHubIssueCommentToNormalized(t *testing.T) {
	now := time.Now()
	issue := &github.Issue{
		Number:    123,
		Title:     "Test Issue",
		Body:      "Test issue",
		State:     "open",
		User:      github.User{ID: 1, Login: "issueauthor"},
		CreatedAt: now,
	}

	comment := &github.Comment{
		ID:        456,
		Body:      "This is a comment",
		User:      github.User{ID: 2, Login: "commenter"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	normalized, err := GitHubIssueCommentToNormalized(comment, issue, "testrepo", "testowner", now)
	if err != nil {
		t.Fatalf("GitHubIssueCommentToNormalized failed: %v", err)
	}

	// Check basic fields
	if normalized.ID != "msg_github_testowner_testrepo_issue_123_comment_456" {
		t.Errorf("Expected ID 'msg_github_testowner_testrepo_issue_123_comment_456', got '%s'", normalized.ID)
	}

	if normalized.IsThreadRoot {
		t.Error("Expected comment to not be thread root")
	}

	if normalized.ParentID != "msg_github_testowner_testrepo_issue_123" {
		t.Errorf("Expected ParentID 'msg_github_testowner_testrepo_issue_123', got '%s'", normalized.ParentID)
	}

	if normalized.ThreadID != "thread_github_testowner_testrepo_issue_123" {
		t.Errorf("Expected ThreadID 'thread_github_testowner_testrepo_issue_123', got '%s'", normalized.ThreadID)
	}

	// Check author
	if normalized.Author == nil {
		t.Fatal("Expected Author to be non-nil")
	}
	if normalized.Author.DisplayName != "commenter" {
		t.Errorf("Expected Author.DisplayName 'commenter', got '%s'", normalized.Author.DisplayName)
	}
}

func TestGitHubPRToNormalized(t *testing.T) {
	now := time.Now()
	pr := &github.PullRequest{
		Number:    456,
		Title:     "Test PR",
		Body:      "This is a test PR with code:\n```go\nfunc test() {}\n```",
		State:     "open",
		User:      github.User{ID: 1, Login: "prauthor"},
		CreatedAt: now,
		UpdatedAt: now,
		Comments:  3,
	}

	normalized, err := GitHubPRToNormalized(pr, "testrepo", "testowner", now)
	if err != nil {
		t.Fatalf("GitHubPRToNormalized failed: %v", err)
	}

	// Check basic fields
	if normalized.ID != "msg_github_testowner_testrepo_pr_456" {
		t.Errorf("Expected ID 'msg_github_testowner_testrepo_pr_456', got '%s'", normalized.ID)
	}

	if !normalized.IsThreadRoot {
		t.Error("Expected PR to be thread root")
	}

	if normalized.ThreadID != "thread_github_testowner_testrepo_pr_456" {
		t.Errorf("Expected ThreadID 'thread_github_testowner_testrepo_pr_456', got '%s'", normalized.ThreadID)
	}

	// Check channel
	if normalized.Channel == nil {
		t.Fatal("Expected Channel to be non-nil")
	}
	if normalized.Channel.Type != "pr" {
		t.Errorf("Expected Channel.Type 'pr', got '%s'", normalized.Channel.Type)
	}

	// Check code blocks
	if len(normalized.CodeBlocks) == 0 {
		t.Error("Expected code blocks to be extracted")
	} else if normalized.CodeBlocks[0].Language != "go" {
		t.Errorf("Expected code block language 'go', got '%s'", normalized.CodeBlocks[0].Language)
	}
}

func TestGitHubPRReviewToNormalized(t *testing.T) {
	now := time.Now()
	pr := &github.PullRequest{
		Number:    456,
		Title:     "Test PR",
		Body:      "Test PR",
		State:     "open",
		User:      github.User{ID: 1, Login: "prauthor"},
		CreatedAt: now,
	}

	review := &github.Review{
		ID:          789,
		Body:        "Looks good to me!",
		User:        github.User{ID: 2, Login: "reviewer"},
		State:       "approved",
		SubmittedAt: now,
	}

	normalized, err := GitHubPRReviewToNormalized(review, pr, "testrepo", "testowner", now)
	if err != nil {
		t.Fatalf("GitHubPRReviewToNormalized failed: %v", err)
	}

	// Check basic fields
	if normalized.ID != "msg_github_testowner_testrepo_pr_456_review_789" {
		t.Errorf("Expected ID 'msg_github_testowner_testrepo_pr_456_review_789', got '%s'", normalized.ID)
	}

	if normalized.IsThreadRoot {
		t.Error("Expected review to not be thread root")
	}

	if normalized.ParentID != "msg_github_testowner_testrepo_pr_456" {
		t.Errorf("Expected ParentID 'msg_github_testowner_testrepo_pr_456', got '%s'", normalized.ParentID)
	}

	// Check author
	if normalized.Author == nil {
		t.Fatal("Expected Author to be non-nil")
	}
	if normalized.Author.DisplayName != "reviewer" {
		t.Errorf("Expected Author.DisplayName 'reviewer', got '%s'", normalized.Author.DisplayName)
	}

	// Check source metadata
	if state, ok := normalized.SourceMetadata["state"].(string); !ok || state != "approved" {
		t.Errorf("Expected SourceMetadata.state 'approved', got '%v'", normalized.SourceMetadata["state"])
	}
}

func TestExtractGitHubMentions(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"@user1 hello @user2", 2},
		{"No mentions here", 0},
		{"@test-user with dash", 1},
		{"Email test@example.com should not match", 0},
	}

	for _, tt := range tests {
		mentions := extractGitHubMentions(tt.text)
		if len(mentions) != tt.expected {
			t.Errorf("extractGitHubMentions(%q) = %d mentions, expected %d", tt.text, len(mentions), tt.expected)
		}
	}
}

func TestExtractGitHubURLs(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"Check out https://example.com", 1},
		{"Two URLs: https://a.com and https://b.com", 2},
		{"No URLs here", 0},
		{"http://insecure.com also works", 1},
	}

	for _, tt := range tests {
		urls := extractGitHubURLs(tt.text)
		if len(urls) != tt.expected {
			t.Errorf("extractGitHubURLs(%q) = %d URLs, expected %d", tt.text, len(urls), tt.expected)
		}
	}
}

func TestExtractGitHubCodeBlocks(t *testing.T) {
	tests := []struct {
		text         string
		expectedLen  int
		expectedLang string
	}{
		{"```go\nfunc test() {}\n```", 1, "go"},
		{"```\nno language\n```", 1, ""},
		{"No code blocks", 0, ""},
		{"```python\nprint('hello')\n```\n```javascript\nconsole.log('hi')\n```", 2, "python"},
		{"```JavaScript\nconsole.log('mixed case')```", 1, "JavaScript"},
		{"```TypeScript\nconst x: string = 'test'```", 1, "TypeScript"},
	}

	for _, tt := range tests {
		blocks := extractGitHubCodeBlocks(tt.text)
		if len(blocks) != tt.expectedLen {
			t.Errorf("extractGitHubCodeBlocks(%q) = %d blocks, expected %d", tt.text, len(blocks), tt.expectedLen)
		}
		if tt.expectedLen > 0 && blocks[0].Language != tt.expectedLang {
			t.Errorf("extractGitHubCodeBlocks(%q) first block language = %q, expected %q", tt.text, blocks[0].Language, tt.expectedLang)
		}
	}
}

func TestNormalizeGitHubMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"**bold** text", "bold text"},
		{"*italic* text", "italic text"},
		{"`inline code`", "inline code"},
		{"__underline__", "underline"},
	}

	for _, tt := range tests {
		result := normalizeGitHubMarkdown(tt.input)
		if result != tt.contains {
			t.Errorf("normalizeGitHubMarkdown(%q) should contain %q, got %q", tt.input, tt.contains, result)
		}
	}
}
