# GitHub Integration Guide

## Overview

ThreadMine supports extracting and analyzing conversations from GitHub issues and pull requests. This enables cross-platform conversation analysis across Slack, GitHub, and other sources.

## Prerequisites

1. **GitHub CLI (`gh`)**: ThreadMine uses the GitHub CLI for authentication and API access.
   - Install from: https://cli.github.com/
   - Authenticate: `gh auth login`

2. **Repository Access**: You need read access to the repositories you want to analyze.

## Quick Start

### Authenticate with GitHub

```bash
# Authenticate using GitHub CLI (one-time setup)
gh auth login

# Verify authentication
gh auth status
```

### Fetch Data from a Repository

```bash
# Fetch issues and PRs from the last 30 days
./mine fetch github --owner solvaholic --repo threadmine --since 30d

# Fetch issues and PRs from a specific date
./mine fetch github --owner myorg --repo myrepo --since 2025-12-01
```

### Query GitHub Messages

```bash
# Get all GitHub messages
./mine messages --source github

# Search for specific content
./mine messages --source github --search "bug"

# Get messages since a specific date
./mine messages --source github --since 7d
```

## What Gets Fetched

ThreadMine fetches the following from each repository:

1. **Issues**
   - Issue body (converted to the root message of a thread)
   - All issue comments (as replies in the thread)

2. **Pull Requests**
   - PR description (converted to the root message of a thread)
   - All PR comments (as replies)
   - All PR reviews (as replies)

## Data Storage

GitHub data is stored in the ThreadMine cache following the SPEC.md structure:

### Raw Data
```
~/.threadmine/raw/github/repos/{owner}-{repo}/
├── issues/
│   ├── _index.json          # List of all issues
│   └── {number}.json        # Individual issue
├── pull_requests/
│   ├── _index.json          # List of all PRs
│   └── {number}.json        # Individual PR
└── comments/
    ├── issue-{number}/
    │   └── comments.json    # Issue comments
    └── pr-{number}/
        ├── comments.json    # PR comments
        └── reviews.json     # PR reviews
```

### Normalized Data
```
~/.threadmine/normalized/messages/
└── by_source/
    └── github.jsonl         # All GitHub messages in normalized format
```

## Normalization Details

### Issues
- **Issue → Thread Root**: Each issue becomes the root message of a thread
- **Issue Comments → Replies**: Comments become child messages in the thread
- **Channel Representation**: Each issue is treated as its own channel/container
- **Thread ID**: `thread_github_{owner}_{repo}_issue_{number}`
- **Message ID**: `msg_github_{owner}_{repo}_issue_{number}`

### Pull Requests
- **PR → Thread Root**: Each PR becomes the root message of a thread
- **PR Comments → Replies**: Comments become child messages
- **PR Reviews → Replies**: Reviews become child messages
- **Channel Representation**: Each PR is treated as its own channel/container
- **Thread ID**: `thread_github_{owner}_{repo}_pr_{number}`
- **Message ID**: `msg_github_{owner}_{repo}_pr_{number}`

### User Mapping
- **User ID**: `user_github_{login}`
- **Source ID**: GitHub user ID (numeric)
- **Display Name**: GitHub username (login)
- **Real Name**: User's full name (if available)
- **Email**: User's email (if public)

## Message Enrichment

GitHub messages receive basic enrichment metadata during fetch:

- **Question flag**: Pattern matching for question marks, question starters, and help-seeking phrases, without confidence scoring
- **Counts**: Content length and whitespace-separated word count
- **Content flags**: Code blocks and links extracted during normalization, plus markdown-style block quotes

Answers, solutions, and acknowledgments are not classified. Advanced analysis belongs in downstream tools; see the Message Enrichment section of [SPEC.md](SPEC.md).

Run the local enrichment demo without fetching data:

```bash
go run ./cmd/enrich-demo
```

## Graph Analysis

GitHub conversations are included in ThreadMine's reply graph:

- **Nodes**: Each issue, PR, comment, and review
- **Edges**: Parent-child relationships (issue → comment, PR → review)
- **Thread Hierarchies**: Full conversation trees for each issue/PR

## Examples

### Find All Questions in a Repository

```bash
# Fetch data
./mine fetch github --repo facebook/react

# Query for questions using the enrichment filter
./mine select --source github --is-question
```

### Analyze PR Review Conversations

```bash
# Fetch PRs
./mine fetch github --owner microsoft --repo vscode --since 7d

# Find PRs with many replies (active discussions)
./mine messages --source github | jq '.messages[] | select(.source_metadata.pr_number != null and .is_thread_root == true)'
```

### Cross-Source Analysis

```bash
# Fetch from both Slack and GitHub
./mine fetch slack --workspace myteam
./mine fetch github --owner myorg --repo myrepo

# Find all messages mentioning "deployment"
./mine messages --search "deployment"
```

## Cache Management

### Cache TTL
- **Issues/PRs**: Cached for 1 hour
- **Comments/Reviews**: Cached for 1 hour

### Clear Cache
```bash
# Remove GitHub cache
rm -rf ~/.threadmine/raw/github/
rm -rf ~/.threadmine/normalized/messages/by_source/github.jsonl
```

## Limitations

### Current Limitations
1. **No PR Review Comments**: Code review comments on specific lines are not yet supported
2. **No Reactions**: GitHub reactions (👍, ❤️, etc.) are not captured
3. **No Commit Messages**: Individual commits are not fetched
4. **Single Repository**: Must fetch one repository at a time

### Future Enhancements
- Support for GitHub Discussions
- Reaction and emoji support
- Batch fetching for multiple repositories
- Organization-wide fetching
- Commit message extraction

## Troubleshooting

### "GitHub CLI (gh) not found"
Install the GitHub CLI: https://cli.github.com/

### "GitHub CLI authentication failed"
Run `gh auth login` to authenticate with GitHub.

### "failed to fetch repository"
- Verify you have access to the repository
- Check the owner and repo names are correct
- Ensure the repository exists and is not archived

### No Issues or PRs Found
- Check the `--since` date range
- Verify the repository has issues/PRs in that timeframe
- Use `--since 365d` or omit `--since` to fetch all data

## API Rate Limits

ThreadMine uses the GitHub CLI's authenticated API access, which has the following limits:

- **Authenticated**: 5,000 requests per hour
- **Caching**: ThreadMine implements aggressive caching to minimize API calls

The tool will automatically handle rate limiting by:
1. Using cached data when available
2. Implementing exponential backoff if rate limited

## See Also

- [SPEC.md](../../docs/SPEC.md) - Full ThreadMine specification
- [README.md](../../README.md) - Project overview
- [internal/normalize/README.md](../internal/normalize/README.md) - Normalization layer details
