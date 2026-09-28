package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/mcpserver"
)

func main() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve executable path: %v\n", err)
		os.Exit(1)
	}
	dataDir := filepath.Dir(exe)

	if err := mcpserver.Run(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
		os.Exit(1)
	}
}
