package slack

import (
	"context"
	"encoding/json"
	"fmt"
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

// FetchMessages retrieves messages from a channel
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
