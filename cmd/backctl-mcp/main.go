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

	switch os.Getenv("MCP_TRANSPORT") {
	case "http":
		addr := os.Getenv("MCP_HTTP_ADDR")
		if addr == "" {
			addr = ":8080"
		}
		httpServer := server.NewStreamableHTTPServer(s,
			server.WithEndpointPath("/mcp"),
			server.WithStateLess(true),
		)
		log.Printf("backctl-mcp listening on %s/mcp", addr)
		if err := httpServer.Start(addr); err != nil {
			log.Fatalf("server error: %v", err)
		}
	default:
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}
}
