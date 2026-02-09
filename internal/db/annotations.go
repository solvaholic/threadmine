package db

import (
	"fmt"
	"time"
)

// Enrichment represents basic message metadata
type Enrichment struct {
	MessageID  string
	IsQuestion bool
	CharCount  int
	WordCount  int
	HasCode    bool
	HasLinks   bool
	HasQuotes  bool
	EnrichedAt time.Time
}

// SaveEnrichment saves message enrichment metadata
func (db *DB) SaveEnrichment(enrich *Enrichment) error {
	_, err := db.Exec(`
		INSERT INTO enrichments (message_id, is_question, char_count, word_count, has_code, has_links, has_quotes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(message_id) DO UPDATE SET
			is_question = excluded.is_question,
			char_count = excluded.char_count,
			word_count = excluded.word_count,
			has_code = excluded.has_code,
			has_links = excluded.has_links,
			has_quotes = excluded.has_quotes,
			enriched_at = CURRENT_TIMESTAMP
	`, enrich.MessageID, enrich.IsQuestion, enrich.CharCount, enrich.WordCount,
	   enrich.HasCode, enrich.HasLinks, enrich.HasQuotes)

	if err != nil {
		return fmt.Errorf("failed to save enrichment: %w", err)
	}

	return nil
}

// GetEnrichment retrieves enrichment metadata for a message
func (db *DB) GetEnrichment(messageID string) (*Enrichment, error) {
	enrich := &Enrichment{}

	err := db.QueryRow(`
		SELECT message_id, is_question, char_count, word_count, has_code, has_links, has_quotes, enriched_at
		FROM enrichments
		WHERE message_id = ?
	`, messageID).Scan(&enrich.MessageID, &enrich.IsQuestion, &enrich.CharCount, &enrich.WordCount,
		&enrich.HasCode, &enrich.HasLinks, &enrich.HasQuotes, &enrich.EnrichedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to query enrichment: %w", err)
	}

	return enrich, nil
}

// Entity represents an extracted entity
type Entity struct {
	ID        int64
	MessageID string
	Type      string
	Value     string
	StartPos  *int
	EndPos    *int
	Metadata  *string
}

// SaveEntity saves an extracted entity
func (db *DB) SaveEntity(entity *Entity) error {
	result, err := db.Exec(`
		INSERT INTO entities (message_id, type, value, start_pos, end_pos, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, entity.MessageID, entity.Type, entity.Value, entity.StartPos, entity.EndPos, entity.Metadata)

	if err != nil {
		return fmt.Errorf("failed to save entity: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		entity.ID = id
	}

	return nil
}

// GetEntities retrieves all entities for a message
func (db *DB) GetEntities(messageID string) ([]*Entity, error) {
	rows, err := db.Query(`
		SELECT id, message_id, type, value, start_pos, end_pos, metadata
		FROM entities
		WHERE message_id = ?
	`, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query entities: %w", err)
	}
	defer rows.Close()

	entities := []*Entity{}
	for rows.Next() {
		entity := &Entity{}
		err := rows.Scan(&entity.ID, &entity.MessageID, &entity.Type, &entity.Value,
			&entity.StartPos, &entity.EndPos, &entity.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating entities: %w", err)
	}

	return entities, nil
}

// MessageRelation represents a relationship between messages
type MessageRelation struct {
	FromMessageID string
	ToMessageID   string
	RelationType  string
	Confidence    float64
}

// SaveMessageRelation saves a message relationship
func (db *DB) SaveMessageRelation(rel *MessageRelation) error {
	_, err := db.Exec(`
		INSERT INTO message_relations (from_message_id, to_message_id, relation_type, confidence)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(from_message_id, to_message_id, relation_type) DO UPDATE SET
			confidence = excluded.confidence
	`, rel.FromMessageID, rel.ToMessageID, rel.RelationType, rel.Confidence)

	if err != nil {
		return fmt.Errorf("failed to save message relation: %w", err)
	}

	return nil
}

// GetMessageRelations retrieves all relations for a message
func (db *DB) GetMessageRelations(messageID string, relationType *string) ([]*MessageRelation, error) {
	query := `
		SELECT from_message_id, to_message_id, relation_type, confidence
		FROM message_relations
		WHERE from_message_id = ? OR to_message_id = ?
	`
	args := []interface{}{messageID, messageID}

	if relationType != nil {
		query += " AND relation_type = ?"
		args = append(args, *relationType)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query message relations: %w", err)
	}
	defer rows.Close()

	relations := []*MessageRelation{}
	for rows.Next() {
		rel := &MessageRelation{}
		err := rows.Scan(&rel.FromMessageID, &rel.ToMessageID, &rel.RelationType, &rel.Confidence)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message relation: %w", err)
		}
		relations = append(relations, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message relations: %w", err)
	}

	return relations, nil
}
