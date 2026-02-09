# ThreadMine Specification

**Version**: 2.0 (Redesign)
**Status**: In Development

## Overview

ThreadMine (`mine`) is a command-line tool for message ingest, storage, and retrieval. It fetches conversations from Slack, GitHub, and email using search APIs, stores them in a local SQLite database with basic enrichment metadata, and provides query capabilities for downstream analysis.

ThreadMine focuses on data collection and basic enrichment. Advanced natural language processing, graph analysis, and generative AI work are handled by separate analysis tools that consume ThreadMine's data.

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
Enrichment Layer (basic metadata: question flags, counts, content features)
```

## Architecture Principles

- **Search-first**: Use source search APIs rather than list/get APIs
- **Complete threads**: Optionally fetch entire conversation threads with `--threads` flag
- **SQLite storage**: All data in a single SQLite database for performance and portability
- **Rate-limited**: Respect API limits (self-limit to 1/2 or 1/3 of published rates)
- **Normalized output**: Common schema across all sources for consistent analysis

## Command Structure

### Fetch Commands

```bash
# Slack: Search-based message fetching
mine fetch slack --workspace TEAM --user alice --channel general --since 7d
mine fetch slack --workspace TEAM --search "kubernetes" --since 30d --threads

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
- **enrichments**: Basic message metadata (question flags, counts, content features)
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

### Message Enrichment

ThreadMine performs basic content analysis during message ingest to add coarse-grained metadata. This enrichment is stored in the `enrichments` table and provides quick filters for downstream analysis tools.

**Enrichment fields:**
- `is_question`: Boolean flag indicating if the message looks like a question
  - Detected using: question marks, question words (how, what, why, when, where, who), help-seeking phrases
  - Uses pattern matching without confidence scoring
- `char_count`: Total character count of message content
- `word_count`: Total word count of message content
- `has_code`: Boolean flag indicating code block presence
  - Extracts: fenced code blocks (```), inline code (`), HTML code tags (<code>)
  - Language detection intentionally omitted (unreliable across Slack/GitHub/markdown)
- `has_links`: Boolean flag indicating URL presence
  - Extracts URLs from message content
- `has_quotes`: Boolean flag indicating markdown-style block quotes (lines starting with '>')

**Code block and URL extraction:**
- Code blocks stored in `messages.code_blocks` (JSON array)
- URLs stored in `messages.urls` (JSON array)
- Extraction happens during normalization in `internal/normalize/extract.go`

Advanced analysis (sentiment, topic modeling, entity extraction, semantic classification, etc.) is performed by external tools that query ThreadMine's database.

## Source-Specific Requirements

### Slack

- Use search API (`search.messages`) for fetching
- Thread fetching (opt-in with `--threads` flag):
  - Extract `thread_ts` from message or permalink
  - If `--threads` enabled and message has `thread_ts`, fetch complete thread via `conversations.replies`
  - Without `--threads`, store only individual search results
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
- ✅ Select query engine with LIKE-based search
- ✅ Rate limiting tracking (per workspace, per endpoint)
- ✅ Slack search API integration
- ✅ Slack complete thread fetching with rate limiting
- ✅ GitHub search API integration (issues and PRs)
- ✅ GitHub complete data fetching (comments, review comments, reviews, timeline)
- ✅ GitHub Discussions support (GraphQL API integration with nested replies)
- ✅ Human-readable name resolution in table output
- ✅ Smart channel name handling (prefixes, IDs, DMs)
- ✅ Basic enrichment engine
  - Question detection (using patterns: question marks, question words, help-seeking phrases)
  - Code block extraction (fenced blocks, inline code, HTML code tags)
  - URL extraction
  - Character and word counts
  - Quote block detection (markdown-style '>' quotes)
  - Automatic enrichment during message fetch

### In Progress
- 🔨 Select command enrichment filters (--is-question, --has-code, etc.)

### Planned
- 📋 Cross-platform identity resolution (email-based matching)
- 📋 Email support (IMAP/mbox)
- 📋 FTS5 full-text search (requires sqlite3 build with FTS5)

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
# Fetch your recent Slack messages with threads
mine fetch slack --workspace myteam --user me --since 7d --threads

# Find questions you asked (enrichment filters coming soon)
mine select --author user_slack_U123 --since 7d --is-question
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
# Fetch from multiple sources with threads
mine fetch slack --workspace myteam --channel engineering --since 14d --threads
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
