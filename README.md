# ThreadMine

Search and analyze conversations across Slack, GitHub, and email - no admin required.

## Overview

ThreadMine (`mine`) is a command-line tool for message ingest, storage, and retrieval. It fetches conversations from multiple platforms (Slack, GitHub, email), stores them locally in SQLite with basic enrichment metadata, and provides query capabilities for downstream analysis tools.

**Two-mode architecture:**
- **Fetch**: Search upstream sources and retrieve complete threads with basic enrichment
- **Select**: Query and filter locally cached data

ThreadMine focuses on data collection. Advanced analysis (NLP, graph analysis, generative AI) is handled by separate tools that consume ThreadMine's data.

## Quick Start

```bash
# Build with FTS5 full-text search support
make build
# Or manually: go build -tags "fts5" -o mine ./cmd/mine

# Fetch from Slack (search-based)
./mine fetch slack --workspace myteam --user alice --since 7d
./mine fetch slack --workspace myteam --search "kubernetes" --since 30d --threads

# Fetch from GitHub (search-based)
./mine fetch github --repo org/repo --label bug --since 30d
./mine fetch github --repo org/repo --author alice --type pr

# Query local data
./mine select --author alice --since 7d
./mine select --search "error" --format table
./mine select --thread thread_123 --format graph

# Help
./mine --help
./mine fetch --help
./mine select --help
```

## Configuration

ThreadMine supports optional configuration via `~/.threadmine/config`.

Configuration provides convenient defaults for commonly-used flags. CLI flags always override config values.

**Example config** (`~/.threadmine/config`):

```ini
[fetch.slack]
    workspace = myteam
    search = from:@myusername
    since = 7d

[fetch.github]
    org = myorg
    commenter = myusername
    limit = 999
    since = 7d

[select]
    author = myusername
    since = 7d
    format = table
```

**Priority order**: Built-in defaults → Config file values → CLI flags (flags always win)

See [`docs/config.example`](docs/config.example) for all available configuration options.

## Key Features

- **Search-first**: Uses source search APIs (Slack `search.messages`, GitHub `/search/issues`)
- **Complete threads**: Optionally fetches entire conversation threads with `--threads` flag
- **SQLite storage**: Fast queries with FTS5 full-text search (boolean queries, phrase matching, relevance ranking)
- **Rate limiting**: Self-limits to 1/2 or 1/3 of API rate limits to avoid abuse
- **Multiple formats**: JSON (default), JSONL (streaming), table (human-readable), graph (visualization)
- **Cross-platform**: Unified schema across Slack, GitHub, and email (planned)

## Architecture

### Two-Mode Design

1. **Fetch Mode**: Search upstream sources → Retrieve complete threads → Store in database
2. **Select Mode**: Query database → Apply filters → Output results

### Three-Layer Data Model

```
Raw Layer (source-specific JSON in database)
    ↓
Normalized Layer (common schema)
    ↓
Enrichment Layer (basic metadata: question flags, counts, content features)
```

### Storage

All data stored in `~/.threadmine/threadmine.db` (SQLite):
- Raw messages as received from APIs
- Normalized messages in common schema
- User profiles and identity mappings
- Basic enrichment metadata
- Rate limiting state

## Command Reference

### Fetch Commands

```bash
# Slack
mine fetch slack --workspace TEAM --user alice --channel general --since 7d
mine fetch slack --workspace TEAM --search "kubernetes" --since 30d --threads

# GitHub
mine fetch github --repo org/repo --label bug --since 30d
mine fetch github --repo org/repo --author alice --type pr --since 7d
mine fetch github --repo org/repo --reviewer bob --type pr
```

### Select Commands

```bash
# Filter by author and time
mine select --author alice --since 7d

# Full-text search (FTS5 with advanced query syntax)
mine select --search "kubernetes"
mine select --search "kubernetes AND deployment"
mine select --search "error OR failure"
mine select --search '"exact phrase"'
mine select --search "deploy*"  # Prefix matching

# Multi-participant threads
mine select --author alice --author bob --author charlie

# Filter by source
mine select --source slack --since 30d
mine select --source github --search "bug"

# Enrichment filters
mine select --is-question --author alice --since 7d
mine select --has-code --search "implementation"
mine select --has-links --since 30d
mine select --has-quotes --source slack

# Output formats
mine select --search "error" --format table
mine select --thread thread_123 --format graph
mine select --author alice --since 30d --format jsonl | jq '.content'

# Pagination
mine select --search "foo" --limit 50 --offset 100
```

## Output Formats

### JSON (default)
```json
[
  {
    "id": "msg_slack_C123_1234567890.123456",
    "source_type": "slack",
    "timestamp": "2026-02-07T10:30:00Z",
    "author_id": "user_slack_U123",
    "content": "How do I configure rate limiting?"
  }
]
```

### JSONL (streaming)
One message per line, pipe-friendly:
```bash
mine select --search "error" --format jsonl | jq '.content'
```

### Table (human-readable)
```
TIMESTAMP           AUTHOR        CHANNEL    CONTENT
2026-02-07 10:30   user_alice    general    How do I configure...
2026-02-07 10:35   user_bob      general    Check the docs at...
```

### Graph (visualization)
```json
{
  "nodes": [{"id": "msg_123", "content": "...", "timestamp": "..."}],
  "edges": [{"from": "msg_124", "to": "msg_123", "type": "reply_to"}]
}
```

## Building

The project includes a Makefile that automatically includes the FTS5 build tag:

```bash
# Build the binary (default target)
make build

# Install to $GOPATH/bin
make install

# Run tests
make test

# Run tests with coverage
make test-coverage

# Clean build artifacts
make clean

# Show all available targets
make help
```

All build commands automatically include `-tags "fts5"` for full-text search support.

## Examples

### Find your recent questions
```bash
# Fetch your Slack messages with threads
mine fetch slack --workspace myteam --user me --since 7d --threads

# Query for questions using enrichment filter
mine select --author user_slack_U123 --since 7d --is-question
```

### Track GitHub issue discussion
```bash
# Fetch issues with keyword
mine fetch github --repo org/repo --search "authentication" --since 30d

# View as table
mine select --search "authentication" --source github --format table
```

### Analyze multi-user conversations
```bash
# Fetch from multiple sources with threads
mine fetch slack --workspace myteam --channel engineering --since 14d --threads
mine fetch github --repo org/repo --since 14d

# Find conversations between specific users
mine select --author alice --author bob --format graph > conversation.json
```

## Current Status

**Completed:**
- ✅ SQLite schema and database layer
- ✅ Command structure (fetch/select)
- ✅ FTS5 full-text search with boolean queries, phrase matching, and relevance ranking
- ✅ Rate limiting framework
- ✅ Slack search API integration with thread fetching
- ✅ GitHub search API integration (issues, PRs, comments, reviews, timeline)
- ✅ GitHub Discussions support
- ✅ Human-readable name resolution in select output
- ✅ Basic enrichment engine
  - Question detection, character/word counts, quote/code/link flags
  - Code block extraction (fenced, inline, HTML)
  - URL extraction
  - Automatic enrichment during fetch
- ✅ Select command enrichment filters
  - --is-question: Filter to messages that look like questions
  - --has-code: Filter to messages containing code blocks
  - --has-links: Filter to messages containing URLs
  - --has-quotes: Filter to messages containing quote blocks

**In Progress:**
- 🔨 (No active work items)

**Planned:**
- 📋 Cross-platform identity resolution (email-based matching)
- 📋 Email support (IMAP/mbox)

## Documentation

- [docs/SPEC.md](docs/SPEC.md) - Complete specification (v2.0)
- [internal/db/schema.sql](internal/db/schema.sql) - Database schema
- [.github/copilot-instructions.md](.github/copilot-instructions.md) - Development guide

## Requirements

- Go 1.25+
- Make (for automated building)
- SQLite with FTS5 support (via `github.com/mattn/go-sqlite3`)
  - Automatically enabled with `make build`
- Slack desktop app (for cookie-based auth)
- GitHub CLI (`gh`) for GitHub authentication

## License

MIT
