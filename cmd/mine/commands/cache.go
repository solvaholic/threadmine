package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/solvaholic/threadmine/internal/cache"
	"github.com/solvaholic/threadmine/internal/classify"
	"github.com/solvaholic/threadmine/internal/graph"
	"github.com/solvaholic/threadmine/internal/normalize"
)

// cacheCmd represents the cache command
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage cache",
	Long:  `Inspect and manage the local cache of fetched data.`,
}

// cacheInfoCmd represents the cache info command
var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show cache information",
	Long:  `Display statistics about the local cache including size, message counts, and date ranges.`,
	RunE:  runCacheInfo,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheInfoCmd)
}

func runCacheInfo(cmd *cobra.Command, args []string) error {
	cacheDir, err := cache.CacheDir()
	if err != nil {
		OutputError("failed to get cache directory: %v", err)
		return err
	}

	normalizedDir, _ := normalize.NormalizedDir()
	graphDir, _ := graph.GraphDir()
	annotationsDir, _ := classify.AnnotationsDir()

	// Calculate directory sizes and counts
	rawStats := calculateDirStats(cacheDir)
	normalizedStats := calculateDirStats(normalizedDir)
	graphStats := calculateDirStats(graphDir)
	annotationsStats := calculateDirStats(annotationsDir)

	// Count messages by source
	messageCounts := make(map[string]int)
	var earliestMsg, latestMsg time.Time

	messagesDir := filepath.Join(normalizedDir, "messages", "by_source")
	if entries, err := os.ReadDir(messagesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
				continue
			}

			source := strings.TrimSuffix(entry.Name(), ".jsonl")
			sourcePath := filepath.Join(messagesDir, entry.Name())

			data, err := os.ReadFile(sourcePath)
			if err != nil {
				continue
			}

			lines := strings.Split(string(data), "\n")
			count := 0
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				count++

				// Parse timestamp for date range
				var msg normalize.NormalizedMessage
				if err := json.Unmarshal([]byte(line), &msg); err == nil {
					if earliestMsg.IsZero() || msg.Timestamp.Before(earliestMsg) {
						earliestMsg = msg.Timestamp
					}
					if latestMsg.IsZero() || msg.Timestamp.After(latestMsg) {
						latestMsg = msg.Timestamp
					}
				}
			}

			messageCounts[source] = count
		}
	}

	// Build output
	output := map[string]interface{}{
		"status":         "success",
		"cache_location": filepath.Dir(cacheDir),
		"total_size":     formatBytes(rawStats.Size + normalizedStats.Size + graphStats.Size + annotationsStats.Size),
		"by_layer": map[string]interface{}{
			"raw": map[string]interface{}{
				"size":  formatBytes(rawStats.Size),
				"files": rawStats.FileCount,
			},
			"normalized": map[string]interface{}{
				"size":  formatBytes(normalizedStats.Size),
				"files": normalizedStats.FileCount,
			},
			"graph": map[string]interface{}{
				"size":  formatBytes(graphStats.Size),
				"files": graphStats.FileCount,
			},
			"annotations": map[string]interface{}{
				"size":  formatBytes(annotationsStats.Size),
				"files": annotationsStats.FileCount,
			},
		},
		"message_counts": messageCounts,
	}

	if !earliestMsg.IsZero() && !latestMsg.IsZero() {
		output["date_range"] = map[string]string{
			"earliest": earliestMsg.Format(time.RFC3339),
			"latest":   latestMsg.Format(time.RFC3339),
		}
	}

	return OutputJSON(output)
}

type dirStats struct {
	Size      int64
	FileCount int
}

func calculateDirStats(dir string) dirStats {
	var stats dirStats

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			stats.Size += info.Size()
			stats.FileCount++
		}
		return nil
	})

	return stats
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
