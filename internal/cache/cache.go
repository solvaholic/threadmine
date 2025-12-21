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
