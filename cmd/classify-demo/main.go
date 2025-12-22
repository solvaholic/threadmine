package main

import (
	"fmt"

	"github.com/solvaholic/threadmine/internal/classify"
	"github.com/solvaholic/threadmine/internal/normalize"
)

func main() {
	fmt.Println("ThreadMine - Message Classification Demo")
	fmt.Println()

	// Example messages to classify
	examples := []struct {
		content string
		desc    string
	}{
		{
			content: "How do I configure rate limiting in the API?",
			desc:    "Question with question mark and 'how do i'",
		},
		{
			content: "You can configure rate limiting by adding this to your config:\n```yaml\nrate_limit:\n  requests_per_second: 100\n```",
			desc:    "Solution with code block",
		},
		{
			content: "I'm stuck trying to get the tests to pass",
			desc:    "Help-seeking without question mark",
		},
		{
			content: "Thanks! That worked perfectly.",
			desc:    "Acknowledgment with thanks and success",
		},
		{
			content: "Check out the documentation at https://docs.example.com/guide",
			desc:    "Solution with documentation link",
		},
		{
			content: "The deployment completed successfully",
			desc:    "Regular statement (no classification expected)",
		},
	}

	for i, ex := range examples {
		fmt.Printf("%d. %s\n", i+1, ex.desc)
		fmt.Printf("   Message: \"%s\"\n", ex.content)

		// Create a normalized message
		msg := &normalize.NormalizedMessage{
			Content: ex.content,
		}

		// Add code blocks if present
		if i == 1 {
			msg.CodeBlocks = []normalize.CodeBlock{
				{Language: "yaml", Code: "rate_limit:\n  requests_per_second: 100"},
			}
		}

		// Add URLs if present
		if i == 4 {
			msg.URLs = []string{"https://docs.example.com/guide"}
		}

		// Classify the message
		classifications := classify.ClassifyMessage(msg, nil)

		if len(classifications) == 0 {
			fmt.Printf("   Classification: None\n\n")
		} else {
			for _, c := range classifications {
				fmt.Printf("   Classification: %s (confidence: %.2f)\n", c.Type, c.Confidence)
				fmt.Printf("   Signals: %v\n", c.Signals)
			}
			fmt.Println()
		}
	}

	fmt.Println("Classification complete! 🎉")
}
