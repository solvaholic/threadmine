package main

import (
	"fmt"

	"github.com/solvaholic/threadmine/internal/classify"
	"github.com/solvaholic/threadmine/internal/normalize"
)

func main() {
	testCases := []struct {
		content  string
		expected string
	}{
		{
			content:  "The city streets were gritty after the storm.",
			expected: "NO acknowledgment (gritty contains 'ty' substring)",
		},
		{
			content:  "That's a pretty interesting approach.",
			expected: "NO acknowledgment (pretty contains 'ty' substring)",
		},
		{
			content:  "I'm working on the database migration this week.",
			expected: "NO acknowledgment (working on != success)",
		},
		{
			content:  "Thanks! That worked perfectly.",
			expected: "YES acknowledgment (valid thanks + success)",
		},
		{
			content:  "ty so much!",
			expected: "YES acknowledgment (valid standalone ty)",
		},
	}

	for i, tc := range testCases {
		msg := &normalize.NormalizedMessage{
			Content: tc.content,
		}

		classifications := classify.ClassifyMessage(msg, nil)

		hasAck := false
		for _, c := range classifications {
			if c.Type == "acknowledgment" {
				hasAck = true
				fmt.Printf("Test %d: Found acknowledgment (%.2f) - Signals: %v\n", i+1, c.Confidence, c.Signals)
			}
		}

		if !hasAck {
			fmt.Printf("Test %d: No acknowledgment found\n", i+1)
		}

		fmt.Printf("  Expected: %s\n", tc.expected)
		fmt.Println()
	}
}
