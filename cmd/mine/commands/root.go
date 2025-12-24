package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global format flag
	outputFormat string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mine",
	Short: "Extract and analyze multi-platform conversations",
	Long: `ThreadMine (mine) is a CLI tool for extracting, caching, and analyzing 
conversations across Slack, GitHub, and email.

It provides a unified interface for working with messages from multiple 
communication platforms, normalizing heterogeneous data sources into a 
common schema for cross-platform analysis and graph-based insights.`,
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
