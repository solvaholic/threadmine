package normalize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SlackMessage represents the raw Slack message structure
type SlackMessage struct {
	Type      string                 `json:"type"`
	User      string                 `json:"user"`
	Text      string                 `json:"text"`
	Timestamp string                 `json:"ts"`
	ThreadTS  string                 `json:"thread_ts,omitempty"`
	BotID     string                 `json:"bot_id,omitempty"`
	Subtype   string                 `json:"subtype,omitempty"`
	Files     []map[string]interface{} `json:"files,omitempty"`
	Metadata  map[string]interface{} `json:"-"` // Catch-all for other fields
}

// SlackChannel represents the raw Slack channel structure
type SlackChannel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsChannel bool   `json:"is_channel"`
	IsPrivate bool   `json:"is_private"`
}

// SlackUser represents the raw Slack user structure
type SlackUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Profile  struct {
		Email     string `json:"email"`
		Image192  string `json:"image_192"`
	} `json:"profile"`
}

var (
	// Regex patterns for parsing Slack markup
	userMentionPattern = regexp.MustCompile(`<@([A-Z0-9]+)(\|([^>]+))?>`)
	channelPattern     = regexp.MustCompile(`<#([A-Z0-9]+)(\|([^>]+))?>`)
	urlPattern         = regexp.MustCompile(`<(https?://[^|>]+)(\|([^>]+))?>`)
	codeBlockPattern   = regexp.MustCompile("```([a-z]*)\n([^`]+)```")
)

// SlackToNormalized converts a Slack message to the normalized schema
func SlackToNormalized(msg *SlackMessage, channel *SlackChannel, user *SlackUser, teamID string, fetchedAt time.Time) (*NormalizedMessage, error) {
	// Parse timestamp
	ts, err := parseSlackTimestamp(msg.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp %s: %w", msg.Timestamp, err)
	}

	// Generate universal ID
	msgID := fmt.Sprintf("msg_slack_%s_%s_%s", teamID, channel.ID, msg.Timestamp)

	// Parse thread context
	threadID := ""
	parentID := ""
	isThreadRoot := msg.ThreadTS == "" || msg.ThreadTS == msg.Timestamp
	
	if msg.ThreadTS != "" {
		threadID = fmt.Sprintf("thread_slack_%s_%s_%s", teamID, channel.ID, msg.ThreadTS)
		if !isThreadRoot {
			parentID = fmt.Sprintf("msg_slack_%s_%s_%s", teamID, channel.ID, msg.ThreadTS)
		}
	}

	// Extract mentions, URLs, and code blocks
	mentions := extractMentions(msg.Text)
	urls := extractURLs(msg.Text)
	codeBlocks := extractCodeBlocks(msg.Text)

	// Convert text to normalized format (remove Slack markup)
	normalizedText := normalizeSlackText(msg.Text)

	// Convert attachments
	attachments := convertSlackAttachments(msg.Files)

	// Build normalized message
	normalized := &NormalizedMessage{
		ID:         msgID,
		SourceType: "slack",
		SourceID:   fmt.Sprintf("%s:%s:%s", teamID, channel.ID, msg.Timestamp),
		Timestamp:  ts,
		Author:     convertSlackUser(user, teamID),
		Content:    normalizedText,
		ContentHTML: "", // Slack doesn't provide HTML
		Channel:    convertSlackChannel(channel, teamID),
		ThreadID:   threadID,
		ParentID:   parentID,
		IsThreadRoot: isThreadRoot,
		Attachments: attachments,
		Mentions:   mentions,
		URLs:       urls,
		CodeBlocks: codeBlocks,
		SourceMetadata: map[string]interface{}{
			"team_id": teamID,
			"channel_id": channel.ID,
			"ts": msg.Timestamp,
			"thread_ts": msg.ThreadTS,
			"type": msg.Type,
			"subtype": msg.Subtype,
			"bot_id": msg.BotID,
		},
		FetchedAt:    fetchedAt,
		NormalizedAt: time.Now(),
		SchemaVersion: SchemaVersion,
	}

	return normalized, nil
}

// parseSlackTimestamp converts Slack timestamp format to time.Time
func parseSlackTimestamp(ts string) (time.Time, error) {
	// Slack timestamps are in format "1234567890.123456"
	f, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return time.Time{}, err
	}
	sec := int64(f)
	nsec := int64((f - float64(sec)) * 1e9)
	return time.Unix(sec, nsec), nil
}

// convertSlackUser converts a Slack user to the normalized User schema
func convertSlackUser(user *SlackUser, teamID string) *User {
	if user == nil {
		return nil
	}
	return &User{
		ID:          fmt.Sprintf("user_slack_%s_%s", teamID, user.ID),
		SourceType:  "slack",
		SourceID:    user.ID,
		DisplayName: user.Name,
		RealName:    user.RealName,
		Email:       user.Profile.Email,
		AvatarURL:   user.Profile.Image192,
		CanonicalID: "", // Will be set by identity resolution
		AlternateIDs: nil,
	}
}

// convertSlackChannel converts a Slack channel to the normalized Channel schema
func convertSlackChannel(channel *SlackChannel, teamID string) *Channel {
	if channel == nil {
		return nil
	}
	
	channelType := "channel"
	if !channel.IsChannel {
		channelType = "dm"
	}
	
	return &Channel{
		ID:          fmt.Sprintf("chan_slack_%s_%s", teamID, channel.ID),
		SourceType:  "slack",
		SourceID:    channel.ID,
		Name:        channel.Name,
		DisplayName: "#" + channel.Name,
		Type:        channelType,
		IsPrivate:   channel.IsPrivate,
		ParentSpace: teamID,
	}
}

// extractMentions extracts user mentions from Slack text
func extractMentions(text string) []string {
	matches := userMentionPattern.FindAllStringSubmatch(text, -1)
	mentions := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			mentions = append(mentions, match[1])
		}
	}
	return mentions
}

// extractURLs extracts URLs from Slack text
func extractURLs(text string) []string {
	matches := urlPattern.FindAllStringSubmatch(text, -1)
	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			urls = append(urls, match[1])
		}
	}
	return urls
}

// extractCodeBlocks extracts code blocks from text
func extractCodeBlocks(text string) []CodeBlock {
	matches := codeBlockPattern.FindAllStringSubmatch(text, -1)
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

// normalizeSlackText converts Slack markup to plain text
func normalizeSlackText(text string) string {
	// Replace user mentions: <@U123|username> -> @username (or @U123 if no label)
	text = userMentionPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := userMentionPattern.FindStringSubmatch(match)
		if len(parts) > 3 && parts[3] != "" {
			return "@" + parts[3]
		}
		if len(parts) > 1 {
			return "@" + parts[1]
		}
		return match
	})
	
	// Replace channel mentions: <#C123|channel-name> -> #channel-name
	text = channelPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := channelPattern.FindStringSubmatch(match)
		if len(parts) > 3 && parts[3] != "" {
			return "#" + parts[3]
		}
		if len(parts) > 1 {
			return "#" + parts[1]
		}
		return match
	})
	
	// Replace URLs: <http://example.com|label> -> label (http://example.com)
	text = urlPattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := urlPattern.FindStringSubmatch(match)
		if len(parts) > 3 && parts[3] != "" {
			return fmt.Sprintf("%s (%s)", parts[3], parts[1])
		}
		if len(parts) > 1 {
			return parts[1]
		}
		return match
	})
	
	// Unescape HTML entities
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	
	return text
}

// convertSlackAttachments converts Slack file attachments
func convertSlackAttachments(files []map[string]interface{}) []Attachment {
	if len(files) == 0 {
		return nil
	}
	
	attachments := make([]Attachment, 0, len(files))
	for _, file := range files {
		att := Attachment{}
		
		if fileType, ok := file["filetype"].(string); ok {
			att.Type = fileType
		}
		if url, ok := file["url_private"].(string); ok {
			att.URL = url
		}
		if title, ok := file["title"].(string); ok {
			att.Title = title
		}
		if mime, ok := file["mimetype"].(string); ok {
			att.MimeType = mime
		}
		
		attachments = append(attachments, att)
	}
	
	return attachments
}
