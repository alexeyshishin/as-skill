package main

import (
	"fmt"
	"os"

	"claude-harness/tools/internal/cli"
	"claude-harness/tools/internal/iostreams"
)

func main() {
	if err := cli.Main(os.Args[1:], iostreams.Std()); err != nil {
		fmt.Fprintln(os.Stderr, "as-skill: error:", err)
		os.Exit(1)
	}
}
