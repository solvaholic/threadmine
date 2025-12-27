package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/solvaholic/threadmine/internal/cache"
	"github.com/solvaholic/threadmine/internal/classify"
	"github.com/solvaholic/threadmine/internal/github"
	"github.com/solvaholic/threadmine/internal/graph"
	"github.com/solvaholic/threadmine/internal/normalize"
	"github.com/solvaholic/threadmine/internal/slack"
	"github.com/solvaholic/threadmine/internal/utils"
	"github.com/spf13/cobra"
)

var (
	fetchWorkspace string
	fetchChannel   string
	fetchSince     string
	fetchOwner     string
	fetchRepo      string
)

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch data from sources",
	Long:  `Fetch messages, users, and metadata from communication platforms.`,
}

// fetchSlackCmd represents the fetch slack command
var fetchSlackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Fetch data from Slack",
	Long:  `Fetch channels, messages, and user data from Slack workspaces using browser cookies.`,
	RunE:  runFetchSlack,
}

// fetchGitHubCmd represents the fetch github command
var fetchGitHubCmd = &cobra.Command{
	Use:   "github",
	Short: "Fetch data from GitHub",
	Long:  `Fetch issues, pull requests, and comments from GitHub repositories using GitHub CLI.`,
	RunE:  runFetchGitHub,
}

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.AddCommand(fetchSlackCmd)
	fetchCmd.AddCommand(fetchGitHubCmd)

	// Flags for fetch slack
	fetchSlackCmd.Flags().StringVarP(&fetchWorkspace, "workspace", "w", "", "Slack workspace name or 'all' for all cached workspaces (required)")
	fetchSlackCmd.Flags().StringVarP(&fetchChannel, "channel", "c", "", "Channel ID to fetch (default: first available)")
	fetchSlackCmd.Flags().StringVarP(&fetchSince, "since", "s", "7d", "Fetch messages since (e.g., '7d', '2025-12-15')")
	fetchSlackCmd.MarkFlagRequired("workspace")

	// Flags for fetch github
	fetchGitHubCmd.Flags().StringVarP(&fetchOwner, "owner", "o", "", "Repository owner (required)")
	fetchGitHubCmd.Flags().StringVarP(&fetchRepo, "repo", "r", "", "Repository name (required)")
	fetchGitHubCmd.Flags().StringVarP(&fetchSince, "since", "s", "30d", "Fetch issues/PRs updated since (e.g., '30d', '2025-12-01')")
	fetchGitHubCmd.MarkFlagRequired("owner")
	fetchGitHubCmd.MarkFlagRequired("repo")
}

func runFetchSlack(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse since date using shared utility
	var oldest time.Time
	if len(fetchSince) > 0 {
		var err error
		oldest, err = utils.ParseSinceDate(fetchSince)
		if err != nil {
			OutputError("invalid date format: %v", err)
			return err
		}
	}

	// Handle --workspace all
	if fetchWorkspace == "all" {
		return runFetchSlackAll(ctx, oldest)
	}

	// Authenticate with Slack
	result, err := slack.Authenticate(fetchWorkspace)
	if err != nil {
		OutputError("authentication failed: %v", err)
		return err
	}

	// Save workspace user info for future "me" resolution
	if err := cache.SaveWorkspaceUser(result.TeamID, result.UserID, result.UserName, result.TeamName); err != nil {
		OutputError("failed to cache workspace user: %v", err)
		// Non-fatal, continue
	}

	// Fetch and process this workspace
	workspaceResult, _, _, err := fetchWorkspaceData(ctx, result, oldest)
	if err != nil {
		OutputError("fetch failed: %v", err)
		return err
	}

	// Add storage paths to single-workspace output
	cacheDir, _ := cache.CacheDir()
	normalizedDir, _ := normalize.NormalizedDir()
	graphDir, _ := graph.GraphDir()
	annotationsDir, _ := classify.AnnotationsDir()

	workspaceResult["storage"] = map[string]string{
		"raw":         cacheDir,
		"normalized":  normalizedDir,
		"graph":       graphDir,
		"annotations": annotationsDir,
	}
	workspaceResult["status"] = "success"

	// Add user info for single workspace
	workspaceResult["user"] = map[string]string{
		"name": result.UserName,
		"id":   result.UserID,
	}

	// Add date range
	if messages, ok := workspaceResult["messages"].(map[string]interface{}); ok {
		messages["date_range"] = map[string]string{
			"from": oldest.Format(time.RFC3339),
			"to":   time.Now().Format(time.RFC3339),
		}
	}

	return OutputJSON(workspaceResult)
}

// runFetchSlackAll fetches from all cached workspaces
func runFetchSlackAll(ctx context.Context, oldest time.Time) error {
	workspaceIDs, err := cache.DiscoverWorkspaces()
	if err != nil {
		OutputError("failed to discover workspaces: %v", err)
		return err
	}

	if len(workspaceIDs) == 0 {
		OutputError("no cached workspaces found")
		return fmt.Errorf("no cached workspaces found")
	}

	allResults := make([]map[string]interface{}, 0)
	var totalMessages, totalNormalized int

	for _, teamID := range workspaceIDs {
		// Get workspace user info
		workspaceUser, err := cache.GetWorkspaceUser(teamID)
		if err != nil {
			OutputError("skipping workspace %s: %v", teamID, err)
			continue
		}

		// Authenticate with Slack using team name
		result, err := slack.Authenticate(workspaceUser.TeamName)
		if err != nil {
			OutputError("authentication failed for %s: %v", workspaceUser.TeamName, err)
			continue
		}

		// Fetch and process this workspace
		workspaceResult, msgCount, normalizedCount, err := fetchWorkspaceData(ctx, result, oldest)
		if err != nil {
			OutputError("failed to fetch workspace %s: %v", workspaceUser.TeamName, err)
			continue
		}

		allResults = append(allResults, workspaceResult)
		totalMessages += msgCount
		totalNormalized += normalizedCount
	}

	// Build summary output
	output := map[string]interface{}{
		"status":           "success",
		"workspaces_count": len(allResults),
		"total_messages":   totalMessages,
		"total_normalized": totalNormalized,
		"workspaces":       allResults,
	}

	return OutputJSON(output)
}

// fetchWorkspaceData performs the fetch operation for a single workspace
func fetchWorkspaceData(ctx context.Context, result *slack.AuthResult, oldest time.Time) (map[string]interface{}, int, int, error) {
	// List channels
	channels, err := result.Client.ListChannels(ctx)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list channels: %w", err)
	}

	// Cache the channels list
	if err := cache.SaveChannelsList(result.TeamID, channels); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to cache channels: %w", err)
	}

	if len(channels) == 0 {
		return nil, 0, 0, fmt.Errorf("no channels found")
	}

	// Select channel
	var targetChannel *slack.Channel
	if fetchChannel != "" {
		for _, ch := range channels {
			if ch.ID == fetchChannel || ch.Name == fetchChannel {
				targetChannel = &ch
				break
			}
		}
		if targetChannel == nil {
			return nil, 0, 0, fmt.Errorf("channel not found: %s", fetchChannel)
		}
	} else {
		targetChannel = &channels[0]
	}

	// Save channel info
	if err := cache.SaveChannelInfo(result.TeamID, targetChannel.ID, *targetChannel); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to cache channel info: %w", err)
	}

	// Get messages using cache-aside pattern
	cacheDir, _ := cache.CacheDir()
	messages, err := result.Client.GetMessages(ctx, targetChannel.ID, oldest, cacheDir)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to retrieve messages: %w", err)
	}

	// Normalize messages
	user := &normalize.SlackUser{
		ID:       result.UserID,
		Name:     result.UserName,
		RealName: result.UserName,
	}

	slackChannel := &normalize.SlackChannel{
		ID:        targetChannel.ID,
		Name:      targetChannel.Name,
		IsChannel: true,
		IsPrivate: false,
	}

	var normalizedMessages []*normalize.NormalizedMessage
	for _, msg := range messages {
		slackMsg := &normalize.SlackMessage{
			Type:      msg.Type,
			User:      msg.User,
			Text:      msg.Text,
			Timestamp: msg.Timestamp,
			ThreadTS:  msg.ThreadTS,
		}

		normalized, err := normalize.SlackToNormalized(slackMsg, slackChannel, user, result.TeamID, time.Now())
		if err != nil {
			continue
		}

		if err := normalize.SaveNormalizedMessage(normalized); err != nil {
			continue
		}

		normalizedMessages = append(normalizedMessages, normalized)
	}

	// Classify messages
	threadContextMap := make(map[string]*classify.ThreadContext)
	for _, msg := range normalizedMessages {
		hasQuestion := false
		questionAuthor := ""
		for _, m := range normalizedMessages {
			if m.ThreadID == msg.ThreadID {
				msgClassifications := classify.ClassifyMessage(m, nil)
				for _, c := range msgClassifications {
					if c.Type == "question" {
						hasQuestion = true
						questionAuthor = m.Author.ID
						break
					}
				}
			}
		}

		threadContextMap[msg.ID] = &classify.ThreadContext{
			HasQuestion:    hasQuestion,
			QuestionAuthor: questionAuthor,
			IsThreadRoot:   msg.IsThreadRoot,
		}
	}

	questionCount := 0
	answerCount := 0
	solutionCount := 0
	acknowledgmentCount := 0

	for _, msg := range normalizedMessages {
		ctx := threadContextMap[msg.ID]
		classifications := classify.ClassifyMessage(msg, ctx)

		if len(classifications) > 0 {
			for _, c := range classifications {
				switch c.Type {
				case "question":
					questionCount++
				case "answer":
					answerCount++
				case "solution":
					solutionCount++
				case "acknowledgment":
					acknowledgmentCount++
				}
			}

			classify.SaveClassifications(msg, classifications)
		}
	}

	// Build reply graph
	replyGraph := graph.BuildFromNormalizedMessages(normalizedMessages)
	if err := graph.SaveReplyGraph(replyGraph); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to save reply graph: %w", err)
	}

	graphStats := replyGraph.Stats()

	// Build workspace result
	workspaceResult := map[string]interface{}{
		"workspace": map[string]string{
			"name": result.TeamName,
			"id":   result.TeamID,
		},
		"channel": map[string]interface{}{
			"id":   targetChannel.ID,
			"name": targetChannel.Name,
		},
		"messages": map[string]interface{}{
			"fetched":    len(messages),
			"normalized": len(normalizedMessages),
		},
		"classifications": map[string]interface{}{
			"questions":       questionCount,
			"answers":         answerCount,
			"solutions":       solutionCount,
			"acknowledgments": acknowledgmentCount,
		},
		"graph": graphStats,
	}

	return workspaceResult, len(messages), len(normalizedMessages), nil
}

func runFetchGitHub(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse since date
	var oldest time.Time
	if len(fetchSince) > 0 {
		var err error
		oldest, err = utils.ParseSinceDate(fetchSince)
		if err != nil {
			OutputError("invalid date format: %v", err)
			return err
		}
	}

	// Authenticate with GitHub CLI
	authResult, err := github.Authenticate()
	if err != nil {
		OutputError("GitHub authentication failed: %v", err)
		return err
	}

	// Create client for the specified repository
	client := github.NewClient(fetchOwner, fetchRepo)

	// Fetch repository metadata
	repo, err := client.GetRepository(ctx)
	if err != nil {
		OutputError("failed to fetch repository: %v", err)
		return err
	}

	fetchedAt := time.Now()
	var normalizedMessages []*normalize.NormalizedMessage

	// Fetch issues
	issues, err := client.GetIssues(ctx, oldest)
	if err != nil {
		OutputError("failed to fetch issues: %v", err)
		return err
	}

	issueCount := len(issues)
	issueCommentCount := 0

	for _, issue := range issues {
		// Normalize the issue itself
		normalized, err := normalize.GitHubIssueToNormalized(&issue, fetchRepo, fetchOwner, fetchedAt)
		if err != nil {
			continue
		}

		if err := normalize.SaveNormalizedMessage(normalized); err != nil {
			continue
		}
		normalizedMessages = append(normalizedMessages, normalized)

		// Fetch and normalize issue comments
		comments, err := client.GetIssueComments(ctx, issue.Number)
		if err != nil {
			continue
		}

		issueCommentCount += len(comments)

		for _, comment := range comments {
			normalizedComment, err := normalize.GitHubIssueCommentToNormalized(&comment, &issue, fetchRepo, fetchOwner, fetchedAt)
			if err != nil {
				continue
			}

			if err := normalize.SaveNormalizedMessage(normalizedComment); err != nil {
				continue
			}
			normalizedMessages = append(normalizedMessages, normalizedComment)
		}
	}

	// Fetch pull requests
	prs, err := client.GetPullRequests(ctx, oldest)
	if err != nil {
		OutputError("failed to fetch pull requests: %v", err)
		return err
	}

	prCount := len(prs)
	prCommentCount := 0
	prReviewCount := 0

	for _, pr := range prs {
		// Normalize the PR itself
		normalized, err := normalize.GitHubPRToNormalized(&pr, fetchRepo, fetchOwner, fetchedAt)
		if err != nil {
			continue
		}

		if err := normalize.SaveNormalizedMessage(normalized); err != nil {
			continue
		}
		normalizedMessages = append(normalizedMessages, normalized)

		// Fetch and normalize PR comments
		comments, err := client.GetPullRequestComments(ctx, pr.Number)
		if err != nil {
			continue
		}

		prCommentCount += len(comments)

		for _, comment := range comments {
			normalizedComment, err := normalize.GitHubPRCommentToNormalized(&comment, &pr, fetchRepo, fetchOwner, fetchedAt)
			if err != nil {
				continue
			}

			if err := normalize.SaveNormalizedMessage(normalizedComment); err != nil {
				continue
			}
			normalizedMessages = append(normalizedMessages, normalizedComment)
		}

		// Fetch and normalize PR reviews
		reviews, err := client.GetPullRequestReviews(ctx, pr.Number)
		if err != nil {
			continue
		}

		prReviewCount += len(reviews)

		for _, review := range reviews {
			normalizedReview, err := normalize.GitHubPRReviewToNormalized(&review, &pr, fetchRepo, fetchOwner, fetchedAt)
			if err != nil {
				continue
			}

			if err := normalize.SaveNormalizedMessage(normalizedReview); err != nil {
				continue
			}
			normalizedMessages = append(normalizedMessages, normalizedReview)
		}
	}

	// Classify messages
	threadContextMap := make(map[string]*classify.ThreadContext)
	for _, msg := range normalizedMessages {
		hasQuestion := false
		questionAuthor := ""
		for _, m := range normalizedMessages {
			if m.ThreadID == msg.ThreadID {
				msgClassifications := classify.ClassifyMessage(m, nil)
				for _, c := range msgClassifications {
					if c.Type == "question" {
						hasQuestion = true
						questionAuthor = m.Author.ID
						break
					}
				}
			}
		}

		threadContextMap[msg.ID] = &classify.ThreadContext{
			HasQuestion:    hasQuestion,
			QuestionAuthor: questionAuthor,
			IsThreadRoot:   msg.IsThreadRoot,
		}
	}

	questionCount := 0
	answerCount := 0
	solutionCount := 0
	acknowledgmentCount := 0

	for _, msg := range normalizedMessages {
		ctx := threadContextMap[msg.ID]
		classifications := classify.ClassifyMessage(msg, ctx)

		if len(classifications) > 0 {
			for _, c := range classifications {
				switch c.Type {
				case "question":
					questionCount++
				case "answer":
					answerCount++
				case "solution":
					solutionCount++
				case "acknowledgment":
					acknowledgmentCount++
				}
			}

			classify.SaveClassifications(msg, classifications)
		}
	}

	// Build reply graph
	replyGraph := graph.BuildFromNormalizedMessages(normalizedMessages)
	if err := graph.SaveReplyGraph(replyGraph); err != nil {
		OutputError("failed to save reply graph: %v", err)
		return err
	}

	graphStats := replyGraph.Stats()

	// Build output
	cacheDir, _ := cache.CacheDir()
	normalizedDir, _ := normalize.NormalizedDir()
	graphDir, _ := graph.GraphDir()
	annotationsDir, _ := classify.AnnotationsDir()

	result := map[string]interface{}{
		"status": "success",
		"repository": map[string]interface{}{
			"owner":       fetchOwner,
			"name":        fetchRepo,
			"full_name":   repo.FullName,
			"description": repo.Description,
			"private":     repo.Private,
		},
		"user": map[string]string{
			"username": authResult.User,
		},
		"issues": map[string]interface{}{
			"count":    issueCount,
			"comments": issueCommentCount,
		},
		"pull_requests": map[string]interface{}{
			"count":    prCount,
			"comments": prCommentCount,
			"reviews":  prReviewCount,
		},
		"messages": map[string]interface{}{
			"normalized": len(normalizedMessages),
			"date_range": map[string]string{
				"from": oldest.Format(time.RFC3339),
				"to":   time.Now().Format(time.RFC3339),
			},
		},
		"classifications": map[string]interface{}{
			"questions":       questionCount,
			"answers":         answerCount,
			"solutions":       solutionCount,
			"acknowledgments": acknowledgmentCount,
		},
		"graph": graphStats,
		"storage": map[string]string{
			"raw":         cacheDir,
			"normalized":  normalizedDir,
			"graph":       graphDir,
			"annotations": annotationsDir,
		},
	}

	return OutputJSON(result)
}
