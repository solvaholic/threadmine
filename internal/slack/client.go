package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rneatherway/slack"
)

// Client wraps the Slack API client
type Client struct {
	client *slack.Client
	teamID string
}

// AuthResult contains authentication information
type AuthResult struct {
	TeamName    string
	TeamID      string
	UserID      string
	UserName    string
	Authenticated bool
	Client      *Client
}

// Authenticate establishes a connection to Slack using cookies from the local Slack app
func Authenticate(team string) (*AuthResult, error) {
	client := slack.NewClient(team)
	
	// Attempt cookie-based authentication
	err := client.WithCookieAuth()
	if err != nil {
		return nil, formatAuthError(err)
	}

	// Validate the authentication by calling auth.test
	bs, err := client.API(context.Background(), "GET", "auth.test", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("authentication validation failed: %w", err)
	}

	// Parse the response
	var authResponse struct {
		OK    bool   `json:"ok"`
		Team  string `json:"team"`
		TeamID string `json:"team_id"`
		User  string `json:"user"`
		UserID string `json:"user_id"`
		Error string `json:"error"`
	}

	if err := json.Unmarshal(bs, &authResponse); err != nil {
		return nil, fmt.Errorf("failed to parse auth.test response: %w", err)
	}

	if !authResponse.OK {
		return nil, fmt.Errorf("Slack API returned error: %s", authResponse.Error)
	}

	return &AuthResult{
		TeamName:      authResponse.Team,
		TeamID:        authResponse.TeamID,
		UserID:        authResponse.UserID,
		UserName:      authResponse.User,
		Authenticated: true,
		Client:        &Client{client: client, teamID: authResponse.TeamID},
	}, nil
}

// SearchResult represents a Slack search result
type SearchResult struct {
	Type      string `json:"type"`
	Channel   Channel `json:"channel"`
	User      string `json:"user"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
	ThreadTS  string `json:"thread_ts,omitempty"`
	Permalink string `json:"permalink"`
}

// SearchResponse represents the response from search.messages
type SearchResponse struct {
	OK      bool `json:"ok"`
	Query   string `json:"query"`
	Messages struct {
		Total   int `json:"total"`
		Matches []SearchResult `json:"matches"`
	} `json:"messages"`
	Error string `json:"error"`
}

// SearchMessages searches for messages using the Slack search API
func (c *Client) SearchMessages(ctx context.Context, query string, count int) (*SearchResponse, error) {
	params := map[string]string{
		"query": query,
		"count": fmt.Sprintf("%d", count),
		"sort": "timestamp",
		"sort_dir": "desc",
	}

	bs, err := c.client.API(ctx, "GET", "search.messages", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	var response SearchResponse
	if err := json.Unmarshal(bs, &response); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Slack API error: %s", response.Error)
	}

	return &response, nil
}

// ThreadMessage represents a message in a thread
type ThreadMessage struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
	ThreadTS  string `json:"thread_ts,omitempty"`
	ParentUserID string `json:"parent_user_id,omitempty"`
}

// GetThreadReplies fetches all replies in a thread
func (c *Client) GetThreadReplies(ctx context.Context, channelID, threadTS string) ([]ThreadMessage, error) {
	params := map[string]string{
		"channel": channelID,
		"ts": threadTS,
		"limit": "1000",
	}

	bs, err := c.client.API(ctx, "GET", "conversations.replies", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread replies: %w", err)
	}

	var response struct {
		OK       bool            `json:"ok"`
		Messages []ThreadMessage `json:"messages"`
		Error    string          `json:"error"`
	}

	if err := json.Unmarshal(bs, &response); err != nil {
		return nil, fmt.Errorf("failed to parse thread replies: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Slack API error: %s", response.Error)
	}

	return response.Messages, nil
}

// GetUserInfo fetches user profile information
func (c *Client) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	params := map[string]string{
		"user": userID,
	}

	bs, err := c.client.API(ctx, "GET", "users.info", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var response struct {
		OK    bool     `json:"ok"`
		User  UserInfo `json:"user"`
		Error string   `json:"error"`
	}

	if err := json.Unmarshal(bs, &response); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Slack API error: %s", response.Error)
	}

	return &response.User, nil
}

// UserInfo represents Slack user information
type UserInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Profile  struct {
		Email     string `json:"email"`
		Image192  string `json:"image_192"`
	} `json:"profile"`
}

// Channel represents a Slack channel
type Channel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IsChannel   bool   `json:"is_channel"`
	IsPrivate   bool   `json:"is_private"`
	IsMember    bool   `json:"is_member"`
	NumMembers  int    `json:"num_members"`
}

// Message represents a Slack message
type Message struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
	ThreadTS  string `json:"thread_ts,omitempty"`
}

// ListChannels fetches all channels the user is a member of
func (c *Client) ListChannels(ctx context.Context) ([]Channel, error) {
	bs, err := c.client.API(ctx, "GET", "conversations.list", map[string]string{
		"types": "public_channel,private_channel",
		"limit": "1000",
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}

	var response struct {
		OK       bool      `json:"ok"`
		Channels []Channel `json:"channels"`
		Error    string    `json:"error"`
	}

	if err := json.Unmarshal(bs, &response); err != nil {
		return nil, fmt.Errorf("failed to parse channels list: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Slack API error: %s", response.Error)
	}

	// Filter to only channels the user is a member of
	var memberChannels []Channel
	for _, ch := range response.Channels {
		if ch.IsMember {
			memberChannels = append(memberChannels, ch)
		}
	}

	return memberChannels, nil
}

// GetMessages implements cache-aside pattern for message retrieval
// Checks cache first, fetches from API on miss, and stores in cache
func (c *Client) GetMessages(ctx context.Context, channelID string, oldest time.Time, cacheDir string) ([]Message, error) {
	// Import cache package for cache operations
	// First, try to load from cache
	cache, err := loadMessagesFromCache(c.teamID, channelID, oldest)
	if err != nil {
		return nil, fmt.Errorf("error checking cache: %w", err)
	}

	// Cache hit - return cached messages
	if cache != nil {
		messages := make([]Message, 0, len(cache.Messages))
		for _, msg := range cache.Messages {
			// Convert interface{} back to Message
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			var message Message
			if err := json.Unmarshal(msgBytes, &message); err != nil {
				continue
			}
			messages = append(messages, message)
		}
		return messages, nil
	}

	// Cache miss - fetch from API
	messages, err := c.FetchMessages(ctx, channelID, oldest)
	if err != nil {
		return nil, err
	}

	// Store in cache
	var messagesToCache []interface{}
	for _, msg := range messages {
		messagesToCache = append(messagesToCache, msg)
	}

	if err := saveMessagesToCache(c.teamID, channelID, messagesToCache); err != nil {
		// Log warning but don't fail - we have the data
		fmt.Fprintf(os.Stderr, "Warning: failed to cache messages: %v\n", err)
	}

	return messages, nil
}

// FetchMessages retrieves messages from a channel (direct API call, no caching)
func (c *Client) FetchMessages(ctx context.Context, channelID string, oldest time.Time) ([]Message, error) {
	params := map[string]string{
		"channel": channelID,
		"limit":   "1000",
	}
	
	if !oldest.IsZero() {
		params["oldest"] = fmt.Sprintf("%d.000000", oldest.Unix())
	}

	bs, err := c.client.API(ctx, "GET", "conversations.history", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	var response struct {
		OK       bool      `json:"ok"`
		Messages []Message `json:"messages"`
		Error    string    `json:"error"`
	}

	if err := json.Unmarshal(bs, &response); err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Slack API error: %s", response.Error)
	}

	return response.Messages, nil
}

// loadMessagesFromCache is a helper function that loads messages from cache
func loadMessagesFromCache(teamID, channelID string, since time.Time) (*messageCache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	msgDir := filepath.Join(home, ".threadmine", "raw", "slack", "workspaces", teamID, "channels", channelID, "messages")
	date := time.Now().Format("2006-01-02")
	filePath := filepath.Join(msgDir, fmt.Sprintf("%s.json", date))

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // Cache miss
	}

	// Read and parse the cache file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache messageCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Check if cache is fresh enough
	if !since.IsZero() && cache.FetchedAt.Before(since) {
		return nil, nil // Cache too old
	}

	return &cache, nil
}

// saveMessagesToCache is a helper function that saves messages to cache
func saveMessagesToCache(teamID, channelID string, messages []interface{}) error {
	if len(messages) == 0 {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	msgDir := filepath.Join(home, ".threadmine", "raw", "slack", "workspaces", teamID, "channels", channelID, "messages")

	// Create directory with restrictive permissions
	if err := os.MkdirAll(msgDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Use today's date for the filename
	date := time.Now().Format("2006-01-02")
	filePath := filepath.Join(msgDir, fmt.Sprintf("%s.json", date))

	cache := messageCache{
		TeamID:    teamID,
		ChannelID: channelID,
		Date:      date,
		FetchedAt: time.Now(),
		Messages:  messages,
	}

	// Marshal to JSON with indentation for human readability
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	// Write to temp file first, then rename (atomic write)
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	return nil
}

// messageCache represents cached message data
type messageCache struct {
	TeamID    string        `json:"team_id"`
	ChannelID string        `json:"channel_id"`
	Date      string        `json:"date"`
	FetchedAt time.Time     `json:"fetched_at"`
	Messages  []interface{} `json:"messages"`
}

// formatAuthError provides user-friendly error messages for common authentication failures
func formatAuthError(err error) error {
	errMsg := err.Error()
	
	// Check for common error patterns and provide helpful guidance
	if contains(errMsg, "no Slack cookie database found") || contains(errMsg, "could not access Slack cookie database") {
		return fmt.Errorf("Slack cookie database not found. Are you logged into the Slack desktop app?\n  Original error: %v", err)
	}
	
	if contains(errMsg, "no matching unlocked items found") {
		return fmt.Errorf("Slack cookie not found in keychain. Try logging out and back into the Slack desktop app.\n  Original error: %v", err)
	}
	
	if contains(errMsg, "failed to get cookie password") {
		return fmt.Errorf("could not retrieve Slack cookie password from keychain. Check that the Slack app has keychain access.\n  Original error: %v", err)
	}
	
	if contains(errMsg, "status code") {
		return fmt.Errorf("failed to authenticate with Slack (network or server error). Check your internet connection.\n  Original error: %v", err)
	}
	
	// Default: return the original error with context
	return fmt.Errorf("Slack authentication failed: %w", err)
}

// contains checks if a string contains a substring (case-sensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		})())
}
