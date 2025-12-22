package graph

import (
	"testing"
	"time"

	"github.com/solvaholic/threadmine/internal/normalize"
)

func TestReplyGraph_AddMessage(t *testing.T) {
	g := NewReplyGraph()

	// Create a root message
	root := &normalize.NormalizedMessage{
		ID:           "msg_slack_1234567890.123456",
		SourceType:   "slack",
		Timestamp:    time.Now(),
		IsThreadRoot: true,
		ThreadID:     "1234567890.123456",
		Author:       &normalize.User{ID: "user_slack_U123"},
		Channel:      &normalize.Channel{ID: "chan_slack_C123"},
	}

	g.AddMessage(root)

	if len(g.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(g.Nodes))
	}

	if len(g.ThreadRoots) != 1 {
		t.Errorf("Expected 1 thread root, got %d", len(g.ThreadRoots))
	}

	if g.ThreadRoots[0] != root.ID {
		t.Errorf("Expected thread root %s, got %s", root.ID, g.ThreadRoots[0])
	}
}

func TestReplyGraph_GetChildren(t *testing.T) {
	g := NewReplyGraph()

	// Create root and reply messages
	root := &normalize.NormalizedMessage{
		ID:           "msg_slack_1234567890.123456",
		IsThreadRoot: true,
		ThreadID:     "1234567890.123456",
	}

	reply1 := &normalize.NormalizedMessage{
		ID:       "msg_slack_1234567890.123457",
		ParentID: root.ID,
		ThreadID: "1234567890.123456",
	}

	reply2 := &normalize.NormalizedMessage{
		ID:       "msg_slack_1234567890.123458",
		ParentID: root.ID,
		ThreadID: "1234567890.123456",
	}

	g.AddMessage(root)
	g.AddMessage(reply1)
	g.AddMessage(reply2)

	children := g.GetChildren(root.ID)
	if len(children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(children))
	}

	// Verify children IDs
	expectedChildren := map[string]bool{
		reply1.ID: true,
		reply2.ID: true,
	}

	for _, childID := range children {
		if !expectedChildren[childID] {
			t.Errorf("Unexpected child ID: %s", childID)
		}
	}
}

func TestReplyGraph_GetThread(t *testing.T) {
	g := NewReplyGraph()

	// Create a thread with root and nested replies
	root := &normalize.NormalizedMessage{
		ID:           "msg_slack_1234567890.123456",
		IsThreadRoot: true,
		ThreadID:     "1234567890.123456",
	}

	reply1 := &normalize.NormalizedMessage{
		ID:       "msg_slack_1234567890.123457",
		ParentID: root.ID,
		ThreadID: "1234567890.123456",
	}

	reply2 := &normalize.NormalizedMessage{
		ID:       "msg_slack_1234567890.123458",
		ParentID: reply1.ID,
		ThreadID: "1234567890.123456",
	}

	g.AddMessage(root)
	g.AddMessage(reply1)
	g.AddMessage(reply2)

	thread := g.GetThread(root.ID)
	if len(thread) != 3 {
		t.Errorf("Expected 3 messages in thread, got %d", len(thread))
	}

	// Verify root is first
	if thread[0].MessageID != root.ID {
		t.Errorf("Expected root message first, got %s", thread[0].MessageID)
	}
}

func TestReplyGraph_GetThreadDepth(t *testing.T) {
	g := NewReplyGraph()

	// Create a thread with depth 2
	root := &normalize.NormalizedMessage{
		ID:           "msg_slack_1234567890.123456",
		IsThreadRoot: true,
		ThreadID:     "1234567890.123456",
	}

	reply1 := &normalize.NormalizedMessage{
		ID:       "msg_slack_1234567890.123457",
		ParentID: root.ID,
		ThreadID: "1234567890.123456",
	}

	reply2 := &normalize.NormalizedMessage{
		ID:       "msg_slack_1234567890.123458",
		ParentID: reply1.ID,
		ThreadID: "1234567890.123456",
	}

	g.AddMessage(root)
	g.AddMessage(reply1)
	g.AddMessage(reply2)

	depth := g.GetThreadDepth(root.ID)
	if depth != 2 {
		t.Errorf("Expected depth 2, got %d", depth)
	}
}

func TestReplyGraph_Stats(t *testing.T) {
	g := NewReplyGraph()

	// Create multiple threads
	root1 := &normalize.NormalizedMessage{
		ID:           "msg_slack_1234567890.123456",
		IsThreadRoot: true,
		ThreadID:     "1234567890.123456",
	}

	reply1 := &normalize.NormalizedMessage{
		ID:       "msg_slack_1234567890.123457",
		ParentID: root1.ID,
		ThreadID: "1234567890.123456",
	}

	root2 := &normalize.NormalizedMessage{
		ID:           "msg_slack_1234567891.123456",
		IsThreadRoot: true,
		ThreadID:     "1234567891.123456",
	}

	g.AddMessage(root1)
	g.AddMessage(reply1)
	g.AddMessage(root2)

	stats := g.Stats()

	if stats["total_messages"] != 3 {
		t.Errorf("Expected 3 total messages, got %v", stats["total_messages"])
	}

	if stats["thread_count"] != 2 {
		t.Errorf("Expected 2 threads, got %v", stats["thread_count"])
	}

	if stats["reply_messages"] != 1 {
		t.Errorf("Expected 1 reply message, got %v", stats["reply_messages"])
	}

	if stats["messages_with_replies"] != 1 {
		t.Errorf("Expected 1 message with replies, got %v", stats["messages_with_replies"])
	}
}

func TestBuildFromNormalizedMessages(t *testing.T) {
	messages := []*normalize.NormalizedMessage{
		{
			ID:           "msg_slack_1234567890.123456",
			IsThreadRoot: true,
			ThreadID:     "1234567890.123456",
		},
		{
			ID:       "msg_slack_1234567890.123457",
			ParentID: "msg_slack_1234567890.123456",
			ThreadID: "1234567890.123456",
		},
		{
			ID:           "msg_slack_1234567891.123456",
			IsThreadRoot: true,
			ThreadID:     "1234567891.123456",
		},
	}

	g := BuildFromNormalizedMessages(messages)

	if len(g.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(g.Nodes))
	}

	if len(g.ThreadRoots) != 2 {
		t.Errorf("Expected 2 thread roots, got %d", len(g.ThreadRoots))
	}

	children := g.GetChildren("msg_slack_1234567890.123456")
	if len(children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(children))
	}
}
