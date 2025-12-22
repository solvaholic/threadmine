package classify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/solvaholic/threadmine/internal/normalize"
)

// AnnotatedMessage represents a message with its classifications
type AnnotatedMessage struct {
	Message         *normalize.NormalizedMessage `json:"message"`
	Classifications []Classification             `json:"classifications"`
	AnnotatedAt     string                       `json:"annotated_at"`
	SchemaVersion   string                       `json:"schema_version"`
}

// AnnotationsDir returns the root directory for annotations
func AnnotationsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".threadmine", "annotations"), nil
}

// MessageAnnotationsDir returns the directory for a specific message's annotations
func MessageAnnotationsDir(messageID string) (string, error) {
	annotationsDir, err := AnnotationsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(annotationsDir, "messages", messageID), nil
}

// SaveClassifications saves message classifications to disk
func SaveClassifications(msg *normalize.NormalizedMessage, classifications []Classification) error {
	if len(classifications) == 0 {
		return nil // Nothing to save
	}

	dir, err := MessageAnnotationsDir(msg.ID)
	if err != nil {
		return err
	}

	// Create directory with restrictive permissions
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create annotations directory: %w", err)
	}

	filePath := filepath.Join(dir, "classifications.json")

	annotated := AnnotatedMessage{
		Message:         msg,
		Classifications: classifications,
		AnnotatedAt:     msg.NormalizedAt.Format("2006-01-02T15:04:05Z07:00"),
		SchemaVersion:   "1.0",
	}

	// Marshal to JSON with indentation for human readability
	data, err := json.MarshalIndent(annotated, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal classifications: %w", err)
	}

	// Write to temp file first, then rename (atomic write)
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// LoadClassifications loads classifications for a message from disk
func LoadClassifications(messageID string) ([]Classification, error) {
	dir, err := MessageAnnotationsDir(messageID)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(dir, "classifications.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No classifications saved yet
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var annotated AnnotatedMessage
	if err := json.Unmarshal(data, &annotated); err != nil {
		return nil, fmt.Errorf("failed to unmarshal classifications: %w", err)
	}

	return annotated.Classifications, nil
}
