package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/solvaholic/threadmine/internal/graph"
)

func main() {
	// Load the graph
	g, err := graph.LoadReplyGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading graph: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Reply Graph Summary\n")
	fmt.Printf("===================\n\n")

	// Display stats
	stats := g.Stats()
	fmt.Printf("Total Messages: %v\n", stats["total_messages"])
	fmt.Printf("Thread Count: %v\n", stats["thread_count"])
	fmt.Printf("Messages with Replies: %v\n", stats["messages_with_replies"])
	fmt.Printf("Reply Messages: %v\n", stats["reply_messages"])
	fmt.Printf("Average Thread Depth: %.2f\n", stats["average_thread_depth"])
	fmt.Printf("Updated: %v\n\n", stats["updated_at"])

	// Find threads with replies
	fmt.Printf("Threads with Replies:\n")
	fmt.Printf("---------------------\n")
	threadsWithReplies := 0
	for _, rootID := range g.ThreadRoots {
		children := g.GetChildren(rootID)
		if len(children) > 0 {
			threadsWithReplies++
			thread := g.GetThread(rootID)
			depth := g.GetThreadDepth(rootID)
			
			fmt.Printf("\nThread Root: %s\n", rootID)
			fmt.Printf("  Messages: %d\n", len(thread))
			fmt.Printf("  Depth: %d\n", depth)
			fmt.Printf("  Direct Replies: %d\n", len(children))
			
			// Display thread structure
			fmt.Printf("  Structure:\n")
			for i, node := range thread {
				indent := ""
				if i > 0 {
					indent = "    "
				}
				fmt.Printf("    %s- %s (%s)\n", indent, node.MessageID, node.Timestamp.Format("15:04:05"))
			}
		}
	}

	if threadsWithReplies == 0 {
		fmt.Printf("No threads with replies found.\n")
	}

	// Output full graph as JSON
	fmt.Printf("\n\nFull Graph Data:\n")
	fmt.Printf("================\n")
	data := map[string]interface{}{
		"nodes":        g.Nodes,
		"adjacency":    g.Adjacency,
		"thread_roots": g.ThreadRoots,
		"stats":        stats,
	}

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}
