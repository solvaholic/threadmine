package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/solvaholic/threadmine/internal/normalize"
	"github.com/solvaholic/threadmine/internal/utils"
	"github.com/spf13/cobra"
)

var (
	msgAuthor    string
	msgChannel   string
	msgWorkspace string
	msgSince     string
	msgUntil     string
	msgSource    string
	msgSearch    string
)

// messagesCmd represents the messages command
var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Query messages",
	Long:  `Query normalized messages across all sources with filtering options.`,
	RunE:  runMessages,
}

func init() {
	rootCmd.AddCommand(messagesCmd)

	messagesCmd.Flags().StringVarP(&msgAuthor, "author", "a", "", "Filter by author ID")
	messagesCmd.Flags().StringVarP(&msgChannel, "channel", "c", "", "Filter by channel/issue")
	messagesCmd.Flags().StringVarP(&msgWorkspace, "workspace", "w", "", "Filter by workspace/team ID")
	messagesCmd.Flags().StringVarP(&msgSince, "since", "s", "", "Start date (e.g., '7d', '2025-12-15')")
	messagesCmd.Flags().StringVarP(&msgUntil, "until", "u", "", "End date (e.g., '7d', '2025-12-15')")
	messagesCmd.Flags().StringVar(&msgSource, "source", "", "Filter by source (slack, github, email)")
	messagesCmd.Flags().StringVar(&msgSearch, "search", "", "Search text in message content")
}

func runMessages(cmd *cobra.Command, args []string) error {
	normalizedDir, err := normalize.NormalizedDir()
	if err != nil {
		OutputError("failed to get normalized directory: %v", err)
		return err
	}

	// Parse date filters using shared utility
	var sinceTime, untilTime time.Time
	if msgSince != "" {
		sinceTime, err = utils.ParseSinceDate(msgSince)
		if err != nil {
			OutputError("invalid since date format: %v", err)
			return err
		}
	}
	if msgUntil != "" {
		untilTime, err = utils.ParseSinceDate(msgUntil)
		if err != nil {
			OutputError("invalid until date format: %v", err)
			return err
		}
	}

	// Read messages from normalized storage
	var messages []*normalize.NormalizedMessage

	// Read from by_source files (most efficient for full scans)
	messagesDir := filepath.Join(normalizedDir, "messages", "by_source")

	sources := []string{"slack.jsonl"}
	if msgSource != "" {
		sources = []string{msgSource + ".jsonl"}
	}

	for _, sourceFile := range sources {
		sourcePath := filepath.Join(messagesDir, sourceFile)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(sourcePath)
		if err != nil {
			continue
		}

		// Parse JSONL
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			var msg normalize.NormalizedMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}

			// Apply filters
			if msgAuthor != "" && msg.Author.ID != msgAuthor {
				continue
			}
			if msgChannel != "" && msg.Channel.ID != msgChannel {
				continue
			}
			if msgWorkspace != "" && msg.Channel.ParentSpace != msgWorkspace {
				continue
			}
			if !sinceTime.IsZero() && msg.Timestamp.Before(sinceTime) {
				continue
			}
			if !untilTime.IsZero() && msg.Timestamp.After(untilTime) {
				continue
			}
			if msgSearch != "" && !strings.Contains(strings.ToLower(msg.Content), strings.ToLower(msgSearch)) {
				continue
			}

			messages = append(messages, &msg)
		}
	}

	// Build output
	output := map[string]interface{}{
		"status":        "success",
		"message_count": len(messages),
		"filters": map[string]interface{}{
			"author":    msgAuthor,
			"channel":   msgChannel,
			"workspace": msgWorkspace,
			"since":     msgSince,
			"until":     msgUntil,
			"source":    msgSource,
			"search":    msgSearch,
		},
		"messages": messages,
	}

	return OutputJSON(output)
}
