package normalize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/solvaholic/threadmine/internal/github"
)

var (
	// GitHub Markdown patterns
	// Match @username but not in email addresses (require word boundary before @)
	githubMentionPattern = regexp.MustCompile(`(?:^|[^a-zA-Z0-9.])@([a-zA-Z0-9][-a-zA-Z0-9]*)`)
	githubURLPattern     = regexp.MustCompile(`https?://[^\s\)]+`)
	githubCodeBlockPattern = regexp.MustCompile("```([a-z]*)\n([^`]+)```")
	githubInlineCodePattern = regexp.MustCompile("`([^`]+)`")
)

// GitHubIssueToNormalized converts a GitHub issue to normalized messages
// The issue itself becomes the root message, and comments become replies
func GitHubIssueToNormalized(issue *github.Issue, repo, owner string, fetchedAt time.Time) (*NormalizedMessage, error) {
	// Generate universal ID for the issue (root message)
	msgID := fmt.Sprintf("msg_github_%s_%s_issue_%d", owner, repo, issue.Number)
	threadID := fmt.Sprintf("thread_github_%s_%s_issue_%d", owner, repo, issue.Number)

	// Extract mentions, URLs, and code blocks from issue body
	mentions := extractGitHubMentions(issue.Body)
	urls := extractGitHubURLs(issue.Body)
	codeBlocks := extractGitHubCodeBlocks(issue.Body)

	// Build normalized message for the issue
	normalized := &NormalizedMessage{
		ID:         msgID,
		SourceType: "github",
		SourceID:   fmt.Sprintf("%s/%s/issues/%d", owner, repo, issue.Number),
		Timestamp:  issue.CreatedAt,
		Author:     convertGitHubUser(&issue.User, owner, repo),
		Content:    normalizeGitHubMarkdown(issue.Body),
		ContentHTML: "", // Could use GitHub's rendering API in the future
		Channel:    convertGitHubIssueToChannel(issue, repo, owner),
		ThreadID:   threadID,
		ParentID:   "",
		IsThreadRoot: true,
		Attachments: nil,
		Mentions:   mentions,
		URLs:       urls,
		CodeBlocks: codeBlocks,
		SourceMetadata: map[string]interface{}{
			"owner":      owner,
			"repo":       repo,
			"issue_number": issue.Number,
			"title":      issue.Title,
			"state":      issue.State,
			"closed_at":  issue.ClosedAt,
		},
		FetchedAt:    fetchedAt,
		NormalizedAt: time.Now(),
		SchemaVersion: SchemaVersion,
	}

	return normalized, nil
}

// GitHubIssueCommentToNormalized converts a GitHub issue comment to a normalized message
func GitHubIssueCommentToNormalized(comment *github.Comment, issue *github.Issue, repo, owner string, fetchedAt time.Time) (*NormalizedMessage, error) {
	// Generate universal IDs
	msgID := fmt.Sprintf("msg_github_%s_%s_issue_%d_comment_%d", owner, repo, issue.Number, comment.ID)
	threadID := fmt.Sprintf("thread_github_%s_%s_issue_%d", owner, repo, issue.Number)
	parentID := fmt.Sprintf("msg_github_%s_%s_issue_%d", owner, repo, issue.Number)

	// Extract mentions, URLs, and code blocks
	mentions := extractGitHubMentions(comment.Body)
	urls := extractGitHubURLs(comment.Body)
	codeBlocks := extractGitHubCodeBlocks(comment.Body)

	normalized := &NormalizedMessage{
		ID:         msgID,
		SourceType: "github",
		SourceID:   fmt.Sprintf("%s/%s/issues/%d#issuecomment-%d", owner, repo, issue.Number, comment.ID),
		Timestamp:  comment.CreatedAt,
		Author:     convertGitHubUser(&comment.User, owner, repo),
		Content:    normalizeGitHubMarkdown(comment.Body),
		ContentHTML: "",
		Channel:    convertGitHubIssueToChannel(issue, repo, owner),
		ThreadID:   threadID,
		ParentID:   parentID,
		IsThreadRoot: false,
		Attachments: nil,
		Mentions:   mentions,
		URLs:       urls,
		CodeBlocks: codeBlocks,
		SourceMetadata: map[string]interface{}{
			"owner":        owner,
			"repo":         repo,
			"issue_number": issue.Number,
			"comment_id":   comment.ID,
			"updated_at":   comment.UpdatedAt,
		},
		FetchedAt:    fetchedAt,
		NormalizedAt: time.Now(),
		SchemaVersion: SchemaVersion,
	}

	return normalized, nil
}

// GitHubPRToNormalized converts a GitHub pull request to normalized messages
func GitHubPRToNormalized(pr *github.PullRequest, repo, owner string, fetchedAt time.Time) (*NormalizedMessage, error) {
	// Generate universal IDs
	msgID := fmt.Sprintf("msg_github_%s_%s_pr_%d", owner, repo, pr.Number)
	threadID := fmt.Sprintf("thread_github_%s_%s_pr_%d", owner, repo, pr.Number)

	// Extract mentions, URLs, and code blocks
	mentions := extractGitHubMentions(pr.Body)
	urls := extractGitHubURLs(pr.Body)
	codeBlocks := extractGitHubCodeBlocks(pr.Body)

	normalized := &NormalizedMessage{
		ID:         msgID,
		SourceType: "github",
		SourceID:   fmt.Sprintf("%s/%s/pull/%d", owner, repo, pr.Number),
		Timestamp:  pr.CreatedAt,
		Author:     convertGitHubUser(&pr.User, owner, repo),
		Content:    normalizeGitHubMarkdown(pr.Body),
		ContentHTML: "",
		Channel:    convertGitHubPRToChannel(pr, repo, owner),
		ThreadID:   threadID,
		ParentID:   "",
		IsThreadRoot: true,
		Attachments: nil,
		Mentions:   mentions,
		URLs:       urls,
		CodeBlocks: codeBlocks,
		SourceMetadata: map[string]interface{}{
			"owner":      owner,
			"repo":       repo,
			"pr_number":  pr.Number,
			"title":      pr.Title,
			"state":      pr.State,
			"merged_at":  pr.MergedAt,
			"closed_at":  pr.ClosedAt,
		},
		FetchedAt:    fetchedAt,
		NormalizedAt: time.Now(),
		SchemaVersion: SchemaVersion,
	}

	return normalized, nil
}

// GitHubPRCommentToNormalized converts a GitHub PR comment to a normalized message
func GitHubPRCommentToNormalized(comment *github.Comment, pr *github.PullRequest, repo, owner string, fetchedAt time.Time) (*NormalizedMessage, error) {
	msgID := fmt.Sprintf("msg_github_%s_%s_pr_%d_comment_%d", owner, repo, pr.Number, comment.ID)
	threadID := fmt.Sprintf("thread_github_%s_%s_pr_%d", owner, repo, pr.Number)
	parentID := fmt.Sprintf("msg_github_%s_%s_pr_%d", owner, repo, pr.Number)

	mentions := extractGitHubMentions(comment.Body)
	urls := extractGitHubURLs(comment.Body)
	codeBlocks := extractGitHubCodeBlocks(comment.Body)

	normalized := &NormalizedMessage{
		ID:         msgID,
		SourceType: "github",
		SourceID:   fmt.Sprintf("%s/%s/pull/%d#issuecomment-%d", owner, repo, pr.Number, comment.ID),
		Timestamp:  comment.CreatedAt,
		Author:     convertGitHubUser(&comment.User, owner, repo),
		Content:    normalizeGitHubMarkdown(comment.Body),
		ContentHTML: "",
		Channel:    convertGitHubPRToChannel(pr, repo, owner),
		ThreadID:   threadID,
		ParentID:   parentID,
		IsThreadRoot: false,
		Attachments: nil,
		Mentions:   mentions,
		URLs:       urls,
		CodeBlocks: codeBlocks,
		SourceMetadata: map[string]interface{}{
			"owner":      owner,
			"repo":       repo,
			"pr_number":  pr.Number,
			"comment_id": comment.ID,
			"updated_at": comment.UpdatedAt,
		},
		FetchedAt:    fetchedAt,
		NormalizedAt: time.Now(),
		SchemaVersion: SchemaVersion,
	}

	return normalized, nil
}

// GitHubPRReviewToNormalized converts a GitHub PR review to a normalized message
func GitHubPRReviewToNormalized(review *github.Review, pr *github.PullRequest, repo, owner string, fetchedAt time.Time) (*NormalizedMessage, error) {
	msgID := fmt.Sprintf("msg_github_%s_%s_pr_%d_review_%d", owner, repo, pr.Number, review.ID)
	threadID := fmt.Sprintf("thread_github_%s_%s_pr_%d", owner, repo, pr.Number)
	parentID := fmt.Sprintf("msg_github_%s_%s_pr_%d", owner, repo, pr.Number)

	mentions := extractGitHubMentions(review.Body)
	urls := extractGitHubURLs(review.Body)
	codeBlocks := extractGitHubCodeBlocks(review.Body)

	normalized := &NormalizedMessage{
		ID:         msgID,
		SourceType: "github",
		SourceID:   fmt.Sprintf("%s/%s/pull/%d#pullrequestreview-%d", owner, repo, pr.Number, review.ID),
		Timestamp:  review.SubmittedAt,
		Author:     convertGitHubUser(&review.User, owner, repo),
		Content:    normalizeGitHubMarkdown(review.Body),
		ContentHTML: "",
		Channel:    convertGitHubPRToChannel(pr, repo, owner),
		ThreadID:   threadID,
		ParentID:   parentID,
		IsThreadRoot: false,
		Attachments: nil,
		Mentions:   mentions,
		URLs:       urls,
		CodeBlocks: codeBlocks,
		SourceMetadata: map[string]interface{}{
			"owner":      owner,
			"repo":       repo,
			"pr_number":  pr.Number,
			"review_id":  review.ID,
			"state":      review.State,
		},
		FetchedAt:    fetchedAt,
		NormalizedAt: time.Now(),
		SchemaVersion: SchemaVersion,
	}

	return normalized, nil
}

// convertGitHubUser converts a GitHub user to the normalized User schema
func convertGitHubUser(user *github.User, owner, repo string) *User {
	if user == nil {
		return nil
	}
	return &User{
		ID:          fmt.Sprintf("user_github_%s", user.Login),
		SourceType:  "github",
		SourceID:    strconv.FormatInt(user.ID, 10),
		DisplayName: user.Login,
		RealName:    user.Name,
		Email:       user.Email,
		AvatarURL:   user.AvatarURL,
		CanonicalID: "",
		AlternateIDs: nil,
	}
}

// convertGitHubIssueToChannel converts a GitHub issue to the normalized Channel schema
func convertGitHubIssueToChannel(issue *github.Issue, repo, owner string) *Channel {
	if issue == nil {
		return nil
	}

	return &Channel{
		ID:          fmt.Sprintf("chan_github_%s_%s_issue_%d", owner, repo, issue.Number),
		SourceType:  "github",
		SourceID:    fmt.Sprintf("%s/%s/issues/%d", owner, repo, issue.Number),
		Name:        fmt.Sprintf("#%d", issue.Number),
		DisplayName: fmt.Sprintf("%s/%s#%d: %s", owner, repo, issue.Number, issue.Title),
		Type:        "issue",
		IsPrivate:   false,
		ParentSpace: fmt.Sprintf("%s/%s", owner, repo),
	}
}

// convertGitHubPRToChannel converts a GitHub PR to the normalized Channel schema
func convertGitHubPRToChannel(pr *github.PullRequest, repo, owner string) *Channel {
	if pr == nil {
		return nil
	}

	return &Channel{
		ID:          fmt.Sprintf("chan_github_%s_%s_pr_%d", owner, repo, pr.Number),
		SourceType:  "github",
		SourceID:    fmt.Sprintf("%s/%s/pull/%d", owner, repo, pr.Number),
		Name:        fmt.Sprintf("#%d", pr.Number),
		DisplayName: fmt.Sprintf("%s/%s#%d: %s", owner, repo, pr.Number, pr.Title),
		Type:        "pr",
		IsPrivate:   false,
		ParentSpace: fmt.Sprintf("%s/%s", owner, repo),
	}
}

// extractGitHubMentions extracts user mentions from GitHub Markdown text
func extractGitHubMentions(text string) []string {
	matches := githubMentionPattern.FindAllStringSubmatch(text, -1)
	mentions := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			mentions = append(mentions, match[1])
		}
	}
	return mentions
}

// extractGitHubURLs extracts URLs from GitHub Markdown text
func extractGitHubURLs(text string) []string {
	matches := githubURLPattern.FindAllString(text, -1)
	return matches
}

// extractGitHubCodeBlocks extracts code blocks from GitHub Markdown
func extractGitHubCodeBlocks(text string) []CodeBlock {
	matches := githubCodeBlockPattern.FindAllStringSubmatch(text, -1)
	blocks := make([]CodeBlock, 0, len(matches))
	for _, match := range matches {
		if len(match) > 2 {
			blocks = append(blocks, CodeBlock{
				Language: match[1],
				Code:     match[2],
			})
		}
	}
	return blocks
}

// normalizeGitHubMarkdown converts GitHub Markdown to plain text
// This is a simple conversion - could be enhanced to preserve more formatting
func normalizeGitHubMarkdown(text string) string {
	// Remove inline code markers for plain text output
	// (but we've already extracted code blocks separately)
	text = githubInlineCodePattern.ReplaceAllString(text, "$1")
	
	// Basic Markdown cleanup
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")
	
	return text
}
