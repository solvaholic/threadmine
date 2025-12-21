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

- Standard Go project layout
- Include extensive error messages with actionable guidance
- Handle rate limits with exponential backoff
- Use restrictive permissions (600/700) for cache files
- Never store credentials in cache

## Focus Areas

When implementing features, prioritize:
1. Cache integrity and performance
2. Cross-platform compatibility (macOS, Linux, Windows)
3. API abuse prevention (respect rate limits)
4. Clear error messages and logging
5. Schema consistency across sources
