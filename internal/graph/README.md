# Graph Package

The `graph` package implements a message reply graph structure for ThreadMine, enabling thread analysis and traversal across all message sources.

## Overview

The reply graph tracks parent-child relationships between messages, making it easy to:
- Identify thread roots
- Traverse complete conversation threads
- Analyze thread depth and structure
- Find messages with replies
- Calculate graph statistics

## Data Structure

The graph uses three main components stored in `~/.threadmine/graph/structure/`:

### 1. Nodes (`nodes.json`)
Maps message IDs to node metadata:
```json
{
  "msg_slack_T123_C456_1234567890.123456": {
    "message_id": "msg_slack_T123_C456_1234567890.123456",
    "thread_id": "1234567890.123456",
    "parent_id": "",
    "is_thread_root": true,
    "author": "user_slack_T123_U789",
    "timestamp": "2025-12-22T10:00:00Z",
    "channel": "chan_slack_T123_C456",
    "source_type": "slack"
  }
}
```

### 2. Adjacency List (`adjacency.json`)
Maps parent message IDs to arrays of child message IDs:
```json
{
  "msg_slack_T123_C456_1234567890.123456": [
    "msg_slack_T123_C456_1234567890.123457",
    "msg_slack_T123_C456_1234567890.123458"
  ]
}
```

### 3. Thread Roots (`thread_roots.json`)
Array of message IDs that are thread roots (no parent):
```json
[
  "msg_slack_T123_C456_1234567890.123456",
  "msg_slack_T123_C456_1234567891.123456"
]
```

### 4. Metadata (`metadata.json`)
Graph statistics and metadata:
```json
{
  "updated_at": "2025-12-22T10:00:00Z",
  "stats": {
    "total_messages": 15,
    "thread_count": 14,
    "reply_messages": 1,
    "messages_with_replies": 1,
    "average_thread_depth": 0.07
  }
}
```

## Usage

### Building a Graph

```go
import (
    "github.com/solvaholic/threadmine/internal/graph"
    "github.com/solvaholic/threadmine/internal/normalize"
)

// From a slice of normalized messages
messages := []*normalize.NormalizedMessage{...}
g := graph.BuildFromNormalizedMessages(messages)

// Or incrementally
g := graph.NewReplyGraph()
g.AddMessage(message1)
g.AddMessage(message2)
```

### Querying the Graph

```go
// Get direct children of a message
children := g.GetChildren(messageID)

// Get complete thread
thread := g.GetThread(rootMessageID)

// Calculate thread depth
depth := g.GetThreadDepth(rootMessageID)

// Get statistics
stats := g.Stats()
```

### Saving and Loading

```go
// Save to disk
if err := graph.SaveReplyGraph(g); err != nil {
    log.Fatal(err)
}

// Load from disk
g, err := graph.LoadReplyGraph()
if err != nil {
    log.Fatal(err)
}
```

## Implementation Details

### Thread Detection

A message is identified as a thread root when:
- `IsThreadRoot` is `true` in the normalized message
- `ParentID` is empty

For Slack messages:
- Root messages have no `thread_ts` or `thread_ts == ts`
- Replies have `thread_ts` pointing to the root

### Graph Traversal

Thread traversal uses recursive depth-first search to collect all messages in a thread:

1. Start at root message
2. Add root to results
3. For each child, recursively collect its children
4. Return complete thread in traversal order

### Depth Calculation

Thread depth is calculated recursively:
- Root has depth 0
- Each reply level adds 1
- Maximum depth is returned for branches

## File Format

All graph files use:
- **Format**: JSON with 2-space indentation
- **Permissions**: 0600 (owner read/write only)
- **Atomic Writes**: Write to `.tmp` file, then rename
- **Human-Readable**: Formatted for inspection and debugging

## Future Enhancements

Potential additions for the graph package:

1. **User Interaction Graph**: Track who replies to whom
2. **Cross-Source Edges**: Link Slack threads mentioning GitHub issues
3. **Graph Queries**: Find conversation paths, detect clusters
4. **PageRank-Style Scoring**: Identify important messages/threads
5. **Incremental Updates**: Efficiently update graph without rebuilding
6. **Graph Export**: Export to DOT/GraphML for visualization

## Related Packages

- `internal/normalize`: Provides normalized message schema
- `internal/cache`: Raw message storage
- `cmd/mine`: CLI that uses the graph for analysis

## Testing

Run tests with:
```bash
go test ./internal/graph/...
```

Tests cover:
- Node addition and lookup
- Parent-child relationships
- Thread traversal
- Depth calculation
- Statistics computation
- Build from messages
