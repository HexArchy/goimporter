package main

import (
	"fmt"
	"goimporter/config"
	"goimporter/formatter"
	"os"
)

func main() {
	// Parse command-line flags.
	cfg := config.ParseFlags()

	// Process Go files according to the configuration.
	err := formatter.ProcessGoFiles(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
