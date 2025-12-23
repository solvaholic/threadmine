# threadmine
Extract and analyze multi-platform conversations, with zero admin overhead

## Features

ThreadMine (`mine` CLI) is a Go-based tool that:

- **Extracts conversations** from Slack, GitHub, and email using local credentials
- **Caches data** efficiently to minimize API calls and respect rate limits
- **Normalizes messages** across platforms into a common schema
- **Builds reply graphs** to track conversation threads and relationships
- **Classifies messages** using heuristics to identify questions, answers, solutions, and acknowledgments

## Current Status

✅ **MVP Steps Completed:**
1. ✅ Authenticate with Slack using browser cookies
2. ✅ Fetch and cache messages from Slack channels
3. ✅ Normalize Slack messages to common schema
4. ✅ Build basic message reply graph
5. ✅ Implement cache-aside pattern for message retrieval
6. ✅ Classify messages as questions or answers using heuristics
7. ✅ Output valid JSON for all commands

**Next Steps:**
8. Cross-platform support (Linux, Windows)

## Quick Start

```bash
# Build the tool
go build -o mine ./cmd/mine

# Fetch Slack messages from a workspace
./mine fetch slack --workspace solvahol

# Query messages
./mine messages --source slack --since 2025-12-20

# Search for specific content
./mine messages --search "error" --source slack

# View cache information
./mine cache info

# Get help for any command
./mine --help
./mine fetch --help
```

### Available Commands

- `mine fetch slack` - Fetch data from Slack workspaces
- `mine messages` - Query normalized messages with filters
- `mine cache info` - Show cache statistics and storage info

All commands output valid JSON by default. Use `--format json` (default), `jsonl`, or `table` for different output formats.

## Architecture

ThreadMine uses a three-layer architecture:

```
Raw Layer (source-specific formats)
    ↓
Normalized Layer (common schema)
    ↓
Analysis Layer (annotations, graph, insights)
```

### Data Storage

All data is stored in `~/.threadmine/`:

- **`raw/`** - Source-specific API responses (JSON)
- **`normalized/`** - Common schema messages (JSON/JSONL)
- **`graph/`** - Reply graphs and thread structures (JSON)
- **`annotations/`** - Message classifications and analysis (JSON)

### Message Classification

Heuristic-based classification identifies:

- **Questions** - Question marks, help-seeking phrases, question starters
- **Answers** - Responses in question threads with answer indicators
- **Solutions** - Code blocks, step-by-step instructions, documentation links
- **Acknowledgments** - Thanks, success confirmations, positive reactions

Each classification includes confidence scores (0.0-1.0) and signals that triggered it.

See `internal/classify/` for implementation.

### Cache-Aside Pattern

ThreadMine implements efficient caching:

- **Check cache first** - Reads from `~/.threadmine/raw/` before API calls
- **Fetch on miss** - Only calls API when data not cached or stale
- **Automatic storage** - Caches API responses transparently
- **Performance** - 7-8x faster on cache hits (~160μs vs ~1.2ms)

See `internal/slack/client.go` `GetMessages()` for implementation.

### Reply Graph

The graph package tracks message relationships:

- **Nodes**: Each message with metadata
- **Adjacency List**: Parent → children mappings
- **Thread Roots**: Top-level messages
- **Statistics**: Graph metrics and analysis

See [`internal/graph/README.md`](internal/graph/README.md) for details.

## Documentation

- [SPEC.md](docs/SPEC.md) - Complete project specification
- [internal/normalize/README.md](internal/normalize/README.md) - Normalization layer
- [internal/graph/README.md](internal/graph/README.md) - Reply graph implementation

## License

MIT

