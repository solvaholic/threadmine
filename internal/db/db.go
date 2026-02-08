package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

const SchemaVersion = 2

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
	path string
}

// Open opens or creates the ThreadMine database at the given path
func Open(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database
	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_journal_mode=WAL&_timeout=5000", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(1) // SQLite works best with single writer
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(time.Hour)

	db := &DB{
		conn: conn,
		path: dbPath,
	}

	// Initialize schema if needed
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// initSchema initializes the database schema if not already present
func (db *DB) initSchema() error {
	// Check current schema version
	var currentVersion int
	err := db.conn.QueryRow("SELECT version FROM schema_version ORDER BY version DESC LIMIT 1").Scan(&currentVersion)

	// If table doesn't exist or is empty, create schema
	if err == sql.ErrNoRows || (err != nil && (err.Error() == "no such table: schema_version" || err.Error() == "SQL logic error: no such table: schema_version")) {
		if _, err := db.conn.Exec(schemaSQL); err != nil {
			return fmt.Errorf("failed to execute schema: %w", err)
		}
		return nil
	}

	// If other error, fail
	if err != nil {
		return fmt.Errorf("failed to check schema version: %w", err)
	}

	// Check if migration is needed
	if currentVersion < SchemaVersion {
		return fmt.Errorf("schema migration needed from version %d to %d (not implemented)", currentVersion, SchemaVersion)
	}

	return nil
}

// Begin starts a new transaction
func (db *DB) Begin() (*sql.Tx, error) {
	return db.conn.Begin()
}

// Exec executes a query without returning rows
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

// Query executes a query that returns rows
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

// QueryRow executes a query that returns at most one row
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

// DefaultDBPath returns the default database path
func DefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./threadmine.db"
	}
	return filepath.Join(home, ".threadmine", "threadmine.db")
}

// Stats returns database statistics
func (db *DB) Stats() (*Stats, error) {
	stats := &Stats{}

	// Count messages
	err := db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&stats.MessageCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count messages: %w", err)
	}

	// Count users
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.UserCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Count channels
	err = db.QueryRow("SELECT COUNT(*) FROM channels").Scan(&stats.ChannelCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count channels: %w", err)
	}

	// Count threads
	err = db.QueryRow("SELECT COUNT(*) FROM threads").Scan(&stats.ThreadCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count threads: %w", err)
	}

	// Get date range
	err = db.QueryRow(`
		SELECT MIN(timestamp), MAX(timestamp)
		FROM messages
	`).Scan(&stats.EarliestMessage, &stats.LatestMessage)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get date range: %w", err)
	}

	// Get database file size
	if info, err := os.Stat(db.path); err == nil {
		stats.DatabaseSize = info.Size()
	}

	return stats, nil
}

// Stats represents database statistics
type Stats struct {
	MessageCount     int64
	UserCount        int64
	ChannelCount     int64
	ThreadCount      int64
	EarliestMessage  *time.Time
	LatestMessage    *time.Time
	DatabaseSize     int64
}
