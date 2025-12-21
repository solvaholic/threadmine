package slack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rneatherway/slack"
)

// AuthResult contains authentication information
type AuthResult struct {
	TeamName    string
	TeamID      string
	UserID      string
	UserName    string
	Authenticated bool
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
	}, nil
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
