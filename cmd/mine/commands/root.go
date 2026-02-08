package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	outputFormat string
	dbPath       string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mine",
	Short: "Search and analyze conversations across platforms",
	Long: `ThreadMine (mine) searches and analyzes conversations from Slack, GitHub, and email.

The tool has two main modes:
  - fetch: Search and retrieve messages from upstream sources (Slack, GitHub, etc.)
  - select: Query and analyze locally cached messages

All data is stored in a local SQLite database for fast querying and analysis.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "json", "Output format (json, jsonl, table)")
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database path (default: ~/.threadmine/threadmine.db)")
}

// OutputJSON writes JSON to stdout with optional pretty printing
func OutputJSON(data interface{}) error {
	var output []byte
	var err error

	if outputFormat == "json" {
		output, err = json.MarshalIndent(data, "", "  ")
	} else {
		output, err = json.Marshal(data)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

// OutputError writes error message to stderr
func OutputError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
