package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/solvaholic/threadmine/internal/db"
	"github.com/solvaholic/threadmine/internal/github"
	"github.com/solvaholic/threadmine/internal/slack"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Search and retrieve messages from sources",
	Long: `Fetch searches upstream sources (Slack, GitHub, etc.) and stores results locally.

The fetch command uses search APIs to find messages matching criteria, then
retrieves complete threads and stores them in the local database.

Examples:
  # Fetch messages from a Slack user in a channel
  mine fetch slack --user alice --channel general --since 7d

  # Fetch GitHub issues with a label
  mine fetch github --repo org/repo --label bug --since 30d

  # Fetch pull requests reviewed by a user
  mine fetch github --repo org/repo --reviewer bob --type pr`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("please specify a source: slack, github, or email")
	},
}

var fetchSlackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Fetch messages from Slack",
	Long: `Fetch messages from Slack workspaces using search.

This command uses Slack's search API to find messages matching criteria.
Use --threads to also fetch complete threads for messages that are part of threads.
Rate limiting is automatically applied to stay within Slack's API limits
(self-limited to 1/2 of published rates).

Examples:
  # Fetch messages from a user in a channel
  mine fetch slack --workspace myteam --user alice --channel general --since 7d

  # Fetch all messages mentioning a keyword with their threads
  mine fetch slack --workspace myteam --search "kubernetes" --since 30d --threads

  # Fetch messages in a date range
  mine fetch slack --workspace myteam --channel engineering --since 2024-01-01 --until 2024-02-01`,
	RunE: runFetchSlack,
}

var fetchGitHubCmd = &cobra.Command{
	Use:   "github",
	Short: "Fetch issues and pull requests from GitHub",
	Long: `Fetch issues and pull requests from GitHub repositories using search.

This command uses GitHub's search API to find issues/PRs matching criteria,
then retrieves all comments, review comments (for PRs), and timeline events.

Note: 'author' searches for issue/PR authors, 'commenter' searches for comment authors.

Examples:
  # Fetch issues with a label from a specific repo
  mine fetch github --repo org/repo --label bug --since 30d

  # Fetch pull requests by issue author
  mine fetch github --repo org/repo --author alice --type pr --since 7d

  # Fetch issues with comments from a specific user
  mine fetch github --repo org/repo --commenter bob --since 14d

  # Fetch from a repo using separate org and repo flags
  mine fetch github --org myorg --repo myrepo --since 7d

  # Org-wide search (not fully implemented yet)
  mine fetch github --org myorg --search "bug" --since 7d`,
	RunE: runFetchGitHub,
}

var (
	// Common fetch flags
	fetchSince  string
	fetchUntil  string
	fetchLimit  int

	// Slack-specific flags
	slackWorkspace string
	slackUser      string
	slackChannel   string
	slackSearch    string
	slackThreads   bool

	// GitHub-specific flags
	githubOrg       string
	githubRepo      string
	githubAuthor    string
	githubCommenter string
	githubReviewer  string
	githubLabel     string
	githubSearch    string
	githubType      string // issue, pr, or all
)

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.AddCommand(fetchSlackCmd)
	fetchCmd.AddCommand(fetchGitHubCmd)

	// Common flags
	fetchSlackCmd.Flags().StringVar(&fetchSince, "since", "7d", "Start date (YYYY-MM-DD or relative like 7d)")
	fetchSlackCmd.Flags().StringVar(&fetchUntil, "until", "", "End date (YYYY-MM-DD)")
	fetchSlackCmd.Flags().IntVar(&fetchLimit, "limit", 1000, "Maximum number of messages to fetch")

	fetchGitHubCmd.Flags().StringVar(&fetchSince, "since", "7d", "Start date (YYYY-MM-DD or relative like 7d)")
	fetchGitHubCmd.Flags().StringVar(&fetchUntil, "until", "", "End date (YYYY-MM-DD)")
	fetchGitHubCmd.Flags().IntVar(&fetchLimit, "limit", 100, "Maximum number of items to fetch")

	// Slack flags
	fetchSlackCmd.Flags().StringVar(&slackWorkspace, "workspace", "", "Slack workspace/team name (required)")
	fetchSlackCmd.Flags().StringVar(&slackUser, "user", "", "Filter by user (login name or 'me')")
	fetchSlackCmd.Flags().StringVar(&slackChannel, "channel", "", "Filter by channel name")
	fetchSlackCmd.Flags().StringVar(&slackSearch, "search", "", "Search query text")
	fetchSlackCmd.Flags().BoolVar(&slackThreads, "threads", false, "Fetch complete threads for messages that are part of threads")
	fetchSlackCmd.MarkFlagRequired("workspace")

	// GitHub flags
	fetchGitHubCmd.Flags().StringVar(&githubOrg, "org", "", "Organization name (use with --repo for single repo, or alone for org-wide search)")
	fetchGitHubCmd.Flags().StringVar(&githubOrg, "owner", "", "Alias for --org")
	fetchGitHubCmd.Flags().StringVar(&githubRepo, "repo", "", "Repository name (use with --org, or use org/repo format)")
	fetchGitHubCmd.Flags().StringVar(&githubAuthor, "author", "", "Filter by issue/PR author username")
	fetchGitHubCmd.Flags().StringVar(&githubCommenter, "commenter", "", "Filter by comment author username")
	fetchGitHubCmd.Flags().StringVar(&githubReviewer, "reviewer", "", "Filter by PR reviewer (PRs only)")
	fetchGitHubCmd.Flags().StringVar(&githubLabel, "label", "", "Filter by label")
	fetchGitHubCmd.Flags().StringVar(&githubSearch, "search", "", "Search query text")
	fetchGitHubCmd.Flags().StringVar(&githubType, "type", "all", "Type: issue, pr, or all")
	// Note: Either --org or --repo (with org/repo format) is required, validated at runtime
}

func runFetchSlack(cmd *cobra.Command, args []string) error {
	// Open database
	dbPathResolved := dbPath
	if dbPathResolved == "" {
		dbPathResolved = db.DefaultDBPath()
	}

	database, err := db.Open(dbPathResolved)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Parse time range
	since, err := parseTimeSpec(fetchSince)
	if err != nil {
		return fmt.Errorf("invalid --since value: %w", err)
	}

	// Build search query for Slack
	queryParts := []string{}
	if slackUser != "" {
		// Don't add prefix if user already provided it
		if !strings.HasPrefix(slackUser, "@") {
			queryParts = append(queryParts, fmt.Sprintf("from:@%s", slackUser))
		} else {
			queryParts = append(queryParts, fmt.Sprintf("from:%s", slackUser))
		}
	}
	if slackChannel != "" {
		// Handle different channel formats
		if strings.HasPrefix(slackChannel, "C") || strings.HasPrefix(slackChannel, "D") {
			// This is a channel ID - we'll need to look it up
			// For now, use it directly and let Slack handle it
			queryParts = append(queryParts, fmt.Sprintf("in:%s", slackChannel))
		} else if strings.HasPrefix(slackChannel, "@") {
			// User already provided @username for DM
			queryParts = append(queryParts, fmt.Sprintf("in:%s", slackChannel))
		} else if strings.HasPrefix(slackChannel, "#") {
			// User already provided #channel
			queryParts = append(queryParts, fmt.Sprintf("in:%s", slackChannel))
		} else {
			// Bare channel name - check if it matches authenticated user (DM case)
			// For now, assume it's a channel and add #
			queryParts = append(queryParts, fmt.Sprintf("in:#%s", slackChannel))
		}
	}
	if slackSearch != "" {
		queryParts = append(queryParts, slackSearch)
	}
	if fetchSince != "" {
		// For Slack's "after:" to be inclusive of the target date,
		// we need to subtract one more day. E.g., if user wants "since 7d" (past 7 days),
		// we compute 7 days ago, then use "after:" with 8 days ago.
		sinceAdjusted := since.AddDate(0, 0, -1)
		queryParts = append(queryParts, fmt.Sprintf("after:%s", sinceAdjusted.Format("2006-01-02")))
	}
	if fetchUntil != "" {
		until, err := parseTimeSpec(fetchUntil)
		if err != nil {
			return fmt.Errorf("invalid --until value: %w", err)
		}
		// For Slack's "before:" to be inclusive, we need to add one day.
		// E.g., if user wants "until 7d" (up to 7 days ago),
		// we compute 7 days ago, then use "before:" with 6 days ago.
		untilAdjusted := until.AddDate(0, 0, 1)
		queryParts = append(queryParts, fmt.Sprintf("before:%s", untilAdjusted.Format("2006-01-02")))
	}

	if len(queryParts) == 0 {
		return fmt.Errorf("please specify at least one search criterion (--user, --channel, or --search)")
	}

	searchQuery := strings.Join(queryParts, " ")

	fmt.Fprintf(cmd.OutOrStderr(), "Fetching Slack messages with query: %s\n", searchQuery)
	fmt.Fprintf(cmd.OutOrStderr(), "Workspace: %s\n", slackWorkspace)

	// Authenticate with Slack
	fmt.Fprintf(cmd.OutOrStderr(), "Authenticating with Slack...\n")
	authResult, err := slack.Authenticate(slackWorkspace)
	if err != nil {
		return fmt.Errorf("Slack authentication failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStderr(), "Authenticated as %s in %s (Team ID: %s)\n",
		authResult.UserName, authResult.TeamName, authResult.TeamID)

	// Initialize rate limiting for search.messages
	endpoint := "search.messages"
	workspaceID := fmt.Sprintf("ws_slack_%s", authResult.TeamID)
	err = database.InitRateLimit("slack", &workspaceID, endpoint, 60, 20, 10)
	if err != nil {
		return fmt.Errorf("failed to initialize rate limiting: %w", err)
	}

	// Initialize rate limiting for conversations.replies (50/min, self-limit to 25/min)
	err = database.InitRateLimit("slack", &workspaceID, "conversations.replies", 60, 50, 25)
	if err != nil {
		return fmt.Errorf("failed to initialize conversations.replies rate limiting: %w", err)
	}

	// Check rate limit before proceeding
	canProceed, err := database.CheckRateLimit("slack", &workspaceID, endpoint)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	if !canProceed {
		return fmt.Errorf("rate limit exceeded for %s, please wait before retrying", endpoint)
	}

	// Execute search
	fmt.Fprintf(cmd.OutOrStderr(), "Searching Slack messages...\n")
	ctx := context.Background()
	searchResult, err := authResult.Client.SearchMessages(ctx, searchQuery, fetchLimit)
	if err != nil {
		return fmt.Errorf("failed to search messages: %w", err)
	}

	// Record the API call
	database.RecordRequest("slack", &workspaceID, endpoint)

	fmt.Fprintf(cmd.OutOrStderr(), "Found %d matching messages\n", len(searchResult.Messages.Matches))

	// Process each search result
	messageCount := 0
	threadCount := 0
	threadsProcessed := make(map[string]bool)

	for i, result := range searchResult.Messages.Matches {
		fmt.Fprintf(cmd.OutOrStderr(), "Processing message %d/%d...\n", i+1, len(searchResult.Messages.Matches))

		// Extract thread_ts from permalink if not directly available
		threadTS := result.ThreadTS
		if threadTS == "" && result.Permalink != "" {
			if idx := strings.Index(result.Permalink, "?thread_ts="); idx != -1 {
				threadTS = result.Permalink[idx+len("?thread_ts="):]
				// Remove any trailing query params
				if ampIdx := strings.Index(threadTS, "&"); ampIdx != -1 {
					threadTS = threadTS[:ampIdx]
				}
			}
		}

		// Check if this message is part of a thread and if we should fetch threads
		if slackThreads && threadTS != "" && !threadsProcessed[threadTS] {
			// Fetch complete thread
			fmt.Fprintf(cmd.OutOrStderr(), "  Fetching thread %s...\n", threadTS)

			// Check rate limit for conversations.replies
			canProceed, err := database.CheckRateLimit("slack", &workspaceID, "conversations.replies")
			if err != nil {
				return fmt.Errorf("failed to check rate limit: %w", err)
			}
			if !canProceed {
				fmt.Fprintf(cmd.OutOrStderr(), "  Rate limit reached, stopping\n")
				break
			}

			threadMessages, err := authResult.Client.GetThreadReplies(ctx, result.Channel.ID, threadTS)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to fetch thread: %v\n", err)
				// Fall back to storing just this message
				if err := storeSlackMessage(database, result, authResult.TeamID, result.Channel.ID, &result.Channel); err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store message: %v\n", err)
					continue
				}
				messageCount++
			} else {
				// Successfully fetched thread
				database.RecordRequest("slack", &workspaceID, "conversations.replies")

				fmt.Fprintf(cmd.OutOrStderr(), "  Found thread with %d messages\n", len(threadMessages))
				threadCount++

				// Store all messages in thread
				for _, msg := range threadMessages {
					if err := storeSlackMessage(database, msg, authResult.TeamID, result.Channel.ID, &result.Channel); err != nil {
						fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store message: %v\n", err)
						continue
					}
					messageCount++
				}

				threadsProcessed[threadTS] = true
			}
		} else {
			// Either --threads not set, or message not part of a thread, or thread already processed
			// Just store this single message
			if err := storeSlackMessage(database, result, authResult.TeamID, result.Channel.ID, &result.Channel); err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store message: %v\n", err)
				continue
			}
			messageCount++
		}
	}

	fmt.Fprintf(cmd.OutOrStderr(), "\nCompleted!\n")
	fmt.Fprintf(cmd.OutOrStderr(), "Messages stored: %d\n", messageCount)
	fmt.Fprintf(cmd.OutOrStderr(), "Threads processed: %d\n", threadCount)

	return nil
}

// storeSlackMessage stores a Slack message (raw + normalized) in the database
func storeSlackMessage(database *db.DB, msg interface{}, teamID, channelID string, channel *slack.Channel) error {
	// Extract message details based on type
	var msgID, timestamp, userID, username string

	switch m := msg.(type) {
	case slack.SearchResult:
		timestamp = m.Timestamp
		userID = m.User
		username = m.Username
		msgID = fmt.Sprintf("msg_slack_%s_%s", channelID, timestamp)
	case slack.ThreadMessage:
		timestamp = m.Timestamp
		userID = m.User
		username = "" // ThreadMessage doesn't have username field
		msgID = fmt.Sprintf("msg_slack_%s_%s", channelID, timestamp)
	default:
		return fmt.Errorf("unsupported message type: %T", msg)
	}

	// Store user info if we have it
	if userID != "" {
		user := &db.User{
			ID:          fmt.Sprintf("user_slack_%s", userID),
			SourceType:  "slack",
			SourceID:    userID,
			FetchedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if username != "" {
			user.DisplayName = &username
		}
		// Save user (will upsert)
		database.SaveUser(user)
	}

	// Store channel info
	if channel != nil {
		chanName := channel.Name
		displayName := "#" + channel.Name
		chanType := "channel"
		if !channel.IsChannel {
			chanType = "dm"
			displayName = channel.Name // DMs don't get # prefix
		}
		workspaceID := fmt.Sprintf("ws_slack_%s", teamID)

		dbChannel := &db.Channel{
			ID:          fmt.Sprintf("chan_slack_%s", channelID),
			SourceType:  "slack",
			SourceID:    channelID,
			WorkspaceID: &workspaceID,
			Name:        chanName,
			DisplayName: &displayName,
			Type:        &chanType,
			IsPrivate:   channel.IsPrivate,
			ParentSpace: &workspaceID,
			FetchedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		database.SaveChannel(dbChannel)
	}

	// Store raw message
	rawData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal raw message: %w", err)
	}

	workspaceID := fmt.Sprintf("ws_slack_%s", teamID)
	sourceID := fmt.Sprintf("%s_%s", channelID, timestamp)

	err = database.SaveRawMessage(msgID, "slack", sourceID, workspaceID, channelID, string(rawData), "")
	if err != nil {
		return fmt.Errorf("failed to save raw message: %w", err)
	}

	// Normalize and store
	normalized, err := normalizeSlackMessage(msg, teamID, channelID)
	if err != nil {
		return fmt.Errorf("failed to normalize message: %w", err)
	}

	err = database.SaveMessage(normalized)
	if err != nil {
		return fmt.Errorf("failed to save normalized message: %w", err)
	}

	return nil
}

// normalizeSlackMessage converts a Slack message to normalized format
func normalizeSlackMessage(msg interface{}, teamID, channelID string) (*db.Message, error) {
	var timestamp, user, text, threadTS, permalink string

	switch m := msg.(type) {
	case slack.SearchResult:
		timestamp = m.Timestamp
		user = m.User
		text = m.Text
		threadTS = m.ThreadTS
		permalink = m.Permalink
	case slack.ThreadMessage:
		timestamp = m.Timestamp
		user = m.User
		text = m.Text
		threadTS = m.ThreadTS
	default:
		return nil, fmt.Errorf("unsupported message type: %T", msg)
	}

	// If thread_ts is not set but we have a permalink, try to extract it from the permalink
	// Slack search API sometimes omits thread_ts but includes it in the permalink
	// Format: https://workspace.slack.com/archives/CHANNEL/pMSGTS?thread_ts=THREADTS
	if threadTS == "" && permalink != "" {
		if idx := strings.Index(permalink, "?thread_ts="); idx != -1 {
			threadTS = permalink[idx+len("?thread_ts="):]
			// Remove any trailing query params
			if ampIdx := strings.Index(threadTS, "&"); ampIdx != -1 {
				threadTS = threadTS[:ampIdx]
			}
		}
	}

	// Parse timestamp
	ts, err := parseSlackTimestamp(timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Generate universal IDs
	msgID := fmt.Sprintf("msg_slack_%s_%s", channelID, timestamp)
	userID := fmt.Sprintf("user_slack_%s", user)
	chanID := fmt.Sprintf("chan_slack_%s", channelID)

	// Determine thread structure
	var threadID *string
	var parentID *string
	isThreadRoot := false

	if threadTS != "" {
		if threadTS == timestamp {
			// This is the thread root
			isThreadRoot = true
			tid := msgID
			threadID = &tid
		} else {
			// This is a reply
			tid := fmt.Sprintf("msg_slack_%s_%s", channelID, threadTS)
			threadID = &tid
			parentID = &tid
		}
	}

	return &db.Message{
		ID:           msgID,
		SourceType:   "slack",
		SourceID:     fmt.Sprintf("%s_%s", channelID, timestamp),
		Timestamp:    ts,
		AuthorID:     userID,
		Content:      text, // Use the text variable
		ChannelID:    chanID,
		ThreadID:     threadID,
		ParentID:     parentID,
		IsThreadRoot: isThreadRoot,
		Mentions:     []string{},
		URLs:         []string{},
		CodeBlocks:   []db.CodeBlock{},
		Attachments:  []db.Attachment{},
		NormalizedAt: time.Now(),
		SchemaVersion: "2.0",
	}, nil
}

// parseSlackTimestamp converts Slack timestamp to time.Time
func parseSlackTimestamp(ts string) (time.Time, error) {
	var sec, usec int64
	_, err := fmt.Sscanf(ts, "%d.%d", &sec, &usec)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp format: %s", ts)
	}
	return time.Unix(sec, usec*1000), nil
}

func runFetchGitHub(cmd *cobra.Command, args []string) error {
	// Open database
	dbPathResolved := dbPath
	if dbPathResolved == "" {
		dbPathResolved = db.DefaultDBPath()
	}

	database, err := db.Open(dbPathResolved)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Parse time range
	since, err := parseTimeSpec(fetchSince)
	if err != nil {
		return fmt.Errorf("invalid --since value: %w", err)
	}

	// Parse org and repo
	var owner, repo string
	var searchScope string // "repo:owner/repo" or "org:owner"

	if githubRepo != "" {
		// Check if repo contains /
		if strings.Contains(githubRepo, "/") {
			// Format: org/repo
			parts := strings.Split(githubRepo, "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid --repo format: %s (expected org/repo or just repo with --org)", githubRepo)
			}
			owner = parts[0]
			repo = parts[1]
			searchScope = fmt.Sprintf("repo:%s/%s", owner, repo)
		} else {
			// Just repo name, need --org
			if githubOrg == "" {
				return fmt.Errorf("when using --repo with just a repo name, --org is required")
			}
			owner = githubOrg
			repo = githubRepo
			searchScope = fmt.Sprintf("repo:%s/%s", owner, repo)
		}
	} else if githubOrg != "" {
		// Org-wide search
		owner = githubOrg
		repo = "" // No specific repo
		searchScope = fmt.Sprintf("org:%s", owner)
	} else {
		return fmt.Errorf("either --org or --repo is required")
	}

	// Build search query for GitHub
	queryParts := []string{searchScope}

	if githubAuthor != "" {
		queryParts = append(queryParts, fmt.Sprintf("author:%s", githubAuthor))
	}
	if githubCommenter != "" {
		queryParts = append(queryParts, fmt.Sprintf("commenter:%s", githubCommenter))
	}
	if githubLabel != "" {
		queryParts = append(queryParts, fmt.Sprintf("label:%s", githubLabel))
	}
	if githubSearch != "" {
		queryParts = append(queryParts, githubSearch)
	}

	// Add updated date filter
	queryParts = append(queryParts, fmt.Sprintf("updated:>=%s", since.Format("2006-01-02")))

	// Add type filter
	if githubType == "issue" {
		queryParts = append(queryParts, "is:issue")
	} else if githubType == "pr" {
		queryParts = append(queryParts, "is:pr")
	}
	// For githubType == "all", don't add a type filter

	searchQuery := strings.Join(queryParts, " ")

	fmt.Fprintf(cmd.OutOrStderr(), "Fetching GitHub items with query: %s\n", searchQuery)
	if repo != "" {
		fmt.Fprintf(cmd.OutOrStderr(), "Repository: %s/%s\n", owner, repo)
	} else {
		fmt.Fprintf(cmd.OutOrStderr(), "Organization: %s\n", owner)
	}

	// Authenticate with GitHub (via gh CLI)
	fmt.Fprintf(cmd.OutOrStderr(), "Checking GitHub authentication...\n")
	ctx := context.Background()
	authResult, err := github.Authenticate()
	if err != nil {
		return fmt.Errorf("GitHub authentication failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStderr(), "Authenticated as %s\n", authResult.User)

	// Create client for this repo (if specific repo was specified)
	// For org-wide searches, we'll create clients per-issue
	var client *github.Client
	if repo != "" {
		client = github.NewClient(owner, repo)
	}

	// Search for issues/PRs
	fmt.Fprintf(cmd.OutOrStderr(), "Searching GitHub...\n")

	// For org-wide search, we need to search without a specific repo client
	// Use a temporary client just for search
	searchClient := github.NewClient(owner, "")
	results, err := searchClient.SearchIssues(ctx, searchQuery, fetchLimit)
	if err != nil {
		return fmt.Errorf("failed to search GitHub: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStderr(), "Found %d items\n", len(results))

	// Process each result
	messageCount := 0
	orgID := fmt.Sprintf("org_github_%s", owner)

	for i, item := range results {
		// For org-wide search, extract repo info from the issue
		var itemOwner, itemRepo string
		if repo == "" {
			// Org-wide search: extract from Repository field in result
			// The gh search adds repository info to each result
			// For now, we'll need to parse it from the issue data
			// This is a limitation - we'll skip items without repo info
			fmt.Fprintf(cmd.OutOrStderr(), "Warning: org-wide search not fully implemented yet, skipping item #%d\n", item.Number)
			continue
		} else {
			itemOwner = owner
			itemRepo = repo
		}

		fmt.Fprintf(cmd.OutOrStderr(), "Processing item %d/%d: #%d %s\n", i+1, len(results), item.Number, item.Title)

		// Create client for this specific item's repo
		if client == nil {
			client = github.NewClient(itemOwner, itemRepo)
		}

		// Determine if this is an issue or PR
		isPR := githubType == "pr" || (githubType == "all" && strings.Contains(searchQuery, "is:pr"))

		// Store the issue/PR body as a message
		if err := storeGitHubIssue(database, &item, itemOwner, itemRepo, orgID); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store issue: %v\n", err)
			continue
		}
		messageCount++

		// Fetch and store comments
		fmt.Fprintf(cmd.OutOrStderr(), "  Fetching comments...\n")
		comments, err := client.GetIssueComments(ctx, item.Number)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to fetch comments: %v\n", err)
		} else {
			for _, comment := range comments {
				if err := storeGitHubComment(database, &comment, &item, itemOwner, itemRepo, orgID); err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store comment: %v\n", err)
					continue
				}
				messageCount++
			}
		}

		// For PRs, fetch review comments and reviews
		if isPR {
			fmt.Fprintf(cmd.OutOrStderr(), "  Fetching PR review comments...\n")
			reviewComments, err := client.GetPullRequestReviewComments(ctx, item.Number)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to fetch review comments: %v\n", err)
			} else {
				for _, rc := range reviewComments {
					if err := storeGitHubReviewComment(database, &rc, &item, itemOwner, itemRepo, orgID); err != nil {
						fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store review comment: %v\n", err)
						continue
					}
					messageCount++
				}
			}

			fmt.Fprintf(cmd.OutOrStderr(), "  Fetching PR reviews...\n")
			reviews, err := client.GetPullRequestReviews(ctx, item.Number)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to fetch reviews: %v\n", err)
			} else {
				for _, review := range reviews {
					if err := storeGitHubReview(database, &review, &item, itemOwner, itemRepo, orgID); err != nil {
						fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store review: %v\n", err)
						continue
					}
					messageCount++
				}
			}
		}

		// Fetch timeline
		fmt.Fprintf(cmd.OutOrStderr(), "  Fetching timeline...\n")
		timeline, err := client.GetIssueTimeline(ctx, item.Number)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to fetch timeline: %v\n", err)
		} else {
			// Store significant timeline events
			significantCount := 0
			for _, event := range timeline {
				if event.IsSignificant() {
					if err := storeGitHubTimelineEvent(database, &event, &item, itemOwner, itemRepo, orgID); err != nil {
						fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store timeline event: %v\n", err)
						continue
					}
					significantCount++
					messageCount++
				}
			}
			fmt.Fprintf(cmd.OutOrStderr(), "  Found %d timeline events (%d significant stored)\n", len(timeline), significantCount)
		}
	}

	// Search for discussions (only for specific repos, not org-wide)
	if repo != "" {
		fmt.Fprintf(cmd.OutOrStderr(), "\nSearching for discussions...\n")
		discussions, err := client.SearchDiscussions(ctx, searchQuery, fetchLimit)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Warning: failed to search discussions: %v\n", err)
		} else {
			fmt.Fprintf(cmd.OutOrStderr(), "Found %d discussions\n", len(discussions))

			for i, discussion := range discussions {
				fmt.Fprintf(cmd.OutOrStderr(), "Processing discussion %d/%d: #%d %s\n", i+1, len(discussions), discussion.Number, discussion.Title)

				// Store the discussion as a message
				if err := storeGitHubDiscussion(database, &discussion, owner, repo, orgID); err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store discussion: %v\n", err)
					continue
				}
				messageCount++

				// Fetch and store discussion comments and replies
				fmt.Fprintf(cmd.OutOrStderr(), "  Fetching discussion comments...\n")
				comments, err := client.GetDiscussionComments(ctx, discussion.Number)
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to fetch discussion comments: %v\n", err)
				} else {
					for _, comment := range comments {
						if err := storeGitHubDiscussionComment(database, &comment, &discussion, owner, repo, orgID); err != nil {
							fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store discussion comment: %v\n", err)
							continue
						}
						messageCount++
					}
				}
			}
		}
	}

	fmt.Fprintf(cmd.OutOrStderr(), "\nCompleted!\n")
	fmt.Fprintf(cmd.OutOrStderr(), "Messages stored: %d\n", messageCount)

	return nil
}

// storeGitHubIssue stores a GitHub issue/PR as a message
func storeGitHubIssue(database *db.DB, issue *github.Issue, owner, repo, orgID string) error {
	// Store user info
	username := issue.User.Login
	user := &db.User{
		ID:          fmt.Sprintf("user_github_%s", username),
		SourceType:  "github",
		SourceID:    username,
		DisplayName: &username,
		FetchedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.SaveUser(user)

	// Store repo/channel info
	repoName := fmt.Sprintf("%s/%s", owner, repo)
	displayName := repoName
	chanType := "repository"
	dbChannel := &db.Channel{
		ID:          fmt.Sprintf("chan_github_%s_%s", owner, repo),
		SourceType:  "github",
		SourceID:    repoName,
		WorkspaceID: &orgID,
		Name:        repoName,
		DisplayName: &displayName,
		Type:        &chanType,
		IsPrivate:   false,
		ParentSpace: &orgID,
		FetchedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.SaveChannel(dbChannel)

	// Store raw issue
	rawData, err := json.Marshal(issue)
	if err != nil {
		return fmt.Errorf("failed to marshal issue: %w", err)
	}

	msgID := fmt.Sprintf("msg_github_%s_%s_%d", owner, repo, issue.Number)
	sourceID := fmt.Sprintf("%s/%s#%d", owner, repo, issue.Number)

	err = database.SaveRawMessage(msgID, "github", sourceID, orgID, dbChannel.ID, string(rawData), "")
	if err != nil {
		return fmt.Errorf("failed to save raw issue: %w", err)
	}

	// Normalize and store
	normalized := &db.Message{
		ID:           msgID,
		SourceType:   "github",
		SourceID:     sourceID,
		Timestamp:    issue.CreatedAt,
		AuthorID:     user.ID,
		Content:      fmt.Sprintf("%s\n\n%s", issue.Title, issue.Body),
		ChannelID:    dbChannel.ID,
		ThreadID:     &msgID, // Issue is the thread root
		IsThreadRoot: true,
		Mentions:     []string{},
		URLs:         []string{},
		CodeBlocks:   []db.CodeBlock{},
		Attachments:  []db.Attachment{},
		NormalizedAt: time.Now(),
		SchemaVersion: "2.0",
	}

	return database.SaveMessage(normalized)
}

// storeGitHubComment stores a GitHub issue comment
func storeGitHubComment(database *db.DB, comment *github.Comment, issue *github.Issue, owner, repo, orgID string) error {
	// Store user info
	username := comment.User.Login
	user := &db.User{
		ID:          fmt.Sprintf("user_github_%s", username),
		SourceType:  "github",
		SourceID:    username,
		DisplayName: &username,
		FetchedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.SaveUser(user)

	// Store raw comment
	rawData, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal comment: %w", err)
	}

	msgID := fmt.Sprintf("msg_github_%s_%s_%d_comment_%d", owner, repo, issue.Number, comment.ID)
	sourceID := fmt.Sprintf("%s/%s#%d-comment-%d", owner, repo, issue.Number, comment.ID)
	channelID := fmt.Sprintf("chan_github_%s_%s", owner, repo)
	threadID := fmt.Sprintf("msg_github_%s_%s_%d", owner, repo, issue.Number)

	err = database.SaveRawMessage(msgID, "github", sourceID, orgID, channelID, string(rawData), "")
	if err != nil {
		return fmt.Errorf("failed to save raw comment: %w", err)
	}

	// Normalize and store
	normalized := &db.Message{
		ID:           msgID,
		SourceType:   "github",
		SourceID:     sourceID,
		Timestamp:    comment.CreatedAt,
		AuthorID:     user.ID,
		Content:      comment.Body,
		ChannelID:    channelID,
		ThreadID:     &threadID,
		ParentID:     &threadID, // Reply to the issue
		IsThreadRoot: false,
		Mentions:     []string{},
		URLs:         []string{},
		CodeBlocks:   []db.CodeBlock{},
		Attachments:  []db.Attachment{},
		NormalizedAt: time.Now(),
		SchemaVersion: "2.0",
	}

	return database.SaveMessage(normalized)
}

// storeGitHubReviewComment stores a GitHub PR review comment
func storeGitHubReviewComment(database *db.DB, comment *github.ReviewComment, pr *github.Issue, owner, repo, orgID string) error {
	username := comment.User.Login
	user := &db.User{
		ID:          fmt.Sprintf("user_github_%s", username),
		SourceType:  "github",
		SourceID:    username,
		DisplayName: &username,
		FetchedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.SaveUser(user)

	rawData, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal review comment: %w", err)
	}

	msgID := fmt.Sprintf("msg_github_%s_%s_%d_review_comment_%d", owner, repo, pr.Number, comment.ID)
	sourceID := fmt.Sprintf("%s/%s#%d-review-comment-%d", owner, repo, pr.Number, comment.ID)
	channelID := fmt.Sprintf("chan_github_%s_%s", owner, repo)
	threadID := fmt.Sprintf("msg_github_%s_%s_%d", owner, repo, pr.Number)

	err = database.SaveRawMessage(msgID, "github", sourceID, orgID, channelID, string(rawData), "")
	if err != nil {
		return fmt.Errorf("failed to save raw review comment: %w", err)
	}

	// Include file path context in content
	content := fmt.Sprintf("[%s:%d] %s", comment.Path, comment.Line, comment.Body)

	normalized := &db.Message{
		ID:           msgID,
		SourceType:   "github",
		SourceID:     sourceID,
		Timestamp:    comment.CreatedAt,
		AuthorID:     user.ID,
		Content:      content,
		ChannelID:    channelID,
		ThreadID:     &threadID,
		ParentID:     &threadID,
		IsThreadRoot: false,
		Mentions:     []string{},
		URLs:         []string{},
		CodeBlocks:   []db.CodeBlock{},
		Attachments:  []db.Attachment{},
		NormalizedAt: time.Now(),
		SchemaVersion: "2.0",
	}

	return database.SaveMessage(normalized)
}

// storeGitHubReview stores a GitHub PR review
func storeGitHubReview(database *db.DB, review *github.Review, pr *github.Issue, owner, repo, orgID string) error {
	// Skip reviews with no body
	if review.Body == "" {
		return nil
	}

	username := review.User.Login
	user := &db.User{
		ID:          fmt.Sprintf("user_github_%s", username),
		SourceType:  "github",
		SourceID:    username,
		DisplayName: &username,
		FetchedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.SaveUser(user)

	rawData, err := json.Marshal(review)
	if err != nil {
		return fmt.Errorf("failed to marshal review: %w", err)
	}

	msgID := fmt.Sprintf("msg_github_%s_%s_%d_review_%d", owner, repo, pr.Number, review.ID)
	sourceID := fmt.Sprintf("%s/%s#%d-review-%d", owner, repo, pr.Number, review.ID)
	channelID := fmt.Sprintf("chan_github_%s_%s", owner, repo)
	threadID := fmt.Sprintf("msg_github_%s_%s_%d", owner, repo, pr.Number)

	err = database.SaveRawMessage(msgID, "github", sourceID, orgID, channelID, string(rawData), "")
	if err != nil {
		return fmt.Errorf("failed to save raw review: %w", err)
	}

	// Include review state in content
	content := fmt.Sprintf("[%s] %s", review.State, review.Body)

	normalized := &db.Message{
		ID:           msgID,
		SourceType:   "github",
		SourceID:     sourceID,
		Timestamp:    review.SubmittedAt,
		AuthorID:     user.ID,
		Content:      content,
		ChannelID:    channelID,
		ThreadID:     &threadID,
		ParentID:     &threadID,
		IsThreadRoot: false,
		Mentions:     []string{},
		URLs:         []string{},
		CodeBlocks:   []db.CodeBlock{},
		Attachments:  []db.Attachment{},
		NormalizedAt: time.Now(),
		SchemaVersion: "2.0",
	}

	return database.SaveMessage(normalized)
}

// storeGitHubDiscussion stores a GitHub discussion as a message
func storeGitHubDiscussion(database *db.DB, discussion *github.Discussion, owner, repo, orgID string) error {
	username := discussion.Author.Login
	user := &db.User{
		ID:          fmt.Sprintf("user_github_%s", username),
		SourceType:  "github",
		SourceID:    username,
		DisplayName: &username,
		FetchedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.SaveUser(user)

	// Store repo/channel info
	repoName := fmt.Sprintf("%s/%s", owner, repo)
	displayName := repoName
	chanType := "repository"
	dbChannel := &db.Channel{
		ID:          fmt.Sprintf("chan_github_%s_%s", owner, repo),
		SourceType:  "github",
		SourceID:    repoName,
		WorkspaceID: &orgID,
		Name:        repoName,
		DisplayName: &displayName,
		Type:        &chanType,
		FetchedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.SaveChannel(dbChannel)

	rawData, err := json.Marshal(discussion)
	if err != nil {
		return fmt.Errorf("failed to marshal discussion: %w", err)
	}

	msgID := fmt.Sprintf("msg_github_%s_%s_discussion_%d", owner, repo, discussion.Number)
	sourceID := fmt.Sprintf("%s/%s/discussions/%d", owner, repo, discussion.Number)
	channelID := fmt.Sprintf("chan_github_%s_%s", owner, repo)
	threadID := msgID // Discussion is its own thread root

	err = database.SaveRawMessage(msgID, "github", sourceID, orgID, channelID, string(rawData), "")
	if err != nil {
		return fmt.Errorf("failed to save raw discussion: %w", err)
	}

	// Include category in content
	content := discussion.Body
	if discussion.Category.Name != "" {
		content = fmt.Sprintf("[%s] %s", discussion.Category.Name, content)
	}

	normalized := &db.Message{
		ID:            msgID,
		SourceType:    "github",
		SourceID:      sourceID,
		Timestamp:     discussion.CreatedAt,
		AuthorID:      user.ID,
		Content:       content,
		ChannelID:     channelID,
		ThreadID:      &threadID,
		ParentID:      nil, // No parent, this is the root
		IsThreadRoot:  true,
		Mentions:      []string{},
		URLs:          []string{},
		CodeBlocks:    []db.CodeBlock{},
		Attachments:   []db.Attachment{},
		NormalizedAt:  time.Now(),
		SchemaVersion: "2.0",
	}

	return database.SaveMessage(normalized)
}

// storeGitHubDiscussionComment stores a discussion comment or reply as a message
func storeGitHubDiscussionComment(database *db.DB, comment *github.DiscussionComment, discussion *github.Discussion, owner, repo, orgID string) error {
	username := comment.Author.Login
	user := &db.User{
		ID:          fmt.Sprintf("user_github_%s", username),
		SourceType:  "github",
		SourceID:    username,
		DisplayName: &username,
		FetchedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.SaveUser(user)

	rawData, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal discussion comment: %w", err)
	}

	msgID := fmt.Sprintf("msg_github_%s_%s_discussion_%d_comment_%s", owner, repo, discussion.Number, comment.ID)
	sourceID := fmt.Sprintf("%s/%s/discussions/%d#%s", owner, repo, discussion.Number, comment.ID)
	channelID := fmt.Sprintf("chan_github_%s_%s", owner, repo)
	threadID := fmt.Sprintf("msg_github_%s_%s_discussion_%d", owner, repo, discussion.Number)

	err = database.SaveRawMessage(msgID, "github", sourceID, orgID, channelID, string(rawData), "")
	if err != nil {
		return fmt.Errorf("failed to save raw discussion comment: %w", err)
	}

	normalized := &db.Message{
		ID:            msgID,
		SourceType:    "github",
		SourceID:      sourceID,
		Timestamp:     comment.CreatedAt,
		AuthorID:      user.ID,
		Content:       comment.Body,
		ChannelID:     channelID,
		ThreadID:      &threadID,
		ParentID:      &threadID, // All comments point to discussion as parent
		IsThreadRoot:  false,
		Mentions:      []string{},
		URLs:          []string{},
		CodeBlocks:    []db.CodeBlock{},
		Attachments:   []db.Attachment{},
		NormalizedAt:  time.Now(),
		SchemaVersion: "2.0",
	}

	return database.SaveMessage(normalized)
}

// storeGitHubTimelineEvent stores a significant timeline event as a message
func storeGitHubTimelineEvent(database *db.DB, event *github.TimelineEvent, issue *github.Issue, owner, repo, orgID string) error {
	username := event.Actor.Login
	user := &db.User{
		ID:          fmt.Sprintf("user_github_%s", username),
		SourceType:  "github",
		SourceID:    username,
		DisplayName: &username,
		FetchedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	database.SaveUser(user)

	rawData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal timeline event: %w", err)
	}

	msgID := fmt.Sprintf("msg_github_%s_%s_%d_timeline_%d", owner, repo, issue.Number, event.ID)
	sourceID := fmt.Sprintf("%s/%s#%d-event-%d", owner, repo, issue.Number, event.ID)
	channelID := fmt.Sprintf("chan_github_%s_%s", owner, repo)
	threadID := fmt.Sprintf("msg_github_%s_%s_%d", owner, repo, issue.Number)

	err = database.SaveRawMessage(msgID, "github", sourceID, orgID, channelID, string(rawData), "")
	if err != nil {
		return fmt.Errorf("failed to save raw timeline event: %w", err)
	}

	// Build content based on event type
	var content string
	if event.Body != "" {
		// Cross-reference with body
		content = fmt.Sprintf("[%s] %s", event.Event, event.Body)
	} else {
		// State change event
		content = fmt.Sprintf("[%s] Issue %s", event.Event, event.Event)

		// Add details for specific event types
		if event.Label != nil {
			content = fmt.Sprintf("[%s] Label: %s", event.Event, event.Label.Name)
		} else if event.Assignee != nil {
			content = fmt.Sprintf("[%s] Assignee: %s", event.Event, event.Assignee.Login)
		} else if event.CommitID != "" {
			content = fmt.Sprintf("[%s] Commit: %s", event.Event, event.CommitID[:7])
		}
	}

	normalized := &db.Message{
		ID:            msgID,
		SourceType:    "github",
		SourceID:      sourceID,
		Timestamp:     event.CreatedAt,
		AuthorID:      user.ID,
		Content:       content,
		ChannelID:     channelID,
		ThreadID:      &threadID,
		ParentID:      &threadID,
		IsThreadRoot:  false,
		Mentions:      []string{},
		URLs:          []string{},
		CodeBlocks:    []db.CodeBlock{},
		Attachments:   []db.Attachment{},
		NormalizedAt:  time.Now(),
		SchemaVersion: "2.0",
	}

	return database.SaveMessage(normalized)
}
