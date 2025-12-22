package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/solvaholic/threadmine/internal/cache"
	"github.com/solvaholic/threadmine/internal/slack"
)

func main() {
	// Authenticate with Slack
	result, err := slack.Authenticate("solvahol")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Testing cache-aside pattern for message retrieval\n\n")

	ctx := context.Background()

	// Get channels
	channels, err := result.Client.ListChannels(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing channels: %v\n", err)
		os.Exit(1)
	}

	if len(channels) == 0 {
		fmt.Printf("No channels found.\n")
		return
	}

	firstChannel := channels[0]
	oldest := time.Now().AddDate(0, 0, -7)
	cacheDir, _ := cache.CacheDir()

	// First retrieval (may be cache or API)
	fmt.Printf("First retrieval from #%s...\n", firstChannel.Name)
	start1 := time.Now()
	messages1, err := result.Client.GetMessages(ctx, firstChannel.ID, oldest, cacheDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	duration1 := time.Since(start1)
	fmt.Printf("✓ Retrieved %d messages in %v\n\n", len(messages1), duration1)

	// Second retrieval (should be from cache)
	fmt.Printf("Second retrieval from #%s...\n", firstChannel.Name)
	start2 := time.Now()
	messages2, err := result.Client.GetMessages(ctx, firstChannel.ID, oldest, cacheDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	duration2 := time.Since(start2)
	fmt.Printf("✓ Retrieved %d messages in %v\n\n", len(messages2), duration2)

	// Show performance improvement
	if duration2 < duration1 {
		speedup := float64(duration1) / float64(duration2)
		fmt.Printf("Cache-aside pattern speedup: %.2fx faster\n", speedup)
	} else {
		fmt.Printf("Both retrievals completed successfully\n")
	}

	// Verify data is identical
	if len(messages1) == len(messages2) {
		fmt.Printf("✓ Data consistency verified: same message count\n")
	} else {
		fmt.Printf("⚠ Warning: message count differs (%d vs %d)\n", len(messages1), len(messages2))
	}
}
