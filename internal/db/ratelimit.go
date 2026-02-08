package db

import (
	"database/sql"
	"fmt"
	"time"
)

// RateLimit represents API rate limiting information
type RateLimit struct {
	SourceType            string
	WorkspaceID           *string
	Endpoint              string
	RequestsMade          int
	WindowStart           time.Time
	WindowDurationSeconds int
	MaxRequests           int
	SafetyLimit           int
}

// CheckRateLimit checks if we can make a request within rate limits
// Returns true if request is allowed, false if rate limited
func (db *DB) CheckRateLimit(sourceType string, workspaceID *string, endpoint string) (bool, error) {
	var rl RateLimit
	wsID := sql.NullString{}
	if workspaceID != nil {
		wsID.String = *workspaceID
		wsID.Valid = true
	}

	err := db.QueryRow(`
		SELECT source_type, workspace_id, endpoint, requests_made, window_start,
		       window_duration_seconds, max_requests, safety_limit
		FROM rate_limits
		WHERE source_type = ? AND workspace_id IS ? AND endpoint = ?
	`, sourceType, wsID, endpoint).Scan(
		&rl.SourceType, &rl.WorkspaceID, &rl.Endpoint, &rl.RequestsMade,
		&rl.WindowStart, &rl.WindowDurationSeconds, &rl.MaxRequests, &rl.SafetyLimit,
	)

	if err == sql.ErrNoRows {
		// No rate limit entry, initialize one
		return true, db.InitRateLimit(sourceType, workspaceID, endpoint, 60, 20, 10)
	}
	if err != nil {
		return false, fmt.Errorf("failed to check rate limit: %w", err)
	}

	// Check if window has expired
	now := time.Now()
	windowEnd := rl.WindowStart.Add(time.Duration(rl.WindowDurationSeconds) * time.Second)
	if now.After(windowEnd) {
		// Reset window
		return true, db.ResetRateLimitWindow(sourceType, workspaceID, endpoint)
	}

	// Check if we're within safety limit
	if rl.RequestsMade >= rl.SafetyLimit {
		return false, nil
	}

	return true, nil
}

// RecordRequest records a successful API request
func (db *DB) RecordRequest(sourceType string, workspaceID *string, endpoint string) error {
	wsID := sql.NullString{}
	if workspaceID != nil {
		wsID.String = *workspaceID
		wsID.Valid = true
	}

	_, err := db.Exec(`
		UPDATE rate_limits
		SET requests_made = requests_made + 1
		WHERE source_type = ? AND workspace_id IS ? AND endpoint = ?
	`, sourceType, wsID, endpoint)

	if err != nil {
		return fmt.Errorf("failed to record request: %w", err)
	}

	return nil
}

// InitRateLimit initializes rate limit tracking for an endpoint
func (db *DB) InitRateLimit(sourceType string, workspaceID *string, endpoint string, windowDuration, maxRequests, safetyLimit int) error {
	wsID := sql.NullString{}
	if workspaceID != nil {
		wsID.String = *workspaceID
		wsID.Valid = true
	}

	_, err := db.Exec(`
		INSERT INTO rate_limits (
			source_type, workspace_id, endpoint, requests_made, window_start,
			window_duration_seconds, max_requests, safety_limit
		) VALUES (?, ?, ?, 0, ?, ?, ?, ?)
		ON CONFLICT(source_type, workspace_id, endpoint) DO NOTHING
	`, sourceType, wsID, endpoint, time.Now(), windowDuration, maxRequests, safetyLimit)

	if err != nil {
		return fmt.Errorf("failed to init rate limit: %w", err)
	}

	return nil
}

// ResetRateLimitWindow resets the rate limit window
func (db *DB) ResetRateLimitWindow(sourceType string, workspaceID *string, endpoint string) error {
	wsID := sql.NullString{}
	if workspaceID != nil {
		wsID.String = *workspaceID
		wsID.Valid = true
	}

	_, err := db.Exec(`
		UPDATE rate_limits
		SET requests_made = 0, window_start = ?
		WHERE source_type = ? AND workspace_id IS ? AND endpoint = ?
	`, time.Now(), sourceType, wsID, endpoint)

	if err != nil {
		return fmt.Errorf("failed to reset rate limit window: %w", err)
	}

	return nil
}

// GetRateLimitStatus returns the current rate limit status
func (db *DB) GetRateLimitStatus(sourceType string, workspaceID *string, endpoint string) (*RateLimit, error) {
	rl := &RateLimit{}
	wsID := sql.NullString{}
	if workspaceID != nil {
		wsID.String = *workspaceID
		wsID.Valid = true
	}

	err := db.QueryRow(`
		SELECT source_type, workspace_id, endpoint, requests_made, window_start,
		       window_duration_seconds, max_requests, safety_limit
		FROM rate_limits
		WHERE source_type = ? AND workspace_id IS ? AND endpoint = ?
	`, sourceType, wsID, endpoint).Scan(
		&rl.SourceType, &rl.WorkspaceID, &rl.Endpoint, &rl.RequestsMade,
		&rl.WindowStart, &rl.WindowDurationSeconds, &rl.MaxRequests, &rl.SafetyLimit,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit status: %w", err)
	}

	return rl, nil
}
