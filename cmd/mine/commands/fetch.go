package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/solvaholic/threadmine/internal/cache"
	"github.com/solvaholic/threadmine/internal/classify"
	"github.com/solvaholic/threadmine/internal/graph"
	"github.com/solvaholic/threadmine/internal/normalize"
	"github.com/solvaholic/threadmine/internal/slack"
	"github.com/solvaholic/threadmine/internal/utils"
)

var (
	fetchWorkspace string
	fetchChannel   string
	fetchSince     string
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

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.AddCommand(fetchSlackCmd)

	// Flags for fetch slack
	fetchSlackCmd.Flags().StringVarP(&fetchWorkspace, "workspace", "w", "", "Slack workspace name (required)")
	fetchSlackCmd.Flags().StringVarP(&fetchChannel, "channel", "c", "", "Channel ID to fetch (default: first available)")
	fetchSlackCmd.Flags().StringVarP(&fetchSince, "since", "s", "7d", "Fetch messages since (e.g., '7d', '2025-12-15')")
	fetchSlackCmd.MarkFlagRequired("workspace")
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

	// Authenticate with Slack
	result, err := slack.Authenticate(fetchWorkspace)
	if err != nil {
		OutputError("authentication failed: %v", err)
		return err
	}

	// List channels
	channels, err := result.Client.ListChannels(ctx)
	if err != nil {
		OutputError("failed to list channels: %v", err)
		return err
	}

	// Cache the channels list
	if err := cache.SaveChannelsList(result.TeamID, channels); err != nil {
		OutputError("failed to cache channels: %v", err)
		return err
	}

	if len(channels) == 0 {
		OutputError("no channels found")
		return fmt.Errorf("no channels found")
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
			OutputError("channel not found: %s", fetchChannel)
			return fmt.Errorf("channel not found")
		}
	} else {
		targetChannel = &channels[0]
	}

	// Save channel info
	if err := cache.SaveChannelInfo(result.TeamID, targetChannel.ID, *targetChannel); err != nil {
		OutputError("failed to cache channel info: %v", err)
		return err
	}

	// Get messages using cache-aside pattern
	cacheDir, _ := cache.CacheDir()
	messages, err := result.Client.GetMessages(ctx, targetChannel.ID, oldest, cacheDir)
	if err != nil {
		OutputError("failed to retrieve messages: %v", err)
		return err
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
		OutputError("failed to save reply graph: %v", err)
		return err
	}

	graphStats := replyGraph.Stats()
	normalizedDir, _ := normalize.NormalizedDir()
	graphDir, _ := graph.GraphDir()
	annotationsDir, _ := classify.AnnotationsDir()

	// Build output
	output := map[string]interface{}{
		"status": "success",
		"workspace": map[string]string{
			"name": result.TeamName,
			"id":   result.TeamID,
		},
		"user": map[string]string{
			"name": result.UserName,
			"id":   result.UserID,
		},
		"channel": map[string]interface{}{
			"id":   targetChannel.ID,
			"name": targetChannel.Name,
		},
		"messages": map[string]interface{}{
			"fetched":    len(messages),
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

	return OutputJSON(output)
}
