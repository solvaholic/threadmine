package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheDir returns the root cache directory path
func CacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".threadmine"), nil
}

// RawSlackDir returns the directory for raw Slack data for a workspace
func RawSlackDir(teamID string) (string, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "raw", "slack", "workspaces", teamID), nil
}

// ChannelMessagesDir returns the directory for messages in a specific channel
func ChannelMessagesDir(teamID, channelID string) (string, error) {
	slackDir, err := RawSlackDir(teamID)
	if err != nil {
		return "", err
	}
	return filepath.Join(slackDir, "channels", channelID, "messages"), nil
}

// MessageCache represents cached message data
type MessageCache struct {
	TeamID    string        `json:"team_id"`
	ChannelID string        `json:"channel_id"`
	Date      string        `json:"date"`
	FetchedAt time.Time     `json:"fetched_at"`
	Messages  []interface{} `json:"messages"`
}

// SaveMessages saves messages to the cache
func SaveMessages(teamID, channelID string, messages []interface{}) error {
	if len(messages) == 0 {
		return nil
	}

	msgDir, err := ChannelMessagesDir(teamID, channelID)
	if err != nil {
		return err
	}

	// Create directory with restrictive permissions
	if err := os.MkdirAll(msgDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Use today's date for the filename
	date := time.Now().Format("2006-01-02")
	filePath := filepath.Join(msgDir, fmt.Sprintf("%s.json", date))

	cache := MessageCache{
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

// SaveChannelInfo saves channel metadata
func SaveChannelInfo(teamID, channelID string, info interface{}) error {
	slackDir, err := RawSlackDir(teamID)
	if err != nil {
		return err
	}

	channelDir := filepath.Join(slackDir, "channels", channelID)
	if err := os.MkdirAll(channelDir, 0700); err != nil {
		return fmt.Errorf("failed to create channel directory: %w", err)
	}

	filePath := filepath.Join(channelDir, "info.json")

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal channel info: %w", err)
	}

	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write channel info: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename channel info file: %w", err)
	}

	return nil
}

// SaveChannelsList saves the list of channels
// LoadMessages retrieves cached messages for a channel and date range
// Returns nil if no cache exists (cache miss)
func LoadMessages(teamID, channelID string, since time.Time) (*MessageCache, error) {
	msgDir, err := ChannelMessagesDir(teamID, channelID)
	if err != nil {
		return nil, err
	}

	// Check if the cache file exists for the requested date
	// For simplicity, check today's date first
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

	var cache MessageCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Check if cache is fresh enough (based on since parameter)
	// If the cached data is older than requested, treat as cache miss
	if !since.IsZero() && cache.FetchedAt.Before(since) {
		return nil, nil // Cache too old
	}

	return &cache, nil
}

func SaveChannelsList(teamID string, channels interface{}) error {
	slackDir, err := RawSlackDir(teamID)
	if err != nil {
		return err
	}

	channelsDir := filepath.Join(slackDir, "channels")
	if err := os.MkdirAll(channelsDir, 0700); err != nil {
		return fmt.Errorf("failed to create channels directory: %w", err)
	}

	filePath := filepath.Join(channelsDir, "_index.json")

	indexData := map[string]interface{}{
		"fetched_at": time.Now(),
		"channels":   channels,
	}

	data, err := json.MarshalIndent(indexData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal channels list: %w", err)
	}

	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write channels list: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename channels list file: %w", err)
	}

	return nil
}

// DiscoverWorkspaces returns all cached Slack workspace IDs
func DiscoverWorkspaces() ([]string, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return nil, err
	}

	workspacesDir := filepath.Join(cacheDir, "raw", "slack", "workspaces")

	// Check if directory exists
	if _, err := os.Stat(workspacesDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspaces directory: %w", err)
	}

	var workspaceIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			workspaceIDs = append(workspaceIDs, entry.Name())
		}
	}

	return workspaceIDs, nil
}

// WorkspaceUser represents the authenticated user for a workspace
type WorkspaceUser struct {
	UserID   string    `json:"user_id"`
	UserName string    `json:"user_name"`
	TeamID   string    `json:"team_id"`
	TeamName string    `json:"team_name"`
	CachedAt time.Time `json:"cached_at"`
}

// SaveWorkspaceUser saves authenticated user info for a workspace
func SaveWorkspaceUser(teamID string, userID, userName, teamName string) error {
	slackDir, err := RawSlackDir(teamID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(slackDir, 0700); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	filePath := filepath.Join(slackDir, "user.json")

	user := WorkspaceUser{
		UserID:   userID,
		UserName: userName,
		TeamID:   teamID,
		TeamName: teamName,
		CachedAt: time.Now(),
	}

	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user info: %w", err)
	}

	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write user info: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename user info file: %w", err)
	}

	return nil
}

// GetWorkspaceUser retrieves the authenticated user for a workspace
func GetWorkspaceUser(teamID string) (*WorkspaceUser, error) {
	slackDir, err := RawSlackDir(teamID)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(slackDir, "user.json")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no cached user info for workspace %s", teamID)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info: %w", err)
	}

	var user WorkspaceUser
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &user, nil
}
