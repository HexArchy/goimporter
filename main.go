package main

import (
	"fmt"
	"os"

	"goimporter/goimporter"
)

func main() {
	cfg := goimporter.ParseFlags()

	if err := goimporter.ProcessGoFiles(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
