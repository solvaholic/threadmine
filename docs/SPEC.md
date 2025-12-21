# ThreadMine Specification

## Project Overview

**ThreadMine** is a command-line tool for extracting, caching, analyzing, and understanding conversations across multiple communication platforms. It provides a unified interface for working with messages from Slack, GitHub, and email, normalizing heterogeneous data sources into a common schema for cross-platform analysis and graph-based insights.

The tool uses locally stored browser credentials to access Slack (no app installation required), standard APIs for GitHub, and standard protocols for email (IMAP/local mbox files), making it accessible to users without administrative privileges in their workspaces.

**Repository**: `threadmine`  
**Command**: `mine`  
**Language**: Go  
**License**: MIT

## Goals and Objectives

### Primary Goals

1. **Enable Multi-Source Conversation Analysis**: Provide a unified view of conversations happening across Slack, GitHub issues/PRs, and email threads
2. **Minimize API Abuse**: Implement intelligent caching to reduce API calls and respect rate limits
3. **No Special Permissions Required**: Work with standard user-level access using local credentials
4. **Human-Readable Storage**: Use filesystem-based storage with JSON/JSONL formats for transparency and debuggability
5. **Graph-Based Insights**: Build conversation graphs to understand relationships, threads, and interaction patterns
6. **Semantic Understanding**: Classify messages (questions, answers, solutions) and assess quality/effectiveness

### Secondary Goals

- Support offline analysis of cached data
- Enable identity resolution across platforms
- Provide extensible architecture for adding new sources
- Maintain data provenance and source fidelity
- Support both interactive CLI usage and programmatic access

## Core Concepts

### 1. Multi-Source Data Integration

ThreadMine treats conversations from different platforms as variations of the same underlying structure:
- **Messages**: Individual units of communication
- **Threads**: Hierarchical discussions (Slack threads, GitHub issue comments, email reply chains)
- **Channels**: Conversation spaces (Slack channels, GitHub issues, email threads)
- **Users**: People who participate across platforms

### 2. Three-Layer Architecture

```
Raw Layer (source-specific formats)
    ↓
Normalized Layer (common schema)
    ↓
Analysis Layer (annotations, graph, insights)
```

### 3. Cache-Aside Pattern

All read operations check the cache first. On cache miss or expiration, data is fetched from the API and stored. This minimizes API calls while keeping data reasonably fresh.

### 4. Graph-Based Analysis

Messages are vertices in a graph. Edges represent relationships:
- Reply relationships (parent-child)
- Thread membership
- User interactions
- Cross-references (e.g., Slack message mentioning GitHub issue)

### 5. Semantic Annotations

Messages are enriched with computed metadata:
- Classifications (question, answer, solution, acknowledgment)
- Entity extraction (users, URLs, code blocks)
- Quality signals (solution acceptance, resolution status)
- Sentiment analysis

## Functional Requirements

### FR1: Data Extraction

#### FR1.1: Slack Integration
- **FR1.1.1**: Extract Slack cookies from system keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- **FR1.1.2**: Exchange cookies for API tokens
- **FR1.1.3**: Fetch channel lists and metadata
- **FR1.1.4**: Fetch messages from channels with date ranges
- **FR1.1.5**: Fetch thread replies for complete conversation context
- **FR1.1.6**: Fetch user profiles
- **FR1.1.7**: Support multiple workspaces

#### FR1.2: GitHub Integration
- **FR1.2.1**: Authenticate using GitHub CLI (`gh`) or personal access tokens
- **FR1.2.2**: Fetch repository metadata
- **FR1.2.3**: Fetch issues with all comments
- **FR1.2.4**: Fetch pull requests with comments and reviews
- **FR1.2.5**: Fetch commit messages and references
- **FR1.2.6**: Fetch user profiles
- **FR1.2.7**: Support multiple repositories

#### FR1.3: Kusto/Azure Data Explorer Integration
- **FR1.3.1**: Authenticate using Azure CLI (`az`) for access tokens
- **FR1.3.2**: Connect to Azure Data Explorer clusters
- **FR1.3.3**: Query conversation data using KQL (Kusto Query Language)
- **FR1.3.4**: Map query results to normalized format (e.g., support tickets, logs)
- **FR1.3.5**: Support custom KQL queries for data extraction
- **FR1.3.6**: Handle result pagination and large datasets
- **FR1.3.7**: Support multiple clusters and databases

#### FR1.4: Email Integration
- **FR1.4.1**: Connect via IMAP to email accounts
- **FR1.4.2**: Parse local mbox/maildir formats
- **FR1.4.3**: Extract email threads using References/In-Reply-To headers
- **FR1.4.4**: Parse MIME multipart messages
- **FR1.4.5**: Extract attachments metadata
- **FR1.4.6**: Build contact list from sender/recipient data
- **FR1.4.7**: Support multiple email accounts

### FR2: Data Storage

#### FR2.1: Raw Data Storage
- **FR2.1.1**: Store source-specific responses in original format
- **FR2.1.2**: Organize by source type and workspace/repo/account
- **FR2.1.3**: Use JSON for structured data
- **FR2.1.4**: Preserve API response metadata (fetch time, pagination cursors)
- **FR2.1.5**: Support incremental updates (append new messages without re-fetching)

#### FR2.2: Normalized Data Storage
- **FR2.2.1**: Convert source-specific formats to common schema
- **FR2.2.2**: Assign universal identifiers (msg_slack_*, user_github_*, etc.)
- **FR2.2.3**: Store normalized messages in JSON Lines format for efficient appends
- **FR2.2.4**: Maintain indexes by date, user, channel, and source
- **FR2.2.5**: Store identity mapping for cross-source user resolution

#### FR2.3: Graph Storage
- **FR2.3.1**: Store adjacency lists for message reply relationships
- **FR2.3.2**: Store thread hierarchies
- **FR2.3.3**: Store user interaction matrices
- **FR2.3.4**: Maintain indexes for efficient graph traversal

#### FR2.4: Annotation Storage
- **FR2.4.1**: Store message-level annotations separately from source data
- **FR2.4.2**: Store thread-level analysis results
- **FR2.4.3**: Support versioning of annotation schemas
- **FR2.4.4**: Enable incremental re-analysis

### FR3: Cache Management

- **FR3.1**: Implement TTL (time-to-live) policies per data type
  - Messages: no expiry (immutable history)
  - User profiles: 24-48 hours
  - Channel metadata: 24 hours
- **FR3.2**: Provide cache inspection commands (size, record counts, oldest/newest)
- **FR3.3**: Provide cache cleaning commands (remove expired, remove by source/date)
- **FR3.4**: Provide cache refresh commands (force update specific records)
- **FR3.5**: Validate cache integrity (checksums, JSON parseability)
- **FR3.6**: Handle partial/interrupted fetches gracefully

### FR4: Data Normalization

- **FR4.1**: Map Slack messages to normalized schema
- **FR4.2**: Map GitHub issues/PRs/comments to normalized schema
- **FR4.3**: Map Kusto query results to normalized schema
- **FR4.4**: Map email messages to normalized schema
- **FR4.5**: Extract common fields (timestamp, author, content, thread structure)
- **FR4.6**: Preserve source-specific metadata in separate field
- **FR4.7**: Handle format differences (Markdown, HTML, plain text)

### FR5: Identity Resolution

- **FR5.1**: Link user identities across sources using email addresses
- **FR5.2**: Assign canonical IDs to unified identities
- **FR5.3**: Handle ambiguous cases (multiple people with same name)
- **FR5.4**: Track confidence scores for identity links
- **FR5.5**: Support manual identity mapping overrides

### FR6: Graph Construction

- **FR6.1**: Build message reply graph from normalized data
- **FR6.2**: Build thread hierarchies
- **FR6.3**: Build user interaction graph (who replies to whom)
- **FR6.4**: Detect conversation clusters (connected components)
- **FR6.5**: Support cross-source edges (Slack message referencing GitHub issue)

### FR7: Semantic Analysis

#### FR7.1: Message Classification
- **FR7.1.1**: Detect questions using heuristics (question marks, question words, help phrases)
- **FR7.1.2**: Detect answers using positional and content signals
- **FR7.1.3**: Detect proposed solutions (code blocks, "try this", documentation links)
- **FR7.1.4**: Detect acknowledgments/acceptances ("thanks", "that worked", reactions)
- **FR7.1.5**: Assign confidence scores to classifications

#### FR7.2: Entity Extraction
- **FR7.2.1**: Extract user mentions and named entities (people, organizations)
- **FR7.2.2**: Extract URLs and categorize (documentation, internal links, external)
- **FR7.2.3**: Extract code blocks
- **FR7.2.4**: Extract technical terms/keywords

#### FR7.3: Solution Quality Assessment
- **FR7.3.1**: Track whether proposed solutions received positive responses
- **FR7.3.2**: Identify solution acceptance signals in subsequent messages
- **FR7.3.3**: Calculate time-to-resolution for threads
- **FR7.3.4**: Identify best responders (high acceptance rate)

#### FR7.4: Thread Analysis
- **FR7.4.1**: Determine if thread contains a question
- **FR7.4.2**: Determine if thread contains answers
- **FR7.4.3**: Determine if thread problem was resolved
- **FR7.4.4**: Identify key participants and their roles

### FR8: Query and Analysis Commands

#### FR8.1: Message Retrieval
- **FR8.1.1**: Get messages by user since date
- **FR8.1.2**: Get messages in channel/issue since date
- **FR8.1.3**: Search messages by text query
- **FR8.1.4**: Search with context (include surrounding messages or full threads)

#### FR8.2: User Queries
- **FR8.2.1**: Look up user by ID (resolve to name/email)
- **FR8.2.2**: Get user activity across all sources
- **FR8.2.3**: Find user interaction patterns
- **FR8.2.4**: Identify best responders

#### FR8.3: Thread Analysis Queries
- **FR8.3.1**: Get complete thread by root message ID
- **FR8.3.2**: Find threads containing search terms
- **FR8.3.3**: Find unanswered questions
- **FR8.3.4**: Find resolved vs. unresolved threads
- **FR8.3.5**: Find threads by participants

#### FR8.4: Cross-Source Queries
- **FR8.4.1**: Find Slack discussions about GitHub issues
- **FR8.4.2**: Find email threads that continued in Slack
- **FR8.4.3**: Track conversations across platforms
- **FR8.4.4**: Find all discussions about a topic across sources

#### FR8.5: Graph Analysis
- **FR8.5.1**: Find conversation paths between users
- **FR8.5.2**: Identify message/thread importance (PageRank-style)
- **FR8.5.3**: Detect user communities
- **FR8.5.4**: Find message clusters

### FR9: Output Formatting

- **FR9.1**: All commands output valid JSON to stdout
- **FR9.2**: Error messages and logs go to stderr
- **FR9.3**: Support pretty-printed JSON for human readability
- **FR9.4**: Support JSON Lines for streaming/piping
- **FR9.5**: Support format flag (--format json|jsonl|table)

### FR10: Configuration

- **FR10.1**: Support configuration file for sources, credentials, paths
- **FR10.2**: Support environment variables for sensitive data
- **FR10.3**: Allow configuration of cache location
- **FR10.4**: Allow configuration of TTL policies per data type
- **FR10.5**: Support per-source configuration (which workspaces, repos, accounts)

## Non-Functional Requirements

### NFR1: Performance
- **NFR1.1**: Cache lookups must be < 100ms for single messages
- **NFR1.2**: Graph traversals (thread retrieval) must be < 500ms for threads with < 100 messages
- **NFR1.3**: Incremental updates must not re-fetch already cached data
- **NFR1.4**: Support lazy loading (fetch on demand, not all at once)

### NFR2: Reliability
- **NFR2.1**: Handle network failures gracefully with retry logic
- **NFR2.2**: Handle API rate limits with exponential backoff
- **NFR2.3**: Use atomic file writes (write to temp, then rename)
- **NFR2.4**: Validate JSON integrity on read
- **NFR2.5**: Log all errors with context for debugging

### NFR3: Security
- **NFR3.1**: Never store credentials in cache files
- **NFR3.2**: Use restrictive file permissions (600/700) for cache directories
- **NFR3.3**: Encrypt tokens if cached (optional, configurable)
- **NFR3.4**: Provide option to exclude sensitive channels/repos
- **NFR3.5**: Clear documentation on what data is stored where

### NFR4: Maintainability
- **NFR4.1**: Use standard Go project layout
- **NFR4.2**: Comprehensive error messages with actionable guidance
- **NFR4.3**: Version the normalized schema for forward compatibility
- **NFR4.4**: Provide migration scripts for schema changes
- **NFR4.5**: Extensive logging with configurable levels

### NFR5: Usability
- **NFR5.1**: Clear, consistent command-line interface
- **NFR5.2**: Helpful error messages (not just stack traces)
- **NFR5.3**: Progress indicators for long-running operations
- **NFR5.4**: Tab completion support (bash, zsh)
- **NFR5.5**: Comprehensive help text and examples

### NFR6: Portability
- **NFR6.1**: Support macOS, Linux, and Windows
- **NFR6.2**: Use cross-platform libraries for keychain access
- **NFR6.3**: Handle filesystem path differences
- **NFR6.4**: Single binary distribution (no runtime dependencies)

## Use Cases

### UC1: Find All My Questions Since Last Week
**Actor**: Software Engineer  
**Goal**: Review questions I've asked across all platforms

**Flow**:
1. User runs: `mine messages --author me --since 2025-12-13 --type question`
2. System checks cache for messages from user since date
3. If cache miss, system fetches from APIs
4. System normalizes messages
5. System applies question classification
6. System outputs JSON array of messages

**Output**:
```json
[
  {
    "id": "msg_slack_C123_1234567890.123456",
    "source_type": "slack",
    "timestamp": "2025-12-15T10:30:00Z",
    "channel": "general",
    "content": "How do I configure rate limiting?",
    "classification": {"type": "question", "confidence": 0.9}
  },
  {
    "id": "msg_github_issue_456",
    "source_type": "github",
    "timestamp": "2025-12-16T14:20:00Z",
    "channel": "org/repo#456",
    "content": "What's the recommended way to handle errors?",
    "classification": {"type": "question", "confidence": 0.85}
  }
]
```

### UC2: Find Unanswered Questions in a Channel
**Actor**: Team Lead  
**Goal**: Identify questions that need responses

**Flow**:
1. User runs: `mine analyze --channel general --unanswered`
2. System retrieves all messages in channel from cache
3. System analyzes threads for question/answer patterns
4. System identifies threads with questions but no answers
5. System outputs results with thread context

**Output**:
```json
[
  {
    "thread_id": "thread_slack_1234567890.123456",
    "root_message": {
      "id": "msg_slack_C123_1234567890.123456",
      "content": "Anyone know why the build is failing?",
      "timestamp": "2025-12-20T09:15:00Z",
      "author": "user_slack_U789"
    },
    "reply_count": 0,
    "age_hours": 36
  }
]
```

### UC3: Find Accepted Solutions for "kubernetes"
**Actor**: Developer  
**Goal**: Learn from past solutions to Kubernetes problems

**Flow**:
1. User runs: `mine search "kubernetes" --with-solutions --accepted-only`
2. System searches normalized messages for keyword
3. System identifies threads containing search term
4. System filters for threads marked as "resolved" with accepted solutions
5. System outputs threads with solution details

**Output**:
```json
[
  {
    "thread_id": "thread_slack_1234567890.123456",
    "question": {
      "content": "Getting OOMKilled in kubernetes pods",
      "author": "user_slack_U123",
      "timestamp": "2025-11-15T10:00:00Z"
    },
    "solution": {
      "content": "Add memory limits to your deployment:\n```yaml\nresources:\n  limits:\n    memory: 2Gi\n```",
      "author": "user_slack_U456",
      "timestamp": "2025-11-15T10:15:00Z",
      "acceptance_signals": ["that worked!", "👍"]
    },
    "time_to_resolution": "15m"
  }
]
```

### UC4: Track GitHub Issue Discussion Across Platforms
**Actor**: Product Manager  
**Goal**: See full conversation about a feature request across GitHub and Slack

**Flow**:
1. User runs: `mine cross-source --github-issue org/repo#123`
2. System fetches GitHub issue and all comments
3. System searches Slack for mentions of issue URL or number
4. System retrieves Slack threads containing mentions
5. System combines timeline of discussions

**Output**:
```json
{
  "github_issue": {
    "number": 123,
    "title": "Add dark mode support",
    "created": "2025-12-01T00:00:00Z",
    "comments": 15
  },
  "slack_discussions": [
    {
      "channel": "product",
      "thread_root": "msg_slack_C456_1234567890.123456",
      "timestamp": "2025-12-02T10:00:00Z",
      "participant_count": 5,
      "message_count": 12,
      "summary": "Team discussed implementation approach"
    }
  ],
  "timeline": [
    {"source": "github", "timestamp": "2025-12-01T00:00:00Z", "event": "issue_opened"},
    {"source": "slack", "timestamp": "2025-12-02T10:00:00Z", "event": "discussion_started"},
    {"source": "github", "timestamp": "2025-12-03T15:00:00Z", "event": "comment_added"}
  ]
}
```

### UC5: Identify Best Responders
**Actor**: Team Lead  
**Goal**: Recognize team members who effectively help others

**Flow**:
1. User runs: `mine analyze --best-responders --timeframe 30d`
2. System retrieves all threads from last 30 days
3. System identifies messages classified as answers/solutions
4. System tracks solution acceptance rates per user
5. System outputs leaderboard

**Output**:
```json
[
  {
    "user": {
      "canonical_id": "identity_abc123",
      "name": "Jane Doe",
      "sources": ["slack:U123", "github:janedoe"]
    },
    "stats": {
      "total_answers": 45,
      "accepted_solutions": 38,
      "acceptance_rate": 0.84,
      "avg_response_time": "15m",
      "sources": ["slack", "github"]
    }
  }
]
```

### UC6: Find Cross-Source User Activity
**Actor**: Engineering Manager  
**Goal**: Understand how a team member communicates across platforms

**Flow**:
1. User runs: `mine user --email john@example.com --activity`
2. System looks up user in identity map
3. System retrieves all messages from user across all sources
4. System analyzes participation patterns
5. System outputs activity summary

**Output**:
```json
{
  "user": {
    "canonical_id": "identity_xyz789",
    "name": "John Smith",
    "email": "john@example.com"
  },
  "activity": {
    "slack": {
      "message_count": 342,
      "channels": 12,
      "threads_started": 23,
      "replies": 319,
      "period": "2025-01-01 to 2025-12-21"
    },
    "github": {
      "issues_opened": 8,
      "comments": 67,
      "pull_requests": 15,
      "reviews": 42,
      "period": "2025-01-01 to 2025-12-21"
    },
    "email": {
      "sent": 156,
      "received": 423,
      "threads": 78,
      "period": "2025-01-01 to 2025-12-21"
    }
  }
}
```

### UC7: Cache Management
**Actor**: Power User  
**Goal**: Understand and manage local cache storage

**Flow**:
1. User runs: `mine cache info`
2. System scans cache directories
3. System calculates storage metrics
4. System outputs summary

**Output**:
```json
{
  "cache_location": "/Users/user/.threadmine",
  "total_size": "245 MB",
  "by_source": {
    "slack": {"raw": "120 MB", "normalized": "40 MB", "message_count": 12500},
    "github": {"raw": "60 MB", "normalized": "20 MB", "message_count": 3200},
    "email": {"raw": "30 MB", "normalized": "15 MB", "message_count": 2100}
  },
  "date_range": {
    "earliest": "2024-01-01T00:00:00Z",
    "latest": "2025-12-21T10:30:00Z"
  }
}
```

## Data Architecture

### Directory Structure

```
~/.threadmine/
├── config/
│   ├── sources.json              # Enabled sources, credentials location
│   ├── schema_version.json       # Track normalization schema version
│   └── retention.json            # Per-source TTL policies
│
├── raw/                          # Source-specific formats
│   ├── slack/
│   │   ├── workspaces/
│   │   │   └── T12345/           # Workspace ID
│   │   │       ├── metadata.json
│   │   │       ├── users/
│   │   │       │   ├── U12345.json
│   │   │       │   └── _index.json
│   │   │       ├── channels/
│   │   │       │   ├── C12345/
│   │   │       │   │   ├── info.json
│   │   │       │   │   └── messages/
│   │   │       │   │       ├── 2025-12-01.json
│   │   │       │   │       └── 2025-12-02.json
│   │   │       │   └── _index.json
│   │   │       └── threads/
│   │   │           └── 1234567890.123456.json
│   │   └── auth/
│   │       └── tokens.json       # Encrypted tokens (if cached)
│   │
│   ├── github/
│   │   ├── repos/
│   │   │   └── owner-repo/       # org-name/repo-name
│   │   │       ├── metadata.json
│   │   │       ├── issues/
│   │   │       │   ├── 123.json
│   │   │       │   └── _index.json
│   │   │       ├── pull_requests/
│   │   │       │   ├── 456.json
│   │   │       │   └── _index.json
│   │   │       ├── comments/
│   │   │       │   ├── issue-123/
│   │   │       │   │   └── comments.json
│   │   │       │   └── pr-456/
│   │   │       │       ├── comments.json
│   │   │       │       └── reviews.json
│   │   │       └── commits/
│   │   │           └── 2025-12.json
│   │   └── users/
│   │       ├── username.json
│   │       └── _index.json
│   │
│   └── email/
│       ├── accounts/
│       │   └── user@example.com/
│       │       ├── metadata.json
│       │       ├── folders/
│       │       │   ├── INBOX/
│       │       │   │   ├── 2025-12-01.mbox
│       │       │   │   └── index.json
│       │       │   └── Sent/
│       │       │       ├── 2025-12-01.mbox
│       │       │       └── index.json
│       │       └── threads/
│       │           └── thread-abc123.json
│       └── contacts/
│           └── contacts.json
│
├── normalized/                   # Common schema across sources
│   ├── messages/
│   │   ├── by_id/
│   │   │   └── msg_<source>_<id>.json
│   │   ├── by_date/
│   │   │   └── 2025-12/
│   │   │       ├── 2025-12-01.jsonl
│   │   │       └── 2025-12-02.jsonl
│   │   └── by_source/
│   │       ├── slack.jsonl
│   │       ├── github.jsonl
│   │       └── email.jsonl
│   │
│   ├── users/
│   │   ├── by_id/
│   │   │   └── user_<source>_<id>.json
│   │   └── identity_map.json
│   │
│   └── channels/
│       ├── by_id/
│       │   └── chan_<source>_<id>.json
│       └── _index.json
│
├── graph/
│   ├── structure/
│   │   ├── adjacency.json
│   │   ├── threads.json
│   │   └── user_interactions.json
│   │
│   ├── indexes/
│   │   ├── by_user.json
│   │   ├── by_channel.json
│   │   └── by_thread.json
│   │
│   └── metadata.json
│
├── annotations/
│   ├── messages/
│   │   └── msg_<source>_<id>/
│   │       ├── classifications.json
│   │       ├── entities.json
│   │       └── sentiment.json
│   │
│   ├── threads/
│   │   └── thread_<id>/
│   │       ├── resolution.json
│   │       ├── participants.json
│   │       └── timeline.json
│   │
│   └── cross_source/
│       └── linked_discussions/
│           └── topic-xyz.json
│
├── processed/
│   ├── reports/
│   │   ├── daily_summaries/
│   │   └── user_activity/
│   │
│   ├── exports/
│   │   ├── markdown/
│   │   └── json/
│   │
│   └── cache/
│       └── query_results/
│
└── logs/
    ├── fetch.log
    ├── normalization.log
    └── analysis.log
```

## Normalized Message Schema

```go
type NormalizedMessage struct {
    // Universal identifiers
    ID            string    `json:"id"`              // msg_slack_1234567890.123456
    SourceType    string    `json:"source_type"`     // "slack", "github", "email"
    SourceID      string    `json:"source_id"`       // Original source identifier
    
    // Common fields
    Timestamp     time.Time `json:"timestamp"`
    Author        *User     `json:"author"`
    Content       string    `json:"content"`         // Normalized text
    ContentHTML   string    `json:"content_html"`    // Rich format if available
    
    // Conversation context
    Channel       *Channel  `json:"channel"`
    ThreadID      string    `json:"thread_id"`
    ParentID      string    `json:"parent_id"`
    IsThreadRoot  bool      `json:"is_thread_root"`
    
    // Metadata
    Attachments   []Attachment `json:"attachments"`
    Mentions      []string     `json:"mentions"`
    URLs          []string     `json:"urls"`
    CodeBlocks    []CodeBlock  `json:"code_blocks"`
    
    // Source-specific (preserved as-is)
    SourceMetadata map[string]interface{} `json:"source_metadata"`
    
    // Provenance
    FetchedAt     time.Time `json:"fetched_at"`
    NormalizedAt  time.Time `json:"normalized_at"`
    SchemaVersion string    `json:"schema_version"`
}

type User struct {
    ID            string `json:"id"`
    SourceType    string `json:"source_type"`
    SourceID      string `json:"source_id"`
    DisplayName   string `json:"display_name"`
    RealName      string `json:"real_name"`
    Email         string `json:"email"`
    AvatarURL     string `json:"avatar_url"`
    CanonicalID   string `json:"canonical_id"`
    AlternateIDs  []string `json:"alternate_ids"`
}

type Channel struct {
    ID            string `json:"id"`
    SourceType    string `json:"source_type"`
    SourceID      string `json:"source_id"`
    Name          string `json:"name"`
    DisplayName   string `json:"display_name"`
    Type          string `json:"type"`
    IsPrivate     bool   `json:"is_private"`
    ParentSpace   string `json:"parent_space"`
}

type Attachment struct {
    Type          string `json:"type"`
    URL           string `json:"url"`
    Title         string `json:"title"`
    MimeType      string `json:"mime_type"`
}

type CodeBlock struct {
    Language      string `json:"language"`
    Code          string `json:"code"`
}
```

## Annotation Schema

```go
type AnnotatedMessage struct {
    Message         *NormalizedMessage `json:"message"`
    ThreadPosition  int                `json:"thread_position"`
    ReplyDepth      int                `json:"reply_depth"`
    ThreadRoot      string             `json:"thread_root"`
    Classifications []Classification   `json:"classifications"`
    Entities        []Entity           `json:"entities"`
    Sentiment       *Sentiment         `json:"sentiment,omitempty"`
    AnswersTo       []string           `json:"answers_to,omitempty"`
    AsksAbout       []string           `json:"asks_about,omitempty"`
    ProposedSolution *Solution         `json:"proposed_solution,omitempty"`
}

type Classification struct {
    Type       string   `json:"type"`
    Confidence float64  `json:"confidence"`
    Signals    []string `json:"signals"`
}

type Entity struct {
    Type       string `json:"type"`
    Value      string `json:"value"`
    Start      int    `json:"start"`
    End        int    `json:"end"`
}

type Sentiment struct {
    Score      float64 `json:"score"`
    Label      string  `json:"label"`
}

type Solution struct {
    Type         string      `json:"type"`
    Completeness float64     `json:"completeness"`
    Acceptance   *Acceptance `json:"acceptance,omitempty"`
}

type Acceptance struct {
    Accepted     bool     `json:"accepted"`
    Signals      []string `json:"signals"`
    ConfirmedBy  []string `json:"confirmed_by"`
}
```

## Identity Resolution Schema

```go
type IdentityMap struct {
    Identities map[string]*Identity `json:"identities"`
}

type Identity struct {
    CanonicalID   string                       `json:"canonical_id"`
    CanonicalName string                       `json:"canonical_name"`
    PrimaryEmail  string                       `json:"primary_email"`
    Sources       map[string]SourceIdentity    `json:"sources"`
    Confidence    float64                      `json:"confidence"`
}

type SourceIdentity struct {
    SourceType string `json:"source_type"`
    SourceID   string `json:"source_id"`
    Name       string `json:"name"`
    Email      string `json:"email"`
}
```

## Command-Line Interface Structure

### Top-Level Commands

```bash
mine <command> [subcommand] [flags]
```

### Command Groups

#### Data Fetching
- `mine fetch slack [flags]` - Fetch data from Slack
- `mine fetch github [flags]` - Fetch data from GitHub
- `mine fetch email [flags]` - Fetch data from email

#### Querying
- `mine messages [flags]` - Query messages
- `mine threads [flags]` - Query threads
- `mine users [flags]` - Query users
- `mine channels [flags]` - Query channels

#### Analysis
- `mine analyze [subcommand] [flags]` - Run analysis
  - `mine analyze questions` - Find questions
  - `mine analyze solutions` - Find solutions
  - `mine analyze responders` - Find best responders
  - `mine analyze resolution` - Analyze thread resolution

#### Cross-Source
- `mine cross-source [flags]` - Cross-source analysis
- `mine identity [flags]` - Manage identity resolution

#### Cache Management
- `mine cache info` - Show cache information
- `mine cache clean [flags]` - Clean cache
- `mine cache refresh [flags]` - Refresh cache data
- `mine cache validate` - Validate cache integrity

#### Configuration
- `mine config init` - Initialize configuration
- `mine config show` - Show current configuration
- `mine config set [key] [value]` - Set configuration value

### Common Flags

- `--source` - Filter by source (slack, github, email)
- `--since` - Start date (YYYY-MM-DD or relative like "7d")
- `--until` - End date
- `--author` - Filter by author (user ID or "me")
- `--channel` - Filter by channel/issue/thread
- `--format` - Output format (json, jsonl, table)
- `--output` - Output file (default: stdout)
- `--verbose` - Verbose logging
- `--config` - Config file path (default: ~/.threadmine/config.yaml)

## Success Criteria

The ThreadMine project will be considered successful when it meets the following criteria:

### Must Have (MVP)
1. ✅ Successfully authenticate with Slack using browser cookies
2. ✅ Fetch and cache messages from at least one Slack channel
3. ✅ Normalize Slack messages to common schema
4. ✅ Build basic message reply graph
5. ✅ Implement cache-aside pattern for message retrieval
6. ✅ Classify messages as questions or answers using heuristics
7. ✅ Output valid JSON for all commands
8. ✅ Work on macOS, Linux, and Windows

### Should Have (Full v1.0)
1. ✅ Support GitHub issues and PRs
2. ✅ Support email (IMAP and mbox)
3. ✅ Cross-source identity resolution
4. ✅ Full graph analysis (threads, user interactions, clusters)
5. ✅ Solution quality assessment
6. ✅ Cache management commands
7. ✅ Comprehensive error handling and logging
8. ✅ Configuration file support

### Could Have (Future)
1. Web UI for browsing cached data
2. Machine learning models for better classification
3. Export to other formats (Markdown, HTML)
4. Real-time monitoring mode
5. Integration with more sources (Discord, Teams, Jira)
6. Collaborative filtering recommendations
7. Automatic summarization

## Out of Scope

The following are explicitly **not** goals for this project:

- ❌ Creating or sending messages (read-only tool)
- ❌ Real-time streaming/notifications
- ❌ Slack bot/app that runs in workspace
- ❌ Cloud/SaaS version
- ❌ Multi-user/collaborative features
- ❌ Advanced NLP/ML models (initially - heuristics only)
- ❌ Mobile apps
- ❌ GUI application

## Notes for Implementation

### Authentication Approach
- For Slack: Use `github.com/rneatherway/slack` library for cookie-based auth
- For GitHub: Use GitHub CLI (`gh`) to obtain tokens or execute API calls directly
- For Kusto: Use Azure CLI (`az`) to obtain access tokens for Azure Data Explorer
- For Email: Support OAuth2 where available, app passwords otherwise

### Key Go Libraries
- **Slack API**: `github.com/rneatherway/slack`
- **GitHub API**: `github.com/google/go-github` or GitHub CLI (`gh`) invocation
- **Kusto/ADX**: `github.com/Azure/azure-kusto-go` with Azure CLI (`az`) for auth
- **Email**: `github.com/emersion/go-imap`
- **Graph**: Consider `gonum.org/v1/gonum/graph` or implement custom
- **CLI**: `github.com/spf13/cobra`
- **Config**: `github.com/spf13/viper`

### Testing Strategy
- Unit tests for normalization logic
- Integration tests with mock APIs
- Fixture-based tests with real (sanitized) data
- Network tests tagged separately (require real API access)

### Documentation Requirements
- Comprehensive README with quickstart
- Architecture documentation
- API/schema documentation
- Examples for common use cases
- Troubleshooting guide

---

**Version**: 1.0  
**Last Updated**: 2025-12-21  
**Status**: Draft - Ready for Implementation
