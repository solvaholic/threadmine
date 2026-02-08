package db

import (
	"database/sql"
	"fmt"
	"time"
)

// User represents a user in the database
type User struct {
	ID           string
	SourceType   string
	SourceID     string
	DisplayName  *string
	RealName     *string
	Email        *string
	AvatarURL    *string
	CanonicalID  *string
	FetchedAt    time.Time
	UpdatedAt    time.Time
}

// SaveUser saves or updates a user
func (db *DB) SaveUser(user *User) error {
	_, err := db.Exec(`
		INSERT INTO users (
			id, source_type, source_id, display_name, real_name, email, avatar_url, canonical_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_type, source_id) DO UPDATE SET
			display_name = excluded.display_name,
			real_name = excluded.real_name,
			email = excluded.email,
			avatar_url = excluded.avatar_url,
			canonical_id = excluded.canonical_id,
			updated_at = CURRENT_TIMESTAMP
	`, user.ID, user.SourceType, user.SourceID, user.DisplayName, user.RealName,
		user.Email, user.AvatarURL, user.CanonicalID)

	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

// GetUser retrieves a user by ID
func (db *DB) GetUser(id string) (*User, error) {
	user := &User{}

	err := db.QueryRow(`
		SELECT id, source_type, source_id, display_name, real_name, email, avatar_url,
		       canonical_id, fetched_at, updated_at
		FROM users
		WHERE id = ?
	`, id).Scan(
		&user.ID, &user.SourceType, &user.SourceID, &user.DisplayName, &user.RealName,
		&user.Email, &user.AvatarURL, &user.CanonicalID, &user.FetchedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserBySourceID retrieves a user by source type and source ID
func (db *DB) GetUserBySourceID(sourceType, sourceID string) (*User, error) {
	user := &User{}

	err := db.QueryRow(`
		SELECT id, source_type, source_id, display_name, real_name, email, avatar_url,
		       canonical_id, fetched_at, updated_at
		FROM users
		WHERE source_type = ? AND source_id = ?
	`, sourceType, sourceID).Scan(
		&user.ID, &user.SourceType, &user.SourceID, &user.DisplayName, &user.RealName,
		&user.Email, &user.AvatarURL, &user.CanonicalID, &user.FetchedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// Identity represents a canonical identity across sources
type Identity struct {
	CanonicalID   string
	CanonicalName *string
	PrimaryEmail  *string
	Confidence    float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SaveIdentity saves or updates a canonical identity
func (db *DB) SaveIdentity(identity *Identity) error {
	_, err := db.Exec(`
		INSERT INTO identities (
			canonical_id, canonical_name, primary_email, confidence
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(canonical_id) DO UPDATE SET
			canonical_name = excluded.canonical_name,
			primary_email = excluded.primary_email,
			confidence = excluded.confidence,
			updated_at = CURRENT_TIMESTAMP
	`, identity.CanonicalID, identity.CanonicalName, identity.PrimaryEmail, identity.Confidence)

	if err != nil {
		return fmt.Errorf("failed to save identity: %w", err)
	}

	return nil
}

// LinkUserToIdentity links a user to a canonical identity
func (db *DB) LinkUserToIdentity(userID, canonicalID string) error {
	_, err := db.Exec(`
		UPDATE users
		SET canonical_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, canonicalID, userID)

	if err != nil {
		return fmt.Errorf("failed to link user to identity: %w", err)
	}

	return nil
}

// GetUsersByIdentity retrieves all users linked to a canonical identity
func (db *DB) GetUsersByIdentity(canonicalID string) ([]*User, error) {
	rows, err := db.Query(`
		SELECT id, source_type, source_id, display_name, real_name, email, avatar_url,
		       canonical_id, fetched_at, updated_at
		FROM users
		WHERE canonical_id = ?
	`, canonicalID)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by identity: %w", err)
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.SourceType, &user.SourceID, &user.DisplayName, &user.RealName,
			&user.Email, &user.AvatarURL, &user.CanonicalID, &user.FetchedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}
