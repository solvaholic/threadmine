package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Message represents a normalized message in the database
type Message struct {
	ID          string
	SourceType  string
	SourceID    string
	Timestamp   time.Time
	AuthorID    string
	Content     string
	ContentHTML *string
	ChannelID   string
	ThreadID    *string
	ParentID    *string
	IsThreadRoot bool
	Mentions    []string
	URLs        []string
	CodeBlocks  []CodeBlock
	Attachments []Attachment
	NormalizedAt time.Time
	SchemaVersion string
}

// CodeBlock represents a code snippet
type CodeBlock struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

// Attachment represents a file attachment
type Attachment struct {
	Type     string `json:"type"`
	URL      string `json:"url"`
	Title    string `json:"title,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// SaveMessage saves a normalized message to the database
func (db *DB) SaveMessage(msg *Message) error {
	// Encode JSON fields
	mentions, err := json.Marshal(msg.Mentions)
	if err != nil {
		return fmt.Errorf("failed to marshal mentions: %w", err)
	}

	urls, err := json.Marshal(msg.URLs)
	if err != nil {
		return fmt.Errorf("failed to marshal urls: %w", err)
	}

	codeBlocks, err := json.Marshal(msg.CodeBlocks)
	if err != nil {
		return fmt.Errorf("failed to marshal code_blocks: %w", err)
	}

	attachments, err := json.Marshal(msg.Attachments)
	if err != nil {
		return fmt.Errorf("failed to marshal attachments: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO messages (
			id, source_type, source_id, timestamp, author_id, content, content_html,
			channel_id, thread_id, parent_id, is_thread_root,
			mentions, urls, code_blocks, attachments,
			normalized_at, schema_version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			content_html = excluded.content_html,
			mentions = excluded.mentions,
			urls = excluded.urls,
			code_blocks = excluded.code_blocks,
			attachments = excluded.attachments,
			normalized_at = excluded.normalized_at
	`, msg.ID, msg.SourceType, msg.SourceID, msg.Timestamp, msg.AuthorID,
		msg.Content, msg.ContentHTML, msg.ChannelID, msg.ThreadID, msg.ParentID,
		msg.IsThreadRoot, mentions, urls, codeBlocks, attachments,
		msg.NormalizedAt, msg.SchemaVersion)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetMessage retrieves a message by ID
func (db *DB) GetMessage(id string) (*Message, error) {
	msg := &Message{}
	var mentions, urls, codeBlocks, attachments string

	err := db.QueryRow(`
		SELECT id, source_type, source_id, timestamp, author_id, content, content_html,
		       channel_id, thread_id, parent_id, is_thread_root,
		       mentions, urls, code_blocks, attachments,
		       normalized_at, schema_version
		FROM messages
		WHERE id = ?
	`, id).Scan(
		&msg.ID, &msg.SourceType, &msg.SourceID, &msg.Timestamp, &msg.AuthorID,
		&msg.Content, &msg.ContentHTML, &msg.ChannelID, &msg.ThreadID, &msg.ParentID,
		&msg.IsThreadRoot, &mentions, &urls, &codeBlocks, &attachments,
		&msg.NormalizedAt, &msg.SchemaVersion,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Decode JSON fields
	if err := json.Unmarshal([]byte(mentions), &msg.Mentions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mentions: %w", err)
	}
	if err := json.Unmarshal([]byte(urls), &msg.URLs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal urls: %w", err)
	}
	if err := json.Unmarshal([]byte(codeBlocks), &msg.CodeBlocks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal code_blocks: %w", err)
	}
	if err := json.Unmarshal([]byte(attachments), &msg.Attachments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attachments: %w", err)
	}

	return msg, nil
}

// SelectMessagesOptions defines options for selecting messages
type SelectMessagesOptions struct {
	SourceType  *string
	AuthorID    *string
	ChannelID   *string
	ThreadID    *string
	Since       *time.Time
	Until       *time.Time
	SearchText  *string
	Limit       int
	Offset      int
}

// SelectMessages queries messages with filters
func (db *DB) SelectMessages(opts SelectMessagesOptions) ([]*Message, error) {
	query := `
		SELECT id, source_type, source_id, timestamp, author_id, content, content_html,
		       channel_id, thread_id, parent_id, is_thread_root,
		       mentions, urls, code_blocks, attachments,
		       normalized_at, schema_version
		FROM messages
		WHERE 1=1
	`
	args := []interface{}{}

	if opts.SourceType != nil {
		query += " AND source_type = ?"
		args = append(args, *opts.SourceType)
	}
	if opts.AuthorID != nil {
		query += " AND author_id = ?"
		args = append(args, *opts.AuthorID)
	}
	if opts.ChannelID != nil {
		query += " AND channel_id = ?"
		args = append(args, *opts.ChannelID)
	}
	if opts.ThreadID != nil {
		query += " AND thread_id = ?"
		args = append(args, *opts.ThreadID)
	}
	if opts.Since != nil {
		query += " AND timestamp >= ?"
		args = append(args, *opts.Since)
	}
	if opts.Until != nil {
		query += " AND timestamp <= ?"
		args = append(args, *opts.Until)
	}
	if opts.SearchText != nil {
		// Use LIKE for text search (FTS5 disabled for now)
		// TODO: Re-enable FTS5 when building with: go build -tags "fts5"
		query += " AND content LIKE ?"
		args = append(args, "%"+*opts.SearchText+"%")
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select messages: %w", err)
	}
	defer rows.Close()

	messages := []*Message{}
	for rows.Next() {
		msg := &Message{}
		var mentions, urls, codeBlocks, attachments string

		err := rows.Scan(
			&msg.ID, &msg.SourceType, &msg.SourceID, &msg.Timestamp, &msg.AuthorID,
			&msg.Content, &msg.ContentHTML, &msg.ChannelID, &msg.ThreadID, &msg.ParentID,
			&msg.IsThreadRoot, &mentions, &urls, &codeBlocks, &attachments,
			&msg.NormalizedAt, &msg.SchemaVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Decode JSON fields
		if err := json.Unmarshal([]byte(mentions), &msg.Mentions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal mentions: %w", err)
		}
		if err := json.Unmarshal([]byte(urls), &msg.URLs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal urls: %w", err)
		}
		if err := json.Unmarshal([]byte(codeBlocks), &msg.CodeBlocks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal code_blocks: %w", err)
		}
		if err := json.Unmarshal([]byte(attachments), &msg.Attachments); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attachments: %w", err)
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// SaveRawMessage saves a raw message to the database
func (db *DB) SaveRawMessage(id, sourceType, sourceID, workspaceID, containerID, rawData, fetchQuery string) error {
	_, err := db.Exec(`
		INSERT INTO raw_messages (
			id, source_type, source_id, workspace_id, container_id, raw_data, fetch_query
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_type, source_id, workspace_id) DO UPDATE SET
			raw_data = excluded.raw_data,
			fetched_at = CURRENT_TIMESTAMP,
			fetch_query = excluded.fetch_query
	`, id, sourceType, sourceID, workspaceID, containerID, rawData, fetchQuery)

	if err != nil {
		return fmt.Errorf("failed to save raw message: %w", err)
	}

	return nil
}
