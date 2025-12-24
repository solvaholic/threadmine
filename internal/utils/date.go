package utils

import (
	"fmt"
	"time"
)

// ParseSinceDate parses a date string that can be in two formats:
// - Relative: "7d" (days ago)
// - Absolute: "2025-12-15" (YYYY-MM-DD)
//
// Returns the parsed time or an error if the format is invalid.
func ParseSinceDate(since string) (time.Time, error) {
	if since == "" {
		return time.Time{}, fmt.Errorf("since date cannot be empty")
	}

	// Check for relative format (e.g., "7d")
	if since[len(since)-1] == 'd' {
		days := 0
		_, err := fmt.Sscanf(since, "%dd", &days)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid relative date format '%s': expected format like '7d'", since)
		}
		if days < 0 {
			return time.Time{}, fmt.Errorf("days cannot be negative: %d", days)
		}
		return time.Now().AddDate(0, 0, -days), nil
	}

	// Try absolute format (YYYY-MM-DD)
	parsed, err := time.Parse("2006-01-02", since)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format '%s': expected 'YYYY-MM-DD' or relative format like '7d'", since)
	}

	return parsed, nil
}
