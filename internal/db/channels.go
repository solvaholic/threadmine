package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Channel represents a channel/repo/container in the database
type Channel struct {
	ID          string
	SourceType  string
	SourceID    string
	WorkspaceID *string
	Name        string
	DisplayName *string
	Type        *string
	IsPrivate   bool
	ParentSpace *string
	Metadata    *string
	FetchedAt   time.Time
	UpdatedAt   time.Time
}

// SaveChannel saves or updates a channel
func (db *DB) SaveChannel(channel *Channel) error {
	_, err := db.Exec(`
		INSERT INTO channels (
			id, source_type, source_id, workspace_id, name, display_name, type,
			is_private, parent_space, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_type, source_id, workspace_id) DO UPDATE SET
			name = excluded.name,
			display_name = excluded.display_name,
			type = excluded.type,
			is_private = excluded.is_private,
			parent_space = excluded.parent_space,
			metadata = excluded.metadata,
			updated_at = CURRENT_TIMESTAMP
	`, channel.ID, channel.SourceType, channel.SourceID, channel.WorkspaceID,
		channel.Name, channel.DisplayName, channel.Type, channel.IsPrivate,
		channel.ParentSpace, channel.Metadata)

	if err != nil {
		return fmt.Errorf("failed to save channel: %w", err)
	}

	return nil
}

// GetChannel retrieves a channel by ID
func (db *DB) GetChannel(id string) (*Channel, error) {
	channel := &Channel{}

	err := db.QueryRow(`
		SELECT id, source_type, source_id, workspace_id, name, display_name, type,
		       is_private, parent_space, metadata, fetched_at, updated_at
		FROM channels
		WHERE id = ?
	`, id).Scan(
		&channel.ID, &channel.SourceType, &channel.SourceID, &channel.WorkspaceID,
		&channel.Name, &channel.DisplayName, &channel.Type, &channel.IsPrivate,
		&channel.ParentSpace, &channel.Metadata, &channel.FetchedAt, &channel.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	return channel, nil
}

// Workspace represents a workspace/organization
type Workspace struct {
	ID                   string
	SourceType           string
	SourceID             string
	Name                 string
	Domain               *string
	AuthenticatedUserID  *string
	Metadata             *string
	FetchedAt            time.Time
	ExpiresAt            *time.Time
}

// SaveWorkspace saves or updates a workspace
func (db *DB) SaveWorkspace(workspace *Workspace) error {
	_, err := db.Exec(`
		INSERT INTO workspaces (
			id, source_type, source_id, name, domain, authenticated_user_id,
			metadata, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_type, source_id) DO UPDATE SET
			name = excluded.name,
			domain = excluded.domain,
			authenticated_user_id = excluded.authenticated_user_id,
			metadata = excluded.metadata,
			expires_at = excluded.expires_at,
			fetched_at = CURRENT_TIMESTAMP
	`, workspace.ID, workspace.SourceType, workspace.SourceID, workspace.Name,
		workspace.Domain, workspace.AuthenticatedUserID, workspace.Metadata, workspace.ExpiresAt)

	if err != nil {
		return fmt.Errorf("failed to save workspace: %w", err)
	}

	return nil
}

// GetWorkspace retrieves a workspace by ID
func (db *DB) GetWorkspace(id string) (*Workspace, error) {
	workspace := &Workspace{}

	err := db.QueryRow(`
		SELECT id, source_type, source_id, name, domain, authenticated_user_id,
		       metadata, fetched_at, expires_at
		FROM workspaces
		WHERE id = ?
	`, id).Scan(
		&workspace.ID, &workspace.SourceType, &workspace.SourceID, &workspace.Name,
		&workspace.Domain, &workspace.AuthenticatedUserID, &workspace.Metadata,
		&workspace.FetchedAt, &workspace.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	return workspace, nil
}

// GetWorkspacesBySource retrieves all workspaces for a source type
func (db *DB) GetWorkspacesBySource(sourceType string) ([]*Workspace, error) {
	rows, err := db.Query(`
		SELECT id, source_type, source_id, name, domain, authenticated_user_id,
		       metadata, fetched_at, expires_at
		FROM workspaces
		WHERE source_type = ?
	`, sourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to query workspaces: %w", err)
	}
	defer rows.Close()

	workspaces := []*Workspace{}
	for rows.Next() {
		ws := &Workspace{}
		err := rows.Scan(
			&ws.ID, &ws.SourceType, &ws.SourceID, &ws.Name, &ws.Domain,
			&ws.AuthenticatedUserID, &ws.Metadata, &ws.FetchedAt, &ws.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workspace: %w", err)
		}
		workspaces = append(workspaces, ws)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workspaces: %w", err)
	}

	return workspaces, nil
}

// FindChannelsByName finds channels by name, display name, or source ID
func (db *DB) FindChannelsByName(name string) ([]*Channel, error) {
	rows, err := db.Query(`
		SELECT id, source_type, source_id, workspace_id, name, display_name, type,
		       is_private, parent_space, metadata, fetched_at, updated_at
		FROM channels
		WHERE name = ? OR display_name = ? OR source_id = ?
	`, name, name, name)
	if err != nil {
		return nil, fmt.Errorf("failed to query channels by name: %w", err)
	}
	defer rows.Close()

	channels := []*Channel{}
	for rows.Next() {
		channel := &Channel{}
		err := rows.Scan(
			&channel.ID, &channel.SourceType, &channel.SourceID, &channel.WorkspaceID,
			&channel.Name, &channel.DisplayName, &channel.Type, &channel.IsPrivate,
			&channel.ParentSpace, &channel.Metadata, &channel.FetchedAt, &channel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating channels: %w", err)
	}

	return channels, nil
}
