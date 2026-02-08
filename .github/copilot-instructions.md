# ThreadMine - Copilot Instructions

## Project Overview

ThreadMine (`mine` CLI) is a Go-based tool for searching and analyzing conversations across Slack, GitHub, and email. It uses search APIs to fetch messages, stores them in a local SQLite database, and provides query capabilities for cross-platform analysis without requiring admin privileges.

## Architecture Principles

- **Two-mode architecture**: Fetch (search upstream sources) and Select (query local database)
- **Search-first**: Use source search APIs rather than list/get APIs for fetching
- **SQLite storage**: All data in a single SQLite database with FTS5 for full-text search
- **Complete threads**: Always fetch entire conversation threads, not partial data
- **Rate limiting**: Self-limit to 1/2 or 1/3 of published API rates
- **Read-only**: Never send/create messages, only fetch and analyze

## Tech Stack

- **Language**: Go (idiomatic Go patterns)
- **CLI**: `github.com/spf13/cobra`
- **Database**: SQLite via `github.com/mattn/go-sqlite3`
- **Data Sources**:
  - Slack: `github.com/rneatherway/slack` (cookie-based auth)
  - GitHub: GitHub CLI (`gh`) for auth/API calls
  - Email: `github.com/emersion/go-imap` (planned)
- **Output**: JSON/JSONL/table to stdout, errors to stderr

## Key Conventions

- Universal IDs: `msg_slack_*`, `user_github_*`, `chan_slack_*`
- Database-first: All data goes through `internal/db` package
- Atomic transactions for multi-step operations
- Comprehensive error context for debugging
- Schema versioning for forward compatibility

## Code Guidelines

- **Package organization**:
  - `internal/db`: Database layer with schema and CRUD operations
  - `internal/slack`, `internal/github`: Source integrations
  - `internal/normalize`: Normalization logic
  - `internal/classify`: Classification engine
  - `cmd/mine/commands`: CLI commands (root, fetch, select)

- **Database operations**:
  - Always use transactions for multi-step operations
  - Check rate limits before API calls
  - Store raw JSON in `raw_messages` table
  - Normalize and store in `messages` table
  - Use FTS5 for full-text search

- **Error handling**: Always wrap errors with `fmt.Errorf` using `%w` for error chains; provide user-friendly context

- **Rate limiting**:
  - Check `rate_limits` table before API calls
  - Record each successful request
  - Self-limit to 1/2 or 1/3 of published rates
  - Track per-workspace, per-endpoint

- **Thread fetching**:
  - Slack: Use `search.messages` then fetch complete threads via `conversations.replies`
  - GitHub: Fetch issue/PR with all comments, review comments (PRs), and timeline
  - Never store partial threads

- **Never store credentials**: Use system keychains or environment variables only

## Development Workflow

- **Main entry point**: `cmd/mine/main.go` calls `commands.Execute()`
- **Commands**: `fetch` and `select` subcommands with source-specific sub-subcommands
- **Database schema**: Defined in `internal/db/schema.sql`, applied on first run
- **Testing**: Build features incrementally, test with real data
- **Migrations**: Schema versioning for forward compatibility

## Command Structure

```bash
# Fetch from upstream sources (search-based)
mine fetch slack --workspace TEAM --user alice --since 7d
mine fetch github --repo org/repo --label bug --since 30d

# Query local database
mine select --author alice --since 7d
mine select --search "kubernetes" --format table
mine select --thread thread_123 --format graph
```

## Focus Areas

When implementing features, prioritize:
1. Database integrity and schema consistency
2. API rate limiting (prevent abuse)
3. Complete thread fetching (no partial data)
4. Clear error messages and logging
5. Search-first approach (use search APIs)
6. macOS-focused development (cross-platform is future goal)
