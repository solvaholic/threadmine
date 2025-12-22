package normalize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NormalizedDir returns the root directory for normalized data
func NormalizedDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".threadmine", "normalized"), nil
}

// MessagesByIDDir returns the directory for messages indexed by ID
func MessagesByIDDir() (string, error) {
	normalizedDir, err := NormalizedDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(normalizedDir, "messages", "by_id"), nil
}

// MessagesByDateDir returns the directory for messages indexed by date
func MessagesByDateDir() (string, error) {
	normalizedDir, err := NormalizedDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(normalizedDir, "messages", "by_date"), nil
}

// MessagesBySourceDir returns the directory for messages indexed by source
func MessagesBySourceDir() (string, error) {
	normalizedDir, err := NormalizedDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(normalizedDir, "messages", "by_source"), nil
}

// SaveNormalizedMessage saves a normalized message to all necessary indexes
func SaveNormalizedMessage(msg *NormalizedMessage) error {
	// Save by ID
	if err := saveMessageByID(msg); err != nil {
		return fmt.Errorf("failed to save message by ID: %w", err)
	}
	
	// Append to date index
	if err := appendMessageByDate(msg); err != nil {
		return fmt.Errorf("failed to append message by date: %w", err)
	}
	
	// Append to source index
	if err := appendMessageBySource(msg); err != nil {
		return fmt.Errorf("failed to append message by source: %w", err)
	}
	
	return nil
}

// saveMessageByID saves a message as an individual JSON file indexed by message ID
func saveMessageByID(msg *NormalizedMessage) error {
	dir, err := MessagesByIDDir()
	if err != nil {
		return err
	}
	
	// Create directory with restrictive permissions
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Build file path
	filePath := filepath.Join(dir, msg.ID+".json")
	
	// Marshal to JSON with indentation for human readability
	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
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

// appendMessageByDate appends a message to the date-indexed JSONL file
func appendMessageByDate(msg *NormalizedMessage) error {
	dir, err := MessagesByDateDir()
	if err != nil {
		return err
	}
	
	// Create directory structure: by_date/YYYY-MM/YYYY-MM-DD.jsonl
	yearMonth := msg.Timestamp.Format("2006-01")
	dateDir := filepath.Join(dir, yearMonth)
	
	if err := os.MkdirAll(dateDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Build file path
	date := msg.Timestamp.Format("2006-01-02")
	filePath := filepath.Join(dateDir, date+".jsonl")
	
	// Marshal message to single-line JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Append to file (create if doesn't exist)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	
	return nil
}

// appendMessageBySource appends a message to the source-indexed JSONL file
func appendMessageBySource(msg *NormalizedMessage) error {
	dir, err := MessagesBySourceDir()
	if err != nil {
		return err
	}
	
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Build file path based on source type
	filePath := filepath.Join(dir, msg.SourceType+".jsonl")
	
	// Marshal message to single-line JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Append to file (create if doesn't exist)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	
	return nil
}

// LoadMessageByID loads a normalized message by its ID
func LoadMessageByID(id string) (*NormalizedMessage, error) {
	dir, err := MessagesByIDDir()
	if err != nil {
		return nil, err
	}
	
	filePath := filepath.Join(dir, id+".json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("message not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	var msg NormalizedMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	
	return &msg, nil
}

// LoadMessagesByDate loads all messages from a specific date
func LoadMessagesByDate(date time.Time) ([]*NormalizedMessage, error) {
	dir, err := MessagesByDateDir()
	if err != nil {
		return nil, err
	}
	
	yearMonth := date.Format("2006-01")
	dateStr := date.Format("2006-01-02")
	filePath := filepath.Join(dir, yearMonth, dateStr+".jsonl")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*NormalizedMessage{}, nil // Return empty slice if no messages for this date
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	// Parse JSONL (one message per line)
	lines := splitLines(data)
	messages := make([]*NormalizedMessage, 0, len(lines))
	
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		
		var msg NormalizedMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message on line %d: %w", i+1, err)
		}
		messages = append(messages, &msg)
	}
	
	return messages, nil
}

// splitLines splits data by newlines
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	
	// Add last line if it doesn't end with newline
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	
	return lines
}
