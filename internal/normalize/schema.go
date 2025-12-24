package normalize

import "time"

// NormalizedMessage represents a message in the common schema across all sources
type NormalizedMessage struct {
	// Universal identifiers
	ID         string `json:"id"`          // msg_slack_1234567890.123456
	SourceType string `json:"source_type"` // "slack", "github", "email"
	SourceID   string `json:"source_id"`   // Original source identifier

	// Common fields
	Timestamp   time.Time `json:"timestamp"`
	Author      *User     `json:"author"`
	Content     string    `json:"content"`      // Normalized text
	ContentHTML string    `json:"content_html"` // Rich format if available

	// Conversation context
	Channel      *Channel `json:"channel"`
	ThreadID     string   `json:"thread_id"`
	ParentID     string   `json:"parent_id"`
	IsThreadRoot bool     `json:"is_thread_root"`

	// Metadata
	Attachments []Attachment `json:"attachments"`
	Mentions    []string     `json:"mentions"`
	URLs        []string     `json:"urls"`
	CodeBlocks  []CodeBlock  `json:"code_blocks"`

	// Source-specific (preserved as-is)
	SourceMetadata map[string]interface{} `json:"source_metadata"`

	// Provenance
	FetchedAt     time.Time `json:"fetched_at"`
	NormalizedAt  time.Time `json:"normalized_at"`
	SchemaVersion string    `json:"schema_version"`
}

// User represents a user across sources
type User struct {
	ID           string   `json:"id"`
	SourceType   string   `json:"source_type"`
	SourceID     string   `json:"source_id"`
	DisplayName  string   `json:"display_name"`
	RealName     string   `json:"real_name"`
	Email        string   `json:"email"`
	AvatarURL    string   `json:"avatar_url"`
	CanonicalID  string   `json:"canonical_id"`
	AlternateIDs []string `json:"alternate_ids"`
}

// Channel represents a conversation container across sources.
// Note: A "Channel" means different things on different platforms:
//   - Slack: A channel (with ParentSpace = workspace)
//   - GitHub: An Issue/PR (with ParentSpace = repository)
//   - Support: A ticket (with ParentSpace = organization)
//   - GitHub Discussions: Deferred (structurally more like Slack channels)
//
// The Type field distinguishes between these ("channel", "issue", "pr", "ticket", etc.)
type Channel struct {
	ID          string `json:"id"`
	SourceType  string `json:"source_type"`
	SourceID    string `json:"source_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"` // "channel", "dm", "issue", "pr", "ticket", etc.
	IsPrivate   bool   `json:"is_private"`
	ParentSpace string `json:"parent_space"` // Workspace, repo, organization, etc.
}

// Attachment represents a file or rich media attachment
type Attachment struct {
	Type     string `json:"type"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	MimeType string `json:"mime_type"`
}

// CodeBlock represents a code snippet in a message
type CodeBlock struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

const SchemaVersion = "1.0"
