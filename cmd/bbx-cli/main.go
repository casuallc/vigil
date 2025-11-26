package main

import (
	"fmt"
	"github.com/casuallc/vigil/cli"
	"os"
)

func main() {
	// Create CLI with default API host
	cli := cli.NewCLI("http://localhost:8080")

	// Execute CLI commands
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
