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

All compilation, vet, and test targets require CGO and append `-tags "fts5" -mod=readonly`. They preserve `GOFLAGS` for other options; use `TAGS` for additional build tags. `GO` selects the Go executable, and `GOTOOLCHAIN` defaults to `local` so automatic toolchain downloads cannot hide minimum-version failures.

### Local checks

Run `make check` before proposing a change. It checks formatting without modifying sources, verifies cached modules, runs vet and race-enabled tests, builds every package (including demos), and builds and runs the production CLI with `--help` in a temporary directory. Failures stop the command with a nonzero exit status; missing Go, formatter, C compiler, or disabled CGO are errors, not skipped checks. No Slack or GitHub credentials are required. The database tests open the real embedded schema in temporary directories and exercise FTS5 search and insert/update/delete index synchronization.

| Target | Behavior |
| --- | --- |
| `check-fmt` | List unformatted Go files and fail without rewriting them; includes new sources, excludes `.git` and `vendor` directories |
| `verify-modules` | Run `go mod verify` without changing manifests |
| `vet` | Run `go vet` on all packages |
| `test` | Run ordinary tests, without the race detector |
| `test-race` | Run all tests with the race detector |
| `build-all` | Compile all packages without leaving binaries in the checkout |
| `smoke` | Build a temporary CLI and execute `--help` |
| `check` | Run the deterministic local checks above, using `test-race` instead of a redundant ordinary test run |
| `vuln` | Opt-in, network-dependent production CLI vulnerability scan |

`fmt` still rewrites sources; `build`, `install`, `test-coverage`, `run`, `clean`, `lint`, and `help` retain their existing roles. `test-coverage` writes `coverage.out` and `coverage.html`; `clean` removes both.

Use separate local invocations for the minimum Go version (exactly 1.26.0) and the current stable release, with CGO enabled and a working C compiler:

```bash
GOTOOLCHAIN=local make check GO=/path/to/go1.26.0/bin/go
GOTOOLCHAIN=local make check GO=/path/to/current-stable/bin/go
make test TAGS=additional_tag GOFLAGS=-count=1
```

`make vuln` runs pinned `golang.org/x/vuln/cmd/govulncheck@v1.9.0` (requires Go 1.26.0 or newer) with `-tags fts5 -show verbose ./cmd/mine`. The versioned `go run` keeps tool dependencies separate from this module's manifests. It needs network access to acquire the tool and query the vulnerability database. The default text-mode exit status gates reachable findings; it neither suppresses OpenPGP findings nor treats tool failures as success. Run it on macOS to scan the production CLI's macOS dependency graph. It is intentionally excluded from `check`.

Go 1.26.0 is a minimum compatibility check, not a recommended security baseline. Use a current patched toolchain for `make vuln`, for example `make vuln GO=/path/to/current-patched/bin/go`. During macOS validation on 2026-11-05, the Go 1.26.0 scan reported 26 reachable standard-library vulnerabilities and failed; Go 1.27.3 reported zero reachable vulnerabilities and passed, while still displaying the non-reachable OpenPGP module advisory GO-2026-5932. Standard-library findings require a patched Go toolchain, not a module dependency upgrade.

These are local commands intended for reuse by future Linux-only CI. They do not add workflows or consume hosted runners. macOS remains the primary development platform; Linux full-CLI compatibility is not established by passing checks on macOS. The existing Slack authentication dependency imports macOS keychain code, so a Linux build may require a separate portability change before CI can use the full aggregate.

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

- Go 1.26.0 or newer
- Make (for automated building)
- CGO enabled (`CGO_ENABLED=1`) and a C compiler on `PATH`
- SQLite with FTS5 support (via `github.com/mattn/go-sqlite3`)
  - Automatically enabled with `make build`
- Slack desktop app (for cookie-based auth)
- GitHub CLI (`gh`) for GitHub authentication

## License

MIT
