package main

import (
	"fmt"

	"github.com/solvaholic/threadmine/internal/classify"
	"github.com/solvaholic/threadmine/internal/normalize"
)

func main() {
	fmt.Println("ThreadMine - Message Enrichment Demo")
	fmt.Println()

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
			desc:    "Message with code block",
		},
		{
			content: "I'm stuck on getting the tests to pass",
			desc:    "Help-seeking without question mark",
		},
		{
			content: "Thanks! That worked perfectly.",
			desc:    "Plain text with no question or content flags",
		},
		{
			content: "Check out the documentation at https://docs.example.com/guide",
			desc:    "Message with documentation link",
		},
		{
			content: "> The deployment completed successfully\nThanks for the update.",
			desc:    "Reply with a block quote",
		},
	}

	for i, ex := range examples {
		fmt.Printf("%d. %s\n", i+1, ex.desc)
		fmt.Printf("   Message: \"%s\"\n", ex.content)

		msg := &normalize.NormalizedMessage{
			ID:         fmt.Sprintf("msg_demo_%d", i+1),
			Content:    ex.content,
			CodeBlocks: normalize.ExtractCodeBlocks(ex.content),
			URLs:       normalize.ExtractURLs(ex.content),
		}

		enrichment := classify.EnrichMessage(msg)
		fmt.Printf("   Question: %t\n", enrichment.IsQuestion)
		fmt.Printf("   Characters: %d, words: %d\n", enrichment.CharCount, enrichment.WordCount)
		fmt.Printf("   Code: %t, links: %t, quotes: %t\n\n", enrichment.HasCode, enrichment.HasLinks, enrichment.HasQuotes)
	}

	fmt.Println("Enrichment complete!")
}
