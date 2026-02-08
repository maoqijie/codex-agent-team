package main

import (
	"flag"
	"log"
	"os"

	"codex-agent-team/internal/api"
)

func main() {
	// Command line flags
	addr := flag.String("addr", ":8080", "HTTP server address")
	codexBin := flag.String("codex", "codex2", "Path to codex app-server binary")
	repoPath := flag.String("repo", ".", "Path to the repository to work on")
	flag.Parse()

	// Validate codex binary
	if _, err := os.Stat(*codexBin); os.IsNotExist(err) {
		log.Fatalf("Codex binary not found at: %s", *codexBin)
	}

	// Create server
	server := api.NewServer(*codexBin, *repoPath)

	// Start server
	log.Printf("Starting Codex Agent Team server on %s", *addr)
	log.Printf("Codex binary: %s", *codexBin)
	log.Printf("Repository: %s", *repoPath)

	if err := server.Start(*addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
