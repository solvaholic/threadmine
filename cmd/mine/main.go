package main

import (
	"os"

	"github.com/solvaholic/threadmine/cmd/mine/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		commands.OutputError("%v", err)
		os.Exit(1)
	}
}
