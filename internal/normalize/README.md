# Normalize Package

The `normalize` package converts source-specific message formats (Slack, GitHub, email) into a common schema for unified analysis across platforms.

## Architecture

The normalization layer sits between raw cached data and the analysis layer:

```
Raw Layer (source-specific) → Normalized Layer → Analysis Layer
```

## Schema

### NormalizedMessage

The core normalized message format with these key fields:

- **Universal identifiers**: `msg_slack_T123_C456_1234567890.123456`
- **Common fields**: timestamp, author, content
- **Conversation context**: channel, thread_id, parent_id
- **Extracted metadata**: mentions, URLs, code blocks
- **Source-specific**: preserved in `source_metadata` field
- **Provenance**: fetched_at, normalized_at, schema_version

### Storage Layout

Normalized messages are stored in three indexes for efficient querying:

```
~/.threadmine/normalized/messages/
├── by_id/              # Individual JSON files by message ID
│   └── msg_slack_*.json
├── by_date/            # JSONL files organized by date
│   └── YYYY-MM/
│       └── YYYY-MM-DD.jsonl
└── by_source/          # JSONL files organized by source
    ├── slack.jsonl
    ├── github.jsonl
    └── email.jsonl
```

## Usage

### Converting Slack Messages

```go
import "github.com/solvaholic/threadmine/internal/normalize"

// Convert a Slack message to normalized format
normalized, err := normalize.SlackToNormalized(
    slackMsg,
    slackChannel,
    slackUser,
    teamID,
    time.Now(), // fetched_at
)

// Save to all indexes
err = normalize.SaveNormalizedMessage(normalized)
```

### Loading Messages

```go
// Load by ID
msg, err := normalize.LoadMessageByID("msg_slack_T123_C456_1234567890.123456")

// Load all messages from a specific date
msgs, err := normalize.LoadMessagesByDate(time.Date(2025, 12, 21, 0, 0, 0, 0, time.UTC))
```

## Features

### Slack Markup Normalization

Converts Slack-specific markup to plain text:

- User mentions: `<@U123|john>` → `@john`
- Channel mentions: `<#C123|general>` → `#general`
- URLs: `<https://example.com|link>` → `link (https://example.com)`
- HTML entities: `&lt;` → `<`, `&gt;` → `>`, `&amp;` → `&`

### Metadata Extraction

Automatically extracts:

- **User mentions**: All `@user` references
- **URLs**: All links in the message
- **Code blocks**: Language and content of code snippets
- **Thread context**: Thread ID, parent message ID, thread root status

### File Operations

All file operations follow ThreadMine conventions:

- **Atomic writes**: Write to `.tmp`, then rename
- **Restrictive permissions**: 0600 for files, 0700 for directories
- **Human-readable**: JSON with 2-space indentation for individual files
- **JSONL for indexes**: One message per line for efficient appends and streaming

## Testing

Run tests with:

```bash
go test ./internal/normalize -v
```

Tests cover:

- Basic message normalization
- Thread relationship handling
- Slack markup conversion
- Mention/URL/code block extraction
- Timestamp parsing

## Future Enhancements

- GitHub issue/PR normalization
- Email message normalization
- Cross-source message linking
- Enhanced entity extraction
