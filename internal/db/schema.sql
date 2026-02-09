-- ThreadMine SQLite Schema
-- Version: 2.0 (Redesign)
-- This schema supports the fetch/select architecture with search-based operations

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- Raw Data Layer: Source-specific data as received from APIs
-- ============================================================================

-- Raw messages from all sources (Slack, GitHub, email, etc.)
CREATE TABLE IF NOT EXISTS raw_messages (
    id TEXT PRIMARY KEY,              -- Universal ID: msg_slack_*, msg_github_*, etc.
    source_type TEXT NOT NULL,        -- slack, github, email, kusto
    source_id TEXT NOT NULL,          -- Original source identifier
    workspace_id TEXT,                -- Slack workspace, GitHub org, email account
    container_id TEXT,                -- Channel ID, repo name, etc.
    raw_data TEXT NOT NULL,           -- JSON blob of original API response
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    fetch_query TEXT,                 -- The search query that retrieved this message
    UNIQUE(source_type, source_id, workspace_id)
);

CREATE INDEX idx_raw_messages_source ON raw_messages(source_type, workspace_id, container_id);
CREATE INDEX idx_raw_messages_fetched ON raw_messages(fetched_at);

-- ============================================================================
-- Normalized Data Layer: Common schema across all sources
-- ============================================================================

-- Normalized messages with common fields
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,              -- Same ID as raw_messages
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,

    -- Temporal info
    timestamp TIMESTAMP NOT NULL,

    -- Author
    author_id TEXT NOT NULL,          -- Foreign key to users.id

    -- Content
    content TEXT NOT NULL,            -- Plain text content
    content_html TEXT,                -- Rich HTML if available

    -- Thread structure
    channel_id TEXT NOT NULL,         -- Foreign key to channels.id
    thread_id TEXT,                   -- Thread root message ID
    parent_id TEXT,                   -- Direct parent message ID
    is_thread_root BOOLEAN DEFAULT 0,

    -- Metadata (JSON blobs for flexibility)
    mentions TEXT,                    -- JSON array of user IDs
    urls TEXT,                        -- JSON array of URLs
    code_blocks TEXT,                 -- JSON array of code blocks
    attachments TEXT,                 -- JSON array of attachments

    -- Provenance
    normalized_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    schema_version TEXT DEFAULT '2.0',

    FOREIGN KEY (author_id) REFERENCES users(id),
    FOREIGN KEY (channel_id) REFERENCES channels(id)
);

CREATE INDEX idx_messages_timestamp ON messages(timestamp);
CREATE INDEX idx_messages_author ON messages(author_id);
CREATE INDEX idx_messages_channel ON messages(channel_id);
CREATE INDEX idx_messages_thread ON messages(thread_id);
CREATE INDEX idx_messages_source ON messages(source_type);
-- CREATE INDEX idx_messages_content ON messages(content); -- For full-text search

-- Full-text search on message content (FTS5 - disabled until build is configured with FTS5 support)
-- Uncomment when building with: go build -tags "fts5"
/*
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    id UNINDEXED,
    content,
    content=messages,
    content_rowid=rowid
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER messages_fts_insert AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, id, content) VALUES (new.rowid, new.id, new.content);
END;

CREATE TRIGGER messages_fts_delete AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, id, content) VALUES('delete', old.rowid, old.id, old.content);
END;

CREATE TRIGGER messages_fts_update AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, id, content) VALUES('delete', old.rowid, old.id, old.content);
    INSERT INTO messages_fts(rowid, id, content) VALUES (new.rowid, new.id, new.content);
END;
*/

-- ============================================================================
-- Users and Identity Resolution
-- ============================================================================

-- Users from all sources
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,              -- Universal ID: user_slack_*, user_github_*, etc.
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,

    -- Profile info
    display_name TEXT,
    real_name TEXT,
    email TEXT,
    avatar_url TEXT,

    -- Identity resolution
    canonical_id TEXT,                -- Links to identities.canonical_id

    -- Provenance
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(source_type, source_id)
);

CREATE INDEX idx_users_canonical ON users(canonical_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_source ON users(source_type, source_id);

-- Canonical identities (merged across sources)
CREATE TABLE IF NOT EXISTS identities (
    canonical_id TEXT PRIMARY KEY,    -- identity_*
    canonical_name TEXT,
    primary_email TEXT,
    confidence REAL DEFAULT 0.0,      -- 0.0 - 1.0
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_identities_email ON identities(primary_email);

-- ============================================================================
-- Channels and Workspaces
-- ============================================================================

-- Channels/repos/containers
CREATE TABLE IF NOT EXISTS channels (
    id TEXT PRIMARY KEY,              -- chan_slack_*, repo_github_*, etc.
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    workspace_id TEXT,

    -- Channel info
    name TEXT NOT NULL,
    display_name TEXT,
    type TEXT,                        -- channel, dm, issue, pr, discussion
    is_private BOOLEAN DEFAULT 0,
    parent_space TEXT,                -- Workspace/org context

    -- Metadata
    metadata TEXT,                    -- JSON blob for source-specific data

    -- Provenance
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(source_type, source_id, workspace_id)
);

CREATE INDEX idx_channels_workspace ON channels(workspace_id);
CREATE INDEX idx_channels_source ON channels(source_type);

-- Workspace/organization metadata cache
CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,              -- ws_slack_T123, org_github_myorg
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,          -- Original ID from source

    -- Workspace info
    name TEXT NOT NULL,
    domain TEXT,

    -- Auth context
    authenticated_user_id TEXT,       -- The "me" for this workspace

    -- Metadata
    metadata TEXT,                    -- JSON blob

    -- TTL management
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,

    UNIQUE(source_type, source_id)
);

CREATE INDEX idx_workspaces_expires ON workspaces(expires_at);

-- ============================================================================
-- Thread Analysis
-- ============================================================================

-- Thread metadata and analysis
CREATE TABLE IF NOT EXISTS threads (
    id TEXT PRIMARY KEY,              -- thread_*
    root_message_id TEXT NOT NULL,   -- Foreign key to messages.id
    channel_id TEXT NOT NULL,

    -- Structure
    message_count INTEGER DEFAULT 0,
    participant_count INTEGER DEFAULT 0,
    max_depth INTEGER DEFAULT 0,

    -- Temporal
    started_at TIMESTAMP NOT NULL,
    last_activity_at TIMESTAMP NOT NULL,

    -- Analysis flags
    has_question BOOLEAN DEFAULT 0,
    has_answer BOOLEAN DEFAULT 0,
    is_resolved BOOLEAN DEFAULT 0,

    -- Metadata
    participants TEXT,                -- JSON array of user IDs

    -- Provenance
    analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (root_message_id) REFERENCES messages(id),
    FOREIGN KEY (channel_id) REFERENCES channels(id)
);

CREATE INDEX idx_threads_channel ON threads(channel_id);
CREATE INDEX idx_threads_resolved ON threads(is_resolved);
CREATE INDEX idx_threads_activity ON threads(last_activity_at);

-- ============================================================================
-- Enrichment Layer: Basic message metadata
-- ============================================================================

-- Message enrichments (basic content features)
CREATE TABLE IF NOT EXISTS enrichments (
    message_id TEXT PRIMARY KEY,

    -- Question detection
    is_question BOOLEAN DEFAULT 0,

    -- Content metrics
    char_count INTEGER NOT NULL,
    word_count INTEGER NOT NULL,

    -- Content features
    has_code BOOLEAN DEFAULT 0,
    has_links BOOLEAN DEFAULT 0,
    has_quotes BOOLEAN DEFAULT 0,

    -- Provenance
    enriched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

CREATE INDEX idx_enrichments_is_question ON enrichments(is_question);
CREATE INDEX idx_enrichments_has_code ON enrichments(has_code);

-- Extracted entities (mentions, URLs, technical terms)
CREATE TABLE IF NOT EXISTS entities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    type TEXT NOT NULL,               -- user_mention, url, code_reference, technical_term
    value TEXT NOT NULL,
    start_pos INTEGER,
    end_pos INTEGER,
    metadata TEXT,                    -- JSON blob for additional data

    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

CREATE INDEX idx_entities_message ON entities(message_id);
CREATE INDEX idx_entities_type ON entities(type);

-- Message relationships (answers, solutions)
CREATE TABLE IF NOT EXISTS message_relations (
    from_message_id TEXT NOT NULL,
    to_message_id TEXT NOT NULL,
    relation_type TEXT NOT NULL,      -- answers_to, solution_for, acknowledges
    confidence REAL DEFAULT 1.0,

    PRIMARY KEY (from_message_id, to_message_id, relation_type),
    FOREIGN KEY (from_message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (to_message_id) REFERENCES messages(id) ON DELETE CASCADE
);

CREATE INDEX idx_relations_from ON message_relations(from_message_id);
CREATE INDEX idx_relations_to ON message_relations(to_message_id);
CREATE INDEX idx_relations_type ON message_relations(relation_type);

-- ============================================================================
-- Metadata Cache: User IDs, team details, relationships with TTL
-- ============================================================================

-- Generic metadata cache with TTL
CREATE TABLE IF NOT EXISTS metadata_cache (
    cache_key TEXT PRIMARY KEY,       -- e.g., "slack_user_U123", "github_org_details"
    source_type TEXT NOT NULL,
    cache_type TEXT NOT NULL,         -- user_profile, channel_info, team_info
    value TEXT NOT NULL,              -- JSON blob

    -- TTL management
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,

    -- Validation
    validated_at TIMESTAMP,
    is_valid BOOLEAN DEFAULT 1
);

CREATE INDEX idx_metadata_expires ON metadata_cache(expires_at);
CREATE INDEX idx_metadata_type ON metadata_cache(source_type, cache_type);

-- User relationships (who interacts with whom)
CREATE TABLE IF NOT EXISTS user_interactions (
    from_user_id TEXT NOT NULL,
    to_user_id TEXT NOT NULL,
    interaction_count INTEGER DEFAULT 1,
    last_interaction TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (from_user_id, to_user_id),
    FOREIGN KEY (from_user_id) REFERENCES users(id),
    FOREIGN KEY (to_user_id) REFERENCES users(id)
);

CREATE INDEX idx_interactions_from ON user_interactions(from_user_id);
CREATE INDEX idx_interactions_to ON user_interactions(to_user_id);

-- ============================================================================
-- Rate Limiting: Track API calls to stay within limits
-- ============================================================================

CREATE TABLE IF NOT EXISTS rate_limits (
    source_type TEXT NOT NULL,        -- slack, github
    workspace_id TEXT,                -- For per-workspace limits
    endpoint TEXT NOT NULL,           -- API endpoint or category

    -- Limit tracking
    requests_made INTEGER DEFAULT 0,
    window_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    window_duration_seconds INTEGER,  -- e.g., 60 for per-minute limits
    max_requests INTEGER,             -- e.g., 20 for Slack tier 2

    -- Self-imposed safety limit (1/2 or 1/3 of max)
    safety_limit INTEGER,

    PRIMARY KEY (source_type, workspace_id, endpoint)
);

CREATE INDEX idx_rate_limits_window ON rate_limits(window_start);

-- Insert initial schema version
INSERT INTO schema_version (version) VALUES (2);
