# ThreadMine Specification

**Version**: 2.0 (Redesign)
**Status**: In Development

## Overview

ThreadMine (`mine`) is a command-line tool for searching and analyzing conversations across Slack, GitHub, and email. It uses search APIs to fetch messages, stores them in a local SQLite database, and provides powerful query capabilities for analysis.

## Core Concepts

### Two-Mode Architecture

1. **Fetch Mode** (`mine fetch`): Search upstream sources using their search APIs, retrieve complete threads, store locally
2. **Select Mode** (`mine select`): Query and analyze locally cached data

### Three-Layer Data Model

```
Raw Layer (source-specific JSON)
    ↓
Normalized Layer (common schema)
    ↓
Analysis Layer (annotations, classifications, relationships)
```

## Architecture Principles

- **Search-first**: Use source search APIs rather than list/get APIs
- **Complete threads**: Always fetch entire conversation threads, not just individual messages
- **SQLite storage**: All data in a single SQLite database for performance and portability
- **Rate-limited**: Respect API limits (self-limit to 1/2 or 1/3 of published rates)
- **Normalized output**: Common schema across all sources for consistent analysis

## Command Structure

### Fetch Commands

```bash
# Slack: Search-based message fetching
mine fetch slack --workspace TEAM --user alice --channel general --since 7d
mine fetch slack --workspace TEAM --search "kubernetes" --since 30d

# GitHub: Search issues and PRs
mine fetch github --repo org/repo --label bug --since 30d
mine fetch github --repo org/repo --author alice --type pr --since 7d
mine fetch github --repo org/repo --reviewer bob --type pr
```

### Select Commands

```bash
# Query by author and time
mine select --author alice --since 7d

# Full-text search
mine select --search "kubernetes"

# Multi-participant threads
mine select --author alice --author bob --author charlie

# Output formats
mine select --search "error" --format table
mine select --thread thread_123 --format graph
mine select --author alice --since 30d --format jsonl | jq '.content'
```

## Database Schema

### Key Tables

- **raw_messages**: Source-specific data as received from APIs
- **messages**: Normalized messages with common fields
- **users**: User profiles across all sources
- **channels**: Channels, repos, and containers
- **workspaces**: Slack workspaces, GitHub orgs, email accounts
- **identities**: Canonical identities linking users across sources
- **classifications**: Message annotations (question, answer, solution, etc.)
- **message_relations**: Relationships between messages
- **rate_limits**: API rate limiting state

### Normalized Message Schema

```go
type Message struct {
    ID           string       // Universal ID: msg_slack_*, msg_github_*
    SourceType   string       // slack, github, email
    SourceID     string       // Original source identifier
    Timestamp    time.Time
    AuthorID     string       // Foreign key to users.id
    Content      string       // Plain text
    ContentHTML  *string      // Rich format
    ChannelID    string       // Foreign key to channels.id
    ThreadID     *string      // Thread root message ID
    ParentID     *string      // Direct parent message ID
    IsThreadRoot bool
    Mentions     []string     // JSON array of user IDs
    URLs         []string
    CodeBlocks   []CodeBlock
    Attachments  []Attachment
}
```

## Source-Specific Requirements

### Slack

- Use search API (`search.messages`) for fetching
- For each result:
  - If message has `thread_ts`, fetch complete thread via `conversations.replies`
  - If message is thread root (`ts == thread_ts`), fetch all replies
- Rate limiting:
  - Tier 2: 20 requests/minute → self-limit to 10 requests/minute
  - Tier 3: 50 requests/minute → self-limit to 25 requests/minute
  - Track per-workspace, per-endpoint
- Cache workspace user IDs, channel details

### GitHub

- Use search API (`/search/issues`, `/search/commits`)
- For each issue/PR:
  - Fetch all comments (`GET /repos/{owner}/{repo}/issues/{number}/comments`)
  - For PRs: Fetch review comments (`GET /repos/{owner}/{repo}/pulls/{number}/comments`)
  - For PRs: Fetch reviews (`GET /repos/{owner}/{repo}/pulls/{number}/reviews`)
  - Fetch timeline (`GET /repos/{owner}/{repo}/issues/{number}/timeline`)
- For discussions (future):
  - Use GraphQL API
  - Fetch all comments and nested replies
- Remember: PR "comments" vs "review comments" are different endpoints

### Email (Future)

- IMAP or local mbox files
- Thread using References/In-Reply-To headers
- Store attachments metadata only

## Output Formats

### JSON (default)
```json
[
  {
    "id": "msg_slack_C123_1234567890.123456",
    "source_type": "slack",
    "timestamp": "2025-12-15T10:30:00Z",
    "author_id": "user_slack_U123",
    "content": "How do I configure rate limiting?",
    "channel_id": "chan_slack_C123"
  }
]
```

### JSONL (streaming)
One message per line, suitable for piping to `jq` or other tools.

### Table (human-readable)
```
TIMESTAMP           AUTHOR        CHANNEL    CONTENT
2026-02-07 10:30   user_alice    general    How do I configure rate limiting?
2026-02-07 10:35   user_bob      general    Check the docs at ...
```

### Graph (visualization)
```json
{
  "nodes": [
    {"id": "msg_123", "type": "message", "content": "...", "timestamp": "..."}
  ],
  "edges": [
    {"from": "msg_124", "to": "msg_123", "type": "reply_to"}
  ]
}
```

## Implementation Status

### Completed
- ✅ SQLite schema design
- ✅ Database layer (internal/db)
- ✅ Command structure (fetch/select)
- ✅ Select query engine with FTS
- ✅ Basic rate limiting tracking

### In Progress
- 🚧 Slack search API integration
- 🚧 GitHub search API integration
- 🚧 Complete thread fetching

### Planned
- 📋 Slack thread fetching with rate limiting
- 📋 GitHub timeline fetching
- 📋 GitHub Discussions support
- 📋 Classification engine
- 📋 Identity resolution
- 📋 Email support

## Development Guidelines

- **Database-first**: All data goes through the database layer
- **Atomic operations**: Use transactions for multi-step operations
- **Rate limiting**: Always check rate limits before API calls
- **Complete threads**: Never store partial threads
- **Idempotent fetches**: Re-fetching same data should be safe
- **Schema versioning**: Support database migrations

## File Structure

```
~/.threadmine/
├── threadmine.db          # SQLite database (all data)
└── logs/                  # Optional logging

internal/
├── db/                    # Database layer
│   ├── schema.sql
│   ├── db.go
│   ├── messages.go
│   ├── users.go
│   ├── channels.go
│   ├── annotations.go
│   └── ratelimit.go
├── slack/                 # Slack integration
├── github/                # GitHub integration
├── normalize/             # Normalization logic
└── classify/              # Classification engine

cmd/mine/commands/
├── root.go
├── fetch.go
└── select.go
```

## Configuration

Configuration is minimal - most settings are command-line flags.

```bash
# Database location (default: ~/.threadmine/threadmine.db)
mine --db /path/to/db.db select --search foo

# Output format
mine select --format table
mine select --format jsonl | jq '.content'
```

## Examples

### Example 1: Find Your Recent Questions
```bash
# Fetch your recent Slack messages
mine fetch slack --workspace myteam --user me --since 7d

# Find questions you asked
mine select --author user_slack_U123 --since 7d | \
  jq '.[] | select(.classifications[]?.type == "question")'
```

### Example 2: Track GitHub Issue Discussion
```bash
# Fetch issue and comments
mine fetch github --repo org/repo --search "authentication" --since 30d

# Select and analyze
mine select --search "authentication" --source github --format table
```

### Example 3: Multi-user Conversation Analysis
```bash
# Fetch from multiple sources
mine fetch slack --workspace myteam --channel engineering --since 14d
mine fetch github --repo org/repo --since 14d

# Find conversations involving specific users
mine select --author alice --author bob --format graph > conversation.json
```

## Design Decisions

### Why SQLite?
- Single-file database, easy backup
- Fast full-text search (FTS5)
- Transaction support
- No server required
- Better performance than JSON files for queries

### Why Search APIs?
- More flexible than list APIs
- Matches user mental model ("find X")
- Naturally supports complex queries
- Less data transfer (fetch only what matches)

### Why Complete Threads?
- Context is critical for understanding
- Partial threads are confusing
- Analysis requires full conversation
- Storage is cheap, API calls are expensive

### Why Rate Limiting?
- Respect upstream services
- Avoid getting blocked
- Self-limit below published rates for safety
- Track per-workspace, per-endpoint

---

**Repository**: https://github.com/solvaholic/threadmine
**License**: MIT
**Language**: Go 1.25+
