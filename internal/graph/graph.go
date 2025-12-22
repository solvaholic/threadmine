package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/solvaholic/threadmine/internal/normalize"
)

// MessageNode represents a node in the reply graph
type MessageNode struct {
	MessageID    string    `json:"message_id"`
	ThreadID     string    `json:"thread_id"`
	ParentID     string    `json:"parent_id"`
	IsThreadRoot bool      `json:"is_thread_root"`
	Author       string    `json:"author"`
	Timestamp    time.Time `json:"timestamp"`
	Channel      string    `json:"channel"`
	SourceType   string    `json:"source_type"`
}

// ReplyGraph represents the message reply structure
type ReplyGraph struct {
	Nodes       map[string]*MessageNode `json:"nodes"`        // message_id -> node
	Adjacency   map[string][]string     `json:"adjacency"`    // parent_id -> [child_ids]
	ThreadRoots []string                `json:"thread_roots"` // list of thread root message IDs
	UpdatedAt   time.Time               `json:"updated_at"`
}

// NewReplyGraph creates a new empty reply graph
func NewReplyGraph() *ReplyGraph {
	return &ReplyGraph{
		Nodes:       make(map[string]*MessageNode),
		Adjacency:   make(map[string][]string),
		ThreadRoots: []string{},
		UpdatedAt:   time.Now(),
	}
}

// AddMessage adds a message to the reply graph
func (g *ReplyGraph) AddMessage(msg *normalize.NormalizedMessage) {
	// Create node
	node := &MessageNode{
		MessageID:    msg.ID,
		ThreadID:     msg.ThreadID,
		ParentID:     msg.ParentID,
		IsThreadRoot: msg.IsThreadRoot,
		Timestamp:    msg.Timestamp,
		SourceType:   msg.SourceType,
	}

	// Extract author ID
	if msg.Author != nil {
		node.Author = msg.Author.ID
	}

	// Extract channel ID
	if msg.Channel != nil {
		node.Channel = msg.Channel.ID
	}

	// Add to nodes map
	g.Nodes[msg.ID] = node

	// Track thread roots
	if msg.IsThreadRoot {
		g.ThreadRoots = append(g.ThreadRoots, msg.ID)
	}

	// Build adjacency list (parent -> children)
	if msg.ParentID != "" {
		g.Adjacency[msg.ParentID] = append(g.Adjacency[msg.ParentID], msg.ID)
	}

	g.UpdatedAt = time.Now()
}

// GetChildren returns the direct children of a message
func (g *ReplyGraph) GetChildren(messageID string) []string {
	return g.Adjacency[messageID]
}

// GetThread returns all messages in a thread, starting from the root
func (g *ReplyGraph) GetThread(rootID string) []*MessageNode {
	result := []*MessageNode{}
	
	// Check if the root exists
	root, exists := g.Nodes[rootID]
	if !exists {
		return result
	}

	// Add root
	result = append(result, root)

	// Recursively add children
	g.collectThreadMessages(rootID, &result)

	return result
}

// collectThreadMessages recursively collects all messages in a thread
func (g *ReplyGraph) collectThreadMessages(messageID string, result *[]*MessageNode) {
	children := g.GetChildren(messageID)
	for _, childID := range children {
		if node, exists := g.Nodes[childID]; exists {
			*result = append(*result, node)
			// Recursively collect children of this child
			g.collectThreadMessages(childID, result)
		}
	}
}

// GetThreadDepth returns the maximum depth of a thread
func (g *ReplyGraph) GetThreadDepth(rootID string) int {
	if _, exists := g.Nodes[rootID]; !exists {
		return 0
	}
	return g.calculateDepth(rootID, 0)
}

// calculateDepth recursively calculates thread depth
func (g *ReplyGraph) calculateDepth(messageID string, currentDepth int) int {
	children := g.GetChildren(messageID)
	if len(children) == 0 {
		return currentDepth
	}

	maxDepth := currentDepth
	for _, childID := range children {
		depth := g.calculateDepth(childID, currentDepth+1)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth
}

// Stats returns statistics about the graph
func (g *ReplyGraph) Stats() map[string]interface{} {
	threadCount := len(g.ThreadRoots)
	totalMessages := len(g.Nodes)
	threadRootMessages := len(g.ThreadRoots)
	replyMessages := totalMessages - threadRootMessages

	// Calculate average thread depth
	totalDepth := 0
	for _, rootID := range g.ThreadRoots {
		totalDepth += g.GetThreadDepth(rootID)
	}
	avgDepth := 0.0
	if threadCount > 0 {
		avgDepth = float64(totalDepth) / float64(threadCount)
	}

	// Count messages with replies
	messagesWithReplies := 0
	for _, children := range g.Adjacency {
		if len(children) > 0 {
			messagesWithReplies++
		}
	}

	return map[string]interface{}{
		"total_messages":         totalMessages,
		"thread_count":           threadCount,
		"reply_messages":         replyMessages,
		"messages_with_replies":  messagesWithReplies,
		"average_thread_depth":   avgDepth,
		"updated_at":             g.UpdatedAt.Format(time.RFC3339),
	}
}

// GraphDir returns the root directory for graph data
func GraphDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".threadmine", "graph"), nil
}

// StructureDir returns the directory for graph structure files
func StructureDir() (string, error) {
	graphDir, err := GraphDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(graphDir, "structure"), nil
}

// SaveReplyGraph saves the reply graph to disk
func SaveReplyGraph(g *ReplyGraph) error {
	dir, err := StructureDir()
	if err != nil {
		return err
	}

	// Create directory with restrictive permissions
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save adjacency list
	if err := saveGraphFile(dir, "adjacency.json", g.Adjacency); err != nil {
		return fmt.Errorf("failed to save adjacency list: %w", err)
	}

	// Save nodes
	if err := saveGraphFile(dir, "nodes.json", g.Nodes); err != nil {
		return fmt.Errorf("failed to save nodes: %w", err)
	}

	// Save thread roots
	if err := saveGraphFile(dir, "thread_roots.json", g.ThreadRoots); err != nil {
		return fmt.Errorf("failed to save thread roots: %w", err)
	}

	// Save metadata
	metadata := map[string]interface{}{
		"updated_at": g.UpdatedAt.Format(time.RFC3339),
		"stats":      g.Stats(),
	}
	if err := saveGraphFile(dir, "metadata.json", metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// saveGraphFile saves a data structure to a JSON file atomically
func saveGraphFile(dir, filename string, data interface{}) error {
	filePath := filepath.Join(dir, filename)

	// Marshal to JSON with indentation for human readability
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write to temp file first, then rename (atomic write)
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// LoadReplyGraph loads the reply graph from disk
func LoadReplyGraph() (*ReplyGraph, error) {
	dir, err := StructureDir()
	if err != nil {
		return nil, err
	}

	g := NewReplyGraph()

	// Load nodes
	nodesPath := filepath.Join(dir, "nodes.json")
	if err := loadGraphFile(nodesPath, &g.Nodes); err != nil {
		return nil, fmt.Errorf("failed to load nodes: %w", err)
	}

	// Load adjacency list
	adjacencyPath := filepath.Join(dir, "adjacency.json")
	if err := loadGraphFile(adjacencyPath, &g.Adjacency); err != nil {
		return nil, fmt.Errorf("failed to load adjacency list: %w", err)
	}

	// Load thread roots
	rootsPath := filepath.Join(dir, "thread_roots.json")
	if err := loadGraphFile(rootsPath, &g.ThreadRoots); err != nil {
		return nil, fmt.Errorf("failed to load thread roots: %w", err)
	}

	// Load metadata to get updated_at
	metadataPath := filepath.Join(dir, "metadata.json")
	var metadata map[string]interface{}
	if err := loadGraphFile(metadataPath, &metadata); err == nil {
		if updatedAtStr, ok := metadata["updated_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
				g.UpdatedAt = t
			}
		}
	}

	return g, nil
}

// loadGraphFile loads data from a JSON file
func loadGraphFile(filePath string, v interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("graph file not found: %s", filePath)
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// BuildFromNormalizedMessages builds a reply graph from a slice of normalized messages
func BuildFromNormalizedMessages(messages []*normalize.NormalizedMessage) *ReplyGraph {
	g := NewReplyGraph()
	for _, msg := range messages {
		g.AddMessage(msg)
	}
	return g
}
