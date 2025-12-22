package normalize

import (
	"testing"
	"time"
)

func TestSlackToNormalized(t *testing.T) {
	// Test message with basic content
	msg := &SlackMessage{
		Type:      "message",
		User:      "U123",
		Text:      "Hello, world!",
		Timestamp: "1234567890.123456",
	}
	
	channel := &SlackChannel{
		ID:        "C123",
		Name:      "general",
		IsChannel: true,
		IsPrivate: false,
	}
	
	user := &SlackUser{
		ID:       "U123",
		Name:     "testuser",
		RealName: "Test User",
	}
	
	normalized, err := SlackToNormalized(msg, channel, user, "T123", time.Now())
	if err != nil {
		t.Fatalf("Failed to normalize message: %v", err)
	}
	
	// Verify fields
	if normalized.SourceType != "slack" {
		t.Errorf("Expected source_type 'slack', got '%s'", normalized.SourceType)
	}
	
	if normalized.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", normalized.Content)
	}
	
	if normalized.Author == nil {
		t.Fatal("Author is nil")
	}
	
	if normalized.Author.DisplayName != "testuser" {
		t.Errorf("Expected author 'testuser', got '%s'", normalized.Author.DisplayName)
	}
	
	if normalized.Channel == nil {
		t.Fatal("Channel is nil")
	}
	
	if normalized.Channel.Name != "general" {
		t.Errorf("Expected channel 'general', got '%s'", normalized.Channel.Name)
	}
	
	if !normalized.IsThreadRoot {
		t.Error("Expected message to be thread root")
	}
	
	if normalized.SchemaVersion != SchemaVersion {
		t.Errorf("Expected schema version '%s', got '%s'", SchemaVersion, normalized.SchemaVersion)
	}
}

func TestSlackToNormalizedWithThread(t *testing.T) {
	// Test message that's a reply in a thread
	msg := &SlackMessage{
		Type:      "message",
		User:      "U123",
		Text:      "This is a reply",
		Timestamp: "1234567890.123457",
		ThreadTS:  "1234567890.123456",
	}
	
	channel := &SlackChannel{
		ID:        "C123",
		Name:      "general",
		IsChannel: true,
		IsPrivate: false,
	}
	
	user := &SlackUser{
		ID:       "U123",
		Name:     "testuser",
		RealName: "Test User",
	}
	
	normalized, err := SlackToNormalized(msg, channel, user, "T123", time.Now())
	if err != nil {
		t.Fatalf("Failed to normalize message: %v", err)
	}
	
	// Verify thread fields
	if normalized.IsThreadRoot {
		t.Error("Expected message to not be thread root")
	}
	
	if normalized.ThreadID == "" {
		t.Error("Expected thread_id to be set")
	}
	
	if normalized.ParentID == "" {
		t.Error("Expected parent_id to be set")
	}
	
	expectedThreadID := "thread_slack_T123_C123_1234567890.123456"
	if normalized.ThreadID != expectedThreadID {
		t.Errorf("Expected thread_id '%s', got '%s'", expectedThreadID, normalized.ThreadID)
	}
	
	expectedParentID := "msg_slack_T123_C123_1234567890.123456"
	if normalized.ParentID != expectedParentID {
		t.Errorf("Expected parent_id '%s', got '%s'", expectedParentID, normalized.ParentID)
	}
}

func TestSlackMarkupNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "user mention with label",
			input:    "Hey <@U123|john>, how are you?",
			expected: "Hey @john, how are you?",
		},
		{
			name:     "user mention without label",
			input:    "Hey <@U123>, how are you?",
			expected: "Hey @U123, how are you?",
		},
		{
			name:     "channel mention",
			input:    "Check out <#C123|general>",
			expected: "Check out #general",
		},
		{
			name:     "URL with label",
			input:    "See <https://example.com|this link>",
			expected: "See this link (https://example.com)",
		},
		{
			name:     "URL without label",
			input:    "See <https://example.com>",
			expected: "See https://example.com",
		},
		{
			name:     "HTML entities",
			input:    "Use &lt;div&gt; tags &amp; styles",
			expected: "Use <div> tags & styles",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSlackText(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestExtractMentions(t *testing.T) {
	text := "Hey <@U123|john> and <@U456>, check this out"
	mentions := extractMentions(text)
	
	if len(mentions) != 2 {
		t.Fatalf("Expected 2 mentions, got %d", len(mentions))
	}
	
	if mentions[0] != "U123" {
		t.Errorf("Expected first mention 'U123', got '%s'", mentions[0])
	}
	
	if mentions[1] != "U456" {
		t.Errorf("Expected second mention 'U456', got '%s'", mentions[1])
	}
}

func TestExtractURLs(t *testing.T) {
	text := "Check <https://example.com|example> and <http://test.com>"
	urls := extractURLs(text)
	
	if len(urls) != 2 {
		t.Fatalf("Expected 2 URLs, got %d", len(urls))
	}
	
	if urls[0] != "https://example.com" {
		t.Errorf("Expected first URL 'https://example.com', got '%s'", urls[0])
	}
	
	if urls[1] != "http://test.com" {
		t.Errorf("Expected second URL 'http://test.com', got '%s'", urls[1])
	}
}

func TestExtractCodeBlocks(t *testing.T) {
	text := "Here's some code:\n```python\nprint('hello')\n```\nAnd more:\n```\nplain text\n```"
	blocks := extractCodeBlocks(text)
	
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 code blocks, got %d", len(blocks))
	}
	
	if blocks[0].Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", blocks[0].Language)
	}
	
	if blocks[0].Code != "print('hello')\n" {
		t.Errorf("Expected code 'print('hello')\\n', got '%s'", blocks[0].Code)
	}
	
	if blocks[1].Language != "" {
		t.Errorf("Expected empty language, got '%s'", blocks[1].Language)
	}
}

func TestParseSlackTimestamp(t *testing.T) {
	ts, err := parseSlackTimestamp("1234567890.123456")
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}
	
	// Check the timestamp is within a reasonable range (nanosecond precision may vary slightly)
	expectedSec := int64(1234567890)
	if ts.Unix() != expectedSec {
		t.Errorf("Expected Unix timestamp %d, got %d", expectedSec, ts.Unix())
	}
	
	// Check nanoseconds are in reasonable range (123456 microseconds = 123456000 nanoseconds)
	expectedNano := int64(123456000)
	actualNano := int64(ts.Nanosecond())
	diff := actualNano - expectedNano
	if diff < -1000 || diff > 1000 { // Allow 1 microsecond tolerance
		t.Errorf("Expected nanoseconds around %d, got %d (diff: %d)", expectedNano, actualNano, diff)
	}
}
