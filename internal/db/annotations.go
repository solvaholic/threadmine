package db

import (
	"encoding/json"
	"fmt"
	"time"
)

// Classification represents a message classification
type Classification struct {
	MessageID   string
	Type        string
	Confidence  float64
	Signals     []string
	ClassifiedAt time.Time
}

// SaveClassification saves a message classification
func (db *DB) SaveClassification(class *Classification) error {
	signals, err := json.Marshal(class.Signals)
	if err != nil {
		return fmt.Errorf("failed to marshal signals: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO classifications (message_id, type, confidence, signals)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(message_id, type) DO UPDATE SET
			confidence = excluded.confidence,
			signals = excluded.signals,
			classified_at = CURRENT_TIMESTAMP
	`, class.MessageID, class.Type, class.Confidence, signals)

	if err != nil {
		return fmt.Errorf("failed to save classification: %w", err)
	}

	return nil
}

// GetClassifications retrieves all classifications for a message
func (db *DB) GetClassifications(messageID string) ([]*Classification, error) {
	rows, err := db.Query(`
		SELECT message_id, type, confidence, signals, classified_at
		FROM classifications
		WHERE message_id = ?
	`, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query classifications: %w", err)
	}
	defer rows.Close()

	classifications := []*Classification{}
	for rows.Next() {
		class := &Classification{}
		var signals string

		err := rows.Scan(&class.MessageID, &class.Type, &class.Confidence, &signals, &class.ClassifiedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan classification: %w", err)
		}

		if err := json.Unmarshal([]byte(signals), &class.Signals); err != nil {
			return nil, fmt.Errorf("failed to unmarshal signals: %w", err)
		}

		classifications = append(classifications, class)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating classifications: %w", err)
	}

	return classifications, nil
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
