package main

import (
	"fmt"
	"os"

	"claude-harness/tools/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "as-skill: error:", err)
		os.Exit(1)
	}
}
