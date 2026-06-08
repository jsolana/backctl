package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jsolana/backctl/internal/backstage"
	"github.com/jsolana/backctl/internal/entityref"
	"github.com/jsolana/backctl/internal/output"
	"github.com/jsolana/backctl/internal/resolver"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	defaultListEntitiesLimit = 50
	maxListEntitiesLimit     = 200

	defaultRelationshipDepth = 3
	maxRelationshipDepth     = 5
)

type handlers struct {
	catalog  *backstage.CatalogService
	search   *backstage.SearchService
	techdocs *backstage.TechDocsService
	resolver *resolver.Resolver
}

func (h *handlers) handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	opts := backstage.SearchOptions{Term: query}

	if t, err := req.RequireString("type"); err == nil && t != "" {
		opts.Types = []string{t}
	}

	if limit, ok := req.GetArguments()["limit"].(float64); ok && limit > 0 {
		opts.Limit = int(limit)
	}

	result, err := h.search.Query(ctx, opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (h *handlers) handleGetEntity(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawRef, err := req.RequireString("ref")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ref, err := entityref.ParseStrict(rawRef, "default")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid entity ref: %v", err)), nil
	}

	entity, err := h.catalog.GetEntityByName(ctx, ref.Kind, ref.Namespace, ref.Name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get entity failed: %v", err)), nil
	}

	data, err := json.Marshal(entity)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (h *handlers) handleGetRelationships(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawRef, err := req.RequireString("ref")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ref, err := entityref.ParseStrict(rawRef, "default")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid entity ref: %v", err)), nil
	}

	opts := resolver.Options{
		Depth:     defaultRelationshipDepth,
		Direction: "outbound",
	}

	if depth, ok := req.GetArguments()["depth"].(float64); ok && depth > 0 {
		opts.Depth = int(depth)
		if opts.Depth > maxRelationshipDepth {
			opts.Depth = maxRelationshipDepth
		}
	}

	if dir, dirErr := req.RequireString("direction"); dirErr == nil && dir != "" {
		switch strings.ToLower(dir) {
		case "outbound", "inbound", "both":
			opts.Direction = strings.ToLower(dir)
		}
	}

	if relType, rtErr := req.RequireString("relation_type"); rtErr == nil && relType != "" {
		opts.Types = []string{relType}
	}

	tree, err := h.resolver.Resolve(ctx, ref.Kind, ref.Namespace, ref.Name, opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolve relationships failed: %v", err)), nil
	}

	data, err := json.Marshal(output.TreeToJSON(tree))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (h *handlers) handleGetTechDocsPage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawRef, err := req.RequireString("ref")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ref, err := entityref.ParseStrict(rawRef, "default")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid entity ref: %v", err)), nil
	}

	pagePath := ""
	if p, pErr := req.RequireString("path"); pErr == nil {
		pagePath = p
	}

	htmlContent, err := h.techdocs.GetPage(ctx, ref.Namespace, ref.Kind, ref.Name, pagePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get techdocs page failed: %v", err)), nil
	}

	text := output.ExtractText(htmlContent)
	return mcp.NewToolResultText(text), nil
}

func (h *handlers) handleListEntities(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := backstage.ListEntitiesOptions{
		Limit: defaultListEntitiesLimit,
	}

	var filters []string

	if kind, err := req.RequireString("kind"); err == nil && kind != "" {
		filters = append(filters, "kind="+kind)
	}

	if filter, err := req.RequireString("filter"); err == nil && filter != "" {
		filters = append(filters, filter)
	}

	opts.Filters = filters

	if limit, ok := req.GetArguments()["limit"].(float64); ok && limit > 0 {
		l := int(limit)
		if l > maxListEntitiesLimit {
			l = maxListEntitiesLimit
		}
		opts.Limit = l
	}

	if cursor, err := req.RequireString("cursor"); err == nil && cursor != "" {
		opts.After = cursor
	}

	result, err := h.catalog.ListEntities(ctx, opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list entities failed: %v", err)), nil
	}

	type entitySummary struct {
		Ref         string `json:"ref"`
		Kind        string `json:"kind"`
		Name        string `json:"name"`
		Namespace   string `json:"namespace"`
		Description string `json:"description,omitempty"`
		Owner       string `json:"owner,omitempty"`
		Lifecycle   string `json:"lifecycle,omitempty"`
		Type        string `json:"type,omitempty"`
	}

	summaries := make([]entitySummary, 0, len(result.Entities))
	for _, e := range result.Entities {
		s := entitySummary{
			Ref:         fmt.Sprintf("%s:%s/%s", e.Kind, e.Metadata.Namespace, e.Metadata.Name),
			Kind:        e.Kind,
			Name:        e.Metadata.Name,
			Namespace:   e.Metadata.Namespace,
			Description: e.Metadata.Description,
		}
		if owner, ok := e.Spec["owner"].(string); ok {
			s.Owner = owner
		}
		if lifecycle, ok := e.Spec["lifecycle"].(string); ok {
			s.Lifecycle = lifecycle
		}
		if t, ok := e.Spec["type"].(string); ok {
			s.Type = t
		}
		summaries = append(summaries, s)
	}

	resp := struct {
		Entities   []entitySummary `json:"entities"`
		Count      int             `json:"count"`
		NextCursor string          `json:"nextCursor,omitempty"`
	}{
		Entities:   summaries,
		Count:      len(summaries),
		NextCursor: result.NextCursor,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (h *handlers) handleListTechDocsPages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawRef, err := req.RequireString("ref")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	ref, err := entityref.ParseStrict(rawRef, "default")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid entity ref: %v", err)), nil
	}

	pages, err := h.techdocs.ListPages(ctx, ref.Namespace, ref.Kind, ref.Name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list techdocs pages failed: %v", err)), nil
	}

	data, err := json.Marshal(pages)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

var allowedCommands = map[string]bool{
	"catalog":   true,
	"search":    true,
	"techdocs":  true,
	"relations": true,
	"locations": true,
	"version":   true,
}

func (h *handlers) handleExecute(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command, err := req.RequireString("command")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := strings.Fields(command)
	if len(args) == 0 {
		return mcp.NewToolResultError("command cannot be empty"), nil
	}

	if !allowedCommands[args[0]] {
		return mcp.NewToolResultError(fmt.Sprintf(
			"command %q is not allowed; available: catalog, search, techdocs, relations, locations, version",
			args[0],
		)), nil
	}

	for _, arg := range args[1:] {
		if strings.ContainsAny(arg, ";|&$`\\\"'{}()<>!") {
			return mcp.NewToolResultError("invalid characters in command arguments"), nil
		}
	}

	args = append(args, "-o", "json")

	cmd := exec.CommandContext(ctx, "backctl", args...)
	cmd.Env = []string{}
	if envURL := getBackstageEnv("BACKSTAGE_URL"); envURL != "" {
		cmd.Env = append(cmd.Env, "BACKSTAGE_URL="+envURL)
	}
	if envToken := getBackstageEnv("BACKSTAGE_TOKEN"); envToken != "" {
		cmd.Env = append(cmd.Env, "BACKSTAGE_TOKEN="+envToken)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return mcp.NewToolResultError(fmt.Sprintf("command failed: %s", strings.TrimSpace(errMsg))), nil
	}

	return mcp.NewToolResultText(stdout.String()), nil
}

func getBackstageEnv(key string) string {
	return os.Getenv(key)
}
