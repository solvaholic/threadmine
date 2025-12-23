# ThreadMine - Copilot Instructions

## Project Overview

ThreadMine (`mine` CLI) is a Go-based tool for extracting, caching, and analyzing conversations across Slack, GitHub, and email. It provides unified cross-platform conversation analysis without requiring admin privileges.

## Architecture Principles

- **Three-layer architecture**: Raw (source-specific) → Normalized (common schema) → Analysis (annotations/graph)
- **Cache-aside pattern**: Always check cache first, fetch from API on miss
- **Filesystem storage**: Human-readable JSON/JSONL files in `~/.threadmine/`
- **Immutable history**: Cached messages never expire; only metadata has TTL
- **Read-only**: Never send/create messages, only fetch and analyze

## Tech Stack

- **Language**: Go (idiomatic Go patterns)
- **CLI**: `github.com/spf13/cobra`
- **Config**: `github.com/spf13/viper`
- **Data Sources**: 
  - Slack: `github.com/rneatherway/slack` (cookie-based auth)
  - GitHub: GitHub CLI (`gh`) for auth/API calls
  - Kusto: `github.com/Azure/azure-kusto-go` with Azure CLI (`az`) for auth
  - Email: `github.com/emersion/go-imap`
- **Output**: JSON to stdout, errors to stderr

## Key Conventions

- Universal IDs: `msg_slack_*`, `user_github_*`, `thread_*`
- Atomic file writes: write to temp, then rename
- Comprehensive error context for debugging
- All commands output valid JSON (use `--format json|jsonl|table`)
- Schema versioning for forward compatibility

## Code Guidelines

- **Package organization**: Use `internal/` for application packages (`internal/slack`, `internal/cache`, `internal/normalize`, etc.)
- **Error handling**: Always wrap errors with `fmt.Errorf` using `%w` for error chains; provide user-friendly context
- **File operations**: 
  - Atomic writes: `os.WriteFile(temp)` → `os.Rename(temp, target)`
  - File permissions: 0600 (files), 0700 (directories)
  - JSON formatting: Use `json.MarshalIndent` with 2-space indent for human readability
- **Struct conventions**: 
  - Export fields that need JSON serialization
  - Always include JSON tags with appropriate options
  - Document exported types and functions
- **Rate limiting**: Implement exponential backoff for API calls
- **Never store credentials**: Use system keychains or environment variables only

## Development Workflow

- **Test harnesses**: It's acceptable to use `cmd/mine/main.go` as a temporary test script during feature development
- **Incremental implementation**: Build features one layer at a time (raw → normalized → analysis)
- **Verify cache structure**: Always check the filesystem output matches the SPEC directory structure
- **Check permissions**: Verify 0600/0700 permissions on created files/directories

## Focus Areas

When implementing features, prioritize:
1. Cache integrity and performance
2. API abuse prevention (respect rate limits)
3. Clear error messages and logging
4. Schema consistency across sources
5. macOS-focused development (cross-platform support is a future goal)
