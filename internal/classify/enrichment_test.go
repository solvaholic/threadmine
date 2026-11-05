package classify

import (
	"strings"
	"testing"

	"github.com/solvaholic/threadmine/internal/normalize"
)

func TestEnrichMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  normalize.NormalizedMessage
		want Enrichment
	}{
		{
			name: "empty message",
		},
		{
			name: "plain text and message ID",
			msg: normalize.NormalizedMessage{
				ID:      "msg_github_example",
				Content: "Hello, world!",
			},
			want: Enrichment{
				MessageID: "msg_github_example",
				CharCount: 13,
				WordCount: 2,
			},
		},
		{
			name: "question with code link and quote",
			msg: normalize.NormalizedMessage{
				ID:         "msg_slack_example",
				Content:    "How?\n> `x()` https://example.com",
				CodeBlocks: []normalize.CodeBlock{{Code: "x()"}},
				URLs:       []string{"https://example.com"},
			},
			want: Enrichment{
				MessageID:  "msg_slack_example",
				IsQuestion: true,
				CharCount:  32,
				WordCount:  4,
				HasCode:    true,
				HasLinks:   true,
				HasQuotes:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnrichMessage(&tt.msg)
			if got == nil {
				t.Fatal("EnrichMessage returned nil")
			}
			if *got != tt.want {
				t.Errorf("EnrichMessage() = %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestEnrichMessageQuestions(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"question mark", "Any suggestions?", true},
		{"question starter without punctuation", "anyone know how to fix this", true},
		{"case insensitive starter", "HOW DO I configure this", true},
		{"help phrase within message", "I'm stuck on configuring the server", true},
		{"case insensitive help phrase", "I NEED HELP configuring the server", true},
		{"help phrase at length threshold", "need help" + strings.Repeat(" ", 11), false},
		{"help phrase above length threshold", "need help" + strings.Repeat(" ", 12), true},
		{"short help phrase", "help me", false},
		{"question mark in short message", "?", true},
		{"solution statement", "Here's the solution for that.", false},
		{"acknowledgment", "Thanks! That worked perfectly.", false},
		{"ordinary statement", "The server is running fine now.", false},
		{"empty content", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnrichMessage(&normalize.NormalizedMessage{Content: tt.content})
			if got.IsQuestion != tt.want {
				t.Errorf("IsQuestion for %q = %t, want %t", tt.content, got.IsQuestion, tt.want)
			}
		})
	}
}

func TestEnrichMessageCounts(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantChars int
		wantWords int
	}{
		{"empty", "", 0, 0},
		{"whitespace only", " \t\n", 3, 0},
		{"single word", "hello", 5, 1},
		{"mixed whitespace", "  one\t two\nthree  ", 18, 3},
		{"punctuation stays in words", "Hello, world!", 13, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnrichMessage(&normalize.NormalizedMessage{Content: tt.content})
			if got.CharCount != tt.wantChars {
				t.Errorf("CharCount = %d, want %d", got.CharCount, tt.wantChars)
			}
			if got.WordCount != tt.wantWords {
				t.Errorf("WordCount = %d, want %d", got.WordCount, tt.wantWords)
			}
		})
	}

	t.Run("Unicode whitespace separates words", func(t *testing.T) {
		got := EnrichMessage(&normalize.NormalizedMessage{Content: "one\u2003two\u00a0three"})
		if got.WordCount != 3 {
			t.Errorf("WordCount = %d, want 3", got.WordCount)
		}
	})
}

func TestEnrichMessageContentFlags(t *testing.T) {
	tests := []struct {
		name       string
		msg        normalize.NormalizedMessage
		wantCode   bool
		wantLinks  bool
		wantQuotes bool
	}{
		{
			name: "nil metadata",
		},
		{
			name: "empty metadata",
			msg: normalize.NormalizedMessage{
				CodeBlocks: []normalize.CodeBlock{},
				URLs:       []string{},
			},
		},
		{
			name: "code from normalized metadata",
			msg: normalize.NormalizedMessage{
				CodeBlocks: []normalize.CodeBlock{{Code: "fmt.Println()"}},
			},
			wantCode: true,
		},
		{
			name: "links from normalized metadata",
			msg: normalize.NormalizedMessage{
				URLs: []string{"https://example.com"},
			},
			wantLinks: true,
		},
		{
			name: "extraction belongs to normalization",
			msg: normalize.NormalizedMessage{
				Content: "`fmt.Println()` https://example.com",
			},
		},
		{
			name:       "quote at start",
			msg:        normalize.NormalizedMessage{Content: "> quoted text"},
			wantQuotes: true,
		},
		{
			name:       "indented quote on later line",
			msg:        normalize.NormalizedMessage{Content: "Reply\n \t> quoted text"},
			wantQuotes: true,
		},
		{
			name: "greater than within line is not a quote",
			msg:  normalize.NormalizedMessage{Content: "value > limit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnrichMessage(&tt.msg)
			if got.HasCode != tt.wantCode {
				t.Errorf("HasCode = %t, want %t", got.HasCode, tt.wantCode)
			}
			if got.HasLinks != tt.wantLinks {
				t.Errorf("HasLinks = %t, want %t", got.HasLinks, tt.wantLinks)
			}
			if got.HasQuotes != tt.wantQuotes {
				t.Errorf("HasQuotes = %t, want %t", got.HasQuotes, tt.wantQuotes)
			}
		})
	}
}
