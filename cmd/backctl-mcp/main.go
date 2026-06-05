package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jsolana/backctl/internal/client"
	mcpserver "github.com/jsolana/backctl/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	baseURL := os.Getenv("BACKSTAGE_URL")
	if baseURL == "" {
		log.Fatal("BACKSTAGE_URL environment variable is required")
	}

	token := os.Getenv("BACKSTAGE_TOKEN")

	timeout := 30 * time.Second
	if t := os.Getenv("BACKSTAGE_TIMEOUT"); t != "" {
		d, err := time.ParseDuration(t)
		if err == nil {
			timeout = d
		}
	}

	httpClient, err := client.New(client.Config{
		BaseURL:   baseURL,
		Token:     token,
		Timeout:   timeout,
		UserAgent: fmt.Sprintf("backctl-mcp/%s", version),
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP client: %v", err)
	}

	s := mcpserver.NewServer(httpClient, version)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
