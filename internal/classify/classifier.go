package classify

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/solvaholic/threadmine/internal/normalize"
)

// Enrichment represents basic message metadata
type Enrichment struct {
	MessageID  string `json:"message_id"`
	IsQuestion bool   `json:"is_question"`
	CharCount  int    `json:"char_count"`
	WordCount  int    `json:"word_count"`
	HasCode    bool   `json:"has_code"`
	HasLinks   bool   `json:"has_links"`
	HasQuotes  bool   `json:"has_quotes"`
}

// EnrichMessage analyzes a message and returns basic enrichment metadata
func EnrichMessage(msg *normalize.NormalizedMessage) *Enrichment {
	return &Enrichment{
		MessageID:  msg.ID,
		IsQuestion: detectQuestion(msg),
		CharCount:  len(msg.Content),
		WordCount:  countWords(msg.Content),
		HasCode:    len(msg.CodeBlocks) > 0,
		HasLinks:   len(msg.URLs) > 0,
		HasQuotes:  detectQuotes(msg.Content),
	}
}

// detectQuestion checks if a message looks like a question
// Uses existing patterns: question marks, question words, help-seeking phrases
func detectQuestion(msg *normalize.NormalizedMessage) bool {
	content := strings.ToLower(msg.Content)

	// Strong signal: Contains question mark
	if strings.Contains(content, "?") {
		return true
	}

	// Question words at start
	questionStarters := []string{
		"how do i", "how can i", "how to", "how would",
		"what is", "what's", "what are", "what if",
		"where is", "where can", "where do",
		"when should", "when do", "when is",
		"why does", "why is", "why would",
		"who can", "who is", "who knows",
		"can someone", "can anyone", "could someone",
		"is there", "are there",
		"does anyone", "does someone",
		"has anyone", "has someone",
		"should i", "would it",
		"any ideas", "anyone know",
	}

	for _, starter := range questionStarters {
		if strings.HasPrefix(content, starter) {
			return true
		}
	}

	// Help-seeking phrases (require both the phrase and reasonable message length)
	if len(msg.Content) > 20 {
		helpPhrases := []string{
			"help me", "stuck on", "having trouble", "problem with",
			"error with", "not working", "doesn't work", "can't get",
			"unable to", "trying to figure", "need help",
		}

		for _, phrase := range helpPhrases {
			if strings.Contains(content, phrase) {
				return true
			}
		}
	}

	return false
}

// countWords counts words in the message content
func countWords(content string) int {
	// Simple word counting: split on whitespace and count non-empty segments
	count := 0
	inWord := false

	for _, r := range content {
		if unicode.IsSpace(r) {
			if inWord {
				count++
				inWord = false
			}
		} else {
			inWord = true
		}
	}

	// Count the last word if we ended in the middle of one
	if inWord {
		count++
	}

	return count
}

// detectQuotes checks if the message contains markdown-style block quotes
// Looks for lines starting with '>' (possibly preceded by whitespace)
func detectQuotes(content string) bool {
	// Match lines that start with optional whitespace followed by '>'
	quotePattern := regexp.MustCompile(`(?m)^\s*>`)
	return quotePattern.MatchString(content)
}
