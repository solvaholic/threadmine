# threadmine
Extract and analyze multi-platform conversations, with zero admin overhead

## Features

ThreadMine (`mine` CLI) is a Go-based tool that:

- **Extracts conversations** from Slack, GitHub, and email using local credentials
- **Caches data** efficiently to minimize API calls and respect rate limits
- **Normalizes messages** across platforms into a common schema
- **Builds reply graphs** to track conversation threads and relationships
- **Analyzes threads** to identify questions, answers, and solutions (coming soon)

## Current Status

✅ **MVP Steps Completed:**
1. ✅ Authenticate with Slack using browser cookies
2. ✅ Fetch and cache messages from Slack channels
3. ✅ Normalize Slack messages to common schema
4. ✅ Build basic message reply graph

**Next Steps:**
5. Implement cache-aside pattern for message retrieval
6. Classify messages as questions or answers using heuristics
7. Output valid JSON for all commands

## Quick Start

```bash
# Build the tool
go build -o mine ./cmd/mine

# Run to fetch Slack messages and build graph
./mine

# View graph statistics
go build -o graph-demo ./cmd/graph-demo
./graph-demo
```

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

