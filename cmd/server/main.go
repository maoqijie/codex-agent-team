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
	skipCheck := flag.Bool("skip-check", false, "Skip codex binary check")
	flag.Parse()

	// Validate codex binary (unless skipped)
	if !*skipCheck {
		if _, err := os.Stat(*codexBin); os.IsNotExist(err) {
			log.Printf("Warning: Codex binary not found at: %s", *codexBin)
			log.Printf("Server will start but agent operations will fail.")
			log.Printf("Use -skip-check to suppress this warning, or provide -codex <path>")
		}
	}

	// Create server
	server := api.NewServer(*codexBin, *repoPath)

	// Start server
	log.Printf("Starting Codex Agent Team server on %s", *addr)
	log.Printf("Codex binary: %s", *codexBin)
	log.Printf("Repository: %s", *repoPath)
	log.Printf("Visit http://localhost%s", *addr)

	if err := server.Start(*addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
