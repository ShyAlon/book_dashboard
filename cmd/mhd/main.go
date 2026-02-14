package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"book_dashboard/internal/workspace"
)

func main() {
	root, err := workspace.EnsureDefault()
	if err != nil {
		log.Fatalf("workspace initialization failed: %v", err)
	}

	fmt.Printf("Manuscript Health workspace ready at: %s\n", filepath.Clean(root))
	fmt.Printf("Home: %s\n", os.Getenv("HOME"))
}
