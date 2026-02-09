package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/solvaholic/threadmine/internal/db"
	"github.com/spf13/cobra"
)

var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Query and analyze cached messages",
	Long: `Select queries the local database for messages matching search criteria.

Examples:
  # Select threads involving a user since a date
  mine select --author alice --since 7d

  # Select threads mentioning a keyword
  mine select --search "kubernetes"

  # Select threads with multiple participants
  mine select --author alice --author bob --author charlie

Output formats:
  - json: Normalized messages with annotations (default, for tools)
  - jsonl: One message per line (for streaming/piping)
  - table: Human-readable table
  - graph: Graph format for visualization tools`,
	RunE: runSelect,
}

var (
	selectAuthors  []string
	selectChannels []string
	selectSources  []string
	selectSearch   string
	selectSince    string
	selectUntil    string
	selectThreadID string
	selectLimit    int
	selectOffset   int
)

func init() {
	rootCmd.AddCommand(selectCmd)

	selectCmd.Flags().StringSliceVar(&selectAuthors, "author", nil, "Filter by author (can be repeated)")
	selectCmd.Flags().StringSliceVar(&selectChannels, "channel", nil, "Filter by channel (can be repeated)")
	selectCmd.Flags().StringSliceVar(&selectSources, "source", nil, "Filter by source type: slack, github, email")
	selectCmd.Flags().StringVar(&selectSearch, "search", "", "Full-text search query")
	selectCmd.Flags().StringVar(&selectSince, "since", "", "Start date (YYYY-MM-DD or relative like 7d)")
	selectCmd.Flags().StringVar(&selectUntil, "until", "", "End date (YYYY-MM-DD)")
	selectCmd.Flags().StringVar(&selectThreadID, "thread", "", "Filter by thread ID")
	selectCmd.Flags().IntVar(&selectLimit, "limit", 100, "Maximum number of results")
	selectCmd.Flags().IntVar(&selectOffset, "offset", 0, "Offset for pagination")
}

func runSelect(cmd *cobra.Command, args []string) error {
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

	// Build query options
	opts := db.SelectMessagesOptions{
		Limit:  selectLimit,
		Offset: selectOffset,
	}

	// Parse since/until dates
	if selectSince != "" {
		since, err := parseTimeSpec(selectSince)
		if err != nil {
			return fmt.Errorf("invalid --since value: %w", err)
		}
		opts.Since = &since
	}

	if selectUntil != "" {
		until, err := parseTimeSpec(selectUntil)
		if err != nil {
			return fmt.Errorf("invalid --until value: %w", err)
		}
		opts.Until = &until
	}

	// Handle source filter
	if len(selectSources) > 0 {
		// For now, just use the first source
		// TODO: Support multiple sources with OR logic
		opts.SourceType = &selectSources[0]
	}

	// Handle author filter
	if len(selectAuthors) > 0 {
		// Look up author by name to get user ID
		// For now, just use the first author
		// TODO: Support multiple authors
		authorName := selectAuthors[0]
		users, err := database.FindUsersByName(authorName)
		if err != nil {
			return fmt.Errorf("failed to find user '%s': %w", authorName, err)
		}
		if len(users) == 0 {
			return fmt.Errorf("no user found with name '%s'", authorName)
		}
		// If multiple users found, use the first one
		// TODO: Let user disambiguate if multiple matches
		opts.AuthorID = &users[0].ID
	}

	// Handle channel filter
	if len(selectChannels) > 0 {
		// Look up channel by name to get channel ID
		channelName := selectChannels[0]
		channels, err := database.FindChannelsByName(channelName)
		if err != nil {
			return fmt.Errorf("failed to find channel '%s': %w", channelName, err)
		}
		if len(channels) == 0 {
			return fmt.Errorf("no channel found with name '%s'", channelName)
		}
		// If multiple channels found, use the first one
		// TODO: Let user disambiguate if multiple matches
		opts.ChannelID = &channels[0].ID
	}

	// Handle thread filter
	if selectThreadID != "" {
		opts.ThreadID = &selectThreadID
	}

	// Handle search
	if selectSearch != "" {
		opts.SearchText = &selectSearch
	}

	// Execute query
	messages, err := database.SelectMessages(opts)
	if err != nil {
		return fmt.Errorf("failed to select messages: %w", err)
	}

	// Output results
	switch outputFormat {
	case "json":
		return OutputJSON(messages)
	case "jsonl":
		return outputJSONL(messages)
	case "table":
		return outputTable(messages)
	case "graph":
		return outputGraph(messages)
	default:
		return fmt.Errorf("unknown format: %s", outputFormat)
	}
}

func outputJSONL(messages []*db.Message) error {
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}
		fmt.Println(string(data))
	}
	return nil
}

func outputTable(messages []*db.Message) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "TIMESTAMP\tAUTHOR\tCHANNEL\tCONTENT\n")
	fmt.Fprintf(w, "---------\t------\t-------\t-------\n")

	// Open database to look up names
	dbPathResolved := dbPath
	if dbPathResolved == "" {
		dbPathResolved = db.DefaultDBPath()
	}
	database, err := db.Open(dbPathResolved)
	if err != nil {
		// If we can't open database, fall back to showing IDs
		for _, msg := range messages {
			content := msg.Content
			if len(content) > 60 {
				content = content[:57] + "..."
			}
			content = strings.ReplaceAll(content, "\n", " ")

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				msg.Timestamp.Format("2006-01-02 15:04"),
				msg.AuthorID,
				msg.ChannelID,
				content,
			)
		}
		return nil
	}
	defer database.Close()

	// Cache for looked-up names
	userNames := make(map[string]string)
	channelNames := make(map[string]string)

	for _, msg := range messages {
		// Look up author name
		authorName := msg.AuthorID
		if cached, ok := userNames[msg.AuthorID]; ok {
			authorName = cached
		} else {
			user, err := database.GetUser(msg.AuthorID)
			if err == nil && user != nil {
				if user.DisplayName != nil && *user.DisplayName != "" {
					authorName = *user.DisplayName
				} else if user.RealName != nil && *user.RealName != "" {
					authorName = *user.RealName
				}
				userNames[msg.AuthorID] = authorName
			}
		}

		// Look up channel name
		channelName := msg.ChannelID
		if cached, ok := channelNames[msg.ChannelID]; ok {
			channelName = cached
		} else {
			channel, err := database.GetChannel(msg.ChannelID)
			if err == nil && channel != nil {
				if channel.DisplayName != nil && *channel.DisplayName != "" {
					channelName = *channel.DisplayName
				} else {
					channelName = channel.Name
				}
				channelNames[msg.ChannelID] = channelName
			}
		}

		// Truncate content for display
		content := msg.Content
		if len(content) > 60 {
			content = content[:57] + "..."
		}
		content = strings.ReplaceAll(content, "\n", " ")

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			msg.Timestamp.Format("2006-01-02 15:04"),
			authorName,
			channelName,
			content,
		)
	}

	return nil
}

func outputGraph(messages []*db.Message) error {
	// Simple graph format: nodes and edges
	type Node struct {
		ID      string    `json:"id"`
		Type    string    `json:"type"`
		Content string    `json:"content"`
		Time    time.Time `json:"timestamp"`
	}

	type Edge struct {
		From string `json:"from"`
		To   string `json:"to"`
		Type string `json:"type"`
	}

	type Graph struct {
		Nodes []Node `json:"nodes"`
		Edges []Edge `json:"edges"`
	}

	graph := Graph{
		Nodes: make([]Node, 0, len(messages)),
		Edges: make([]Edge, 0),
	}

	// Add message nodes
	for _, msg := range messages {
		graph.Nodes = append(graph.Nodes, Node{
			ID:      msg.ID,
			Type:    "message",
			Content: msg.Content,
			Time:    msg.Timestamp,
		})

		// Add reply edges
		if msg.ParentID != nil && *msg.ParentID != "" {
			graph.Edges = append(graph.Edges, Edge{
				From: msg.ID,
				To:   *msg.ParentID,
				Type: "reply_to",
			})
		}
	}

	return OutputJSON(graph)
}

// parseTimeSpec parses time specifications like "7d", "2024-01-01", "3w"
func parseTimeSpec(spec string) (time.Time, error) {
	// Try parsing as RFC3339 or common date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, spec); err == nil {
			return t, nil
		}
	}

	// Try parsing as relative time (e.g., "7d", "3w", "2h")
	if len(spec) < 2 {
		return time.Time{}, fmt.Errorf("invalid time specification: %s", spec)
	}

	unit := spec[len(spec)-1]
	valueStr := spec[:len(spec)-1]

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return time.Time{}, fmt.Errorf("invalid time value: %s", spec)
	}

	now := time.Now()
	switch unit {
	case 'h':
		return now.Add(-time.Duration(value) * time.Hour), nil
	case 'd':
		return now.AddDate(0, 0, -value), nil
	case 'w':
		return now.AddDate(0, 0, -value*7), nil
	case 'm':
		return now.AddDate(0, -value, 0), nil
	case 'y':
		return now.AddDate(-value, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unknown time unit: %c (use h, d, w, m, y)", unit)
	}
}
