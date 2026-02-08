package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/solvaholic/threadmine/internal/db"
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

This command uses Slack's search API to find messages matching criteria,
then retrieves complete threads. Rate limiting is automatically applied
to stay within Slack's API limits (self-limited to 1/2 of published rates).

Examples:
  # Fetch messages from a user in a channel
  mine fetch slack --workspace myteam --user alice --channel general --since 7d

  # Fetch all messages mentioning a keyword
  mine fetch slack --workspace myteam --search "kubernetes" --since 30d

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

Examples:
  # Fetch issues with a label
  mine fetch github --repo org/repo --label bug --since 30d

  # Fetch pull requests by author
  mine fetch github --repo org/repo --author alice --type pr --since 7d

  # Fetch issues mentioning a keyword
  mine fetch github --repo org/repo --search "authentication" --since 90d`,
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

	// GitHub-specific flags
	githubRepo     string
	githubAuthor   string
	githubReviewer string
	githubLabel    string
	githubSearch   string
	githubType     string // issue, pr, or all
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
	fetchSlackCmd.MarkFlagRequired("workspace")

	// GitHub flags
	fetchGitHubCmd.Flags().StringVar(&githubRepo, "repo", "", "Repository (org/repo format, required)")
	fetchGitHubCmd.Flags().StringVar(&githubAuthor, "author", "", "Filter by author username")
	fetchGitHubCmd.Flags().StringVar(&githubReviewer, "reviewer", "", "Filter by PR reviewer (PRs only)")
	fetchGitHubCmd.Flags().StringVar(&githubLabel, "label", "", "Filter by label")
	fetchGitHubCmd.Flags().StringVar(&githubSearch, "search", "", "Search query text")
	fetchGitHubCmd.Flags().StringVar(&githubType, "type", "all", "Type: issue, pr, or all")
	fetchGitHubCmd.MarkFlagRequired("repo")
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
		queryParts = append(queryParts, fmt.Sprintf("after:%s", since.Format("2006-01-02")))
	}
	if fetchUntil != "" {
		until, err := parseTimeSpec(fetchUntil)
		if err != nil {
			return fmt.Errorf("invalid --until value: %w", err)
		}
		queryParts = append(queryParts, fmt.Sprintf("before:%s", until.Format("2006-01-02")))
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

	// Initialize rate limiting
	endpoint := "search.messages"
	workspaceID := fmt.Sprintf("ws_slack_%s", authResult.TeamID)
	err = database.InitRateLimit("slack", &workspaceID, endpoint, 60, 20, 10)
	if err != nil {
		return fmt.Errorf("failed to initialize rate limiting: %w", err)
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

		// Check if this message is part of a thread
		if result.ThreadTS != "" && !threadsProcessed[result.ThreadTS] {
			// Fetch complete thread
			fmt.Fprintf(cmd.OutOrStderr(), "  Fetching thread %s...\n", result.ThreadTS)

			// Check rate limit for conversations.replies
			canProceed, err := database.CheckRateLimit("slack", &workspaceID, "conversations.replies")
			if err != nil {
				return fmt.Errorf("failed to check rate limit: %w", err)
			}
			if !canProceed {
				fmt.Fprintf(cmd.OutOrStderr(), "  Rate limit reached, stopping\n")
				break
			}

			threadMessages, err := authResult.Client.GetThreadReplies(ctx, result.Channel.ID, result.ThreadTS)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to fetch thread: %v\n", err)
				continue
			}

			database.RecordRequest("slack", &workspaceID, "conversations.replies")

			// Store all messages in thread
			for _, msg := range threadMessages {
				if err := storeSlackMessage(database, msg, authResult.TeamID, result.Channel.ID, &result.Channel); err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "  Warning: failed to store message: %v\n", err)
					continue
				}
				messageCount++
			}

			threadsProcessed[result.ThreadTS] = true
			threadCount++
		} else if result.ThreadTS == "" {
			// Single message, not part of a thread
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
	var timestamp, user, text, threadTS string

	switch m := msg.(type) {
	case slack.SearchResult:
		timestamp = m.Timestamp
		user = m.User
		text = m.Text
		threadTS = m.ThreadTS
	case slack.ThreadMessage:
		timestamp = m.Timestamp
		user = m.User
		text = m.Text
		threadTS = m.ThreadTS
	default:
		return nil, fmt.Errorf("unsupported message type: %T", msg)
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

	// Parse repo (org/repo format)
	parts := strings.Split(githubRepo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s (expected org/repo)", githubRepo)
	}

	// Build search query for GitHub
	queryParts := []string{fmt.Sprintf("repo:%s", githubRepo)}

	if githubAuthor != "" {
		queryParts = append(queryParts, fmt.Sprintf("author:%s", githubAuthor))
	}
	if githubReviewer != "" && (githubType == "pr" || githubType == "all") {
		queryParts = append(queryParts, fmt.Sprintf("reviewed-by:%s", githubReviewer))
	}
	if githubLabel != "" {
		queryParts = append(queryParts, fmt.Sprintf("label:%s", githubLabel))
	}
	if githubSearch != "" {
		queryParts = append(queryParts, githubSearch)
	}

	// Add type filter
	if githubType == "issue" {
		queryParts = append(queryParts, "is:issue")
	} else if githubType == "pr" {
		queryParts = append(queryParts, "is:pr")
	}

	searchQuery := strings.Join(queryParts, " ")

	fmt.Fprintf(cmd.OutOrStderr(), "Fetching GitHub items with query: %s\n", searchQuery)
	fmt.Fprintf(cmd.OutOrStderr(), "Since: %s\n", since.Format("2006-01-02"))
	fmt.Fprintf(cmd.OutOrStderr(), "Note: GitHub search integration not yet fully implemented\n")
	fmt.Fprintf(cmd.OutOrStderr(), "TODO: Implement GitHub search with complete data fetching\n")

	// TODO: Implement GitHub search API integration
	// 1. Authenticate with GitHub (via gh CLI)
	// 2. Execute search query
	// 3. For each issue/PR:
	//    - Fetch all comments
	//    - Fetch all review comments (PRs only)
	//    - Fetch timeline events
	// 4. For discussions (if requested):
	//    - Fetch all comments and replies
	// 5. Store raw data in database
	// 6. Normalize and store in messages table

	ctx := context.Background()
	_ = ctx // Will be used for API calls

	return fmt.Errorf("GitHub fetching not yet fully implemented - coming soon")
}
