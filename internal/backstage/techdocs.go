package backstage

import (
	"context"
	"fmt"
	"strings"

	"github.com/jsolana/backctl/internal/client"
)

type TechDocsService struct {
	client *client.Client
}

func NewTechDocsService(c *client.Client) *TechDocsService {
	return &TechDocsService{client: c}
}

func (s *TechDocsService) GetPage(ctx context.Context, namespace, kind, name, pagePath string) ([]byte, error) {
	if pagePath == "" {
		pagePath = "index.html"
	}
	if !strings.HasSuffix(pagePath, ".html") && !strings.HasSuffix(pagePath, "/") {
		pagePath += "/index.html"
	}
	path := fmt.Sprintf("/api/techdocs/static/docs/%s/%s/%s/%s",
		namespace, strings.ToLower(kind), name, pagePath)
	return s.client.GetRaw(ctx, path, nil)
}

func (s *TechDocsService) GetMetadata(ctx context.Context, namespace, kind, name string) (*TechDocsMetadata, error) {
	path := fmt.Sprintf("/api/techdocs/metadata/techdocs/%s/%s/%s",
		namespace, strings.ToLower(kind), name)
	var meta TechDocsMetadata
	err := s.client.GetJSON(ctx, path, nil, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *TechDocsService) GetEntityMetadata(ctx context.Context, namespace, kind, name string) (*TechDocsEntityMetadata, error) {
	path := fmt.Sprintf("/api/techdocs/metadata/entity/%s/%s/%s",
		namespace, strings.ToLower(kind), name)
	var meta TechDocsEntityMetadata
	err := s.client.GetJSON(ctx, path, nil, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *TechDocsService) ListPages(ctx context.Context, namespace, kind, name string) ([]TechDocsPage, error) {
	path := fmt.Sprintf("/api/techdocs/static/docs/%s/%s/%s/search/search_index.json",
		namespace, strings.ToLower(kind), name)

	var index struct {
		Docs []struct {
			Location string `json:"location"`
			Title    string `json:"title"`
		} `json:"docs"`
	}
	err := s.client.GetJSON(ctx, path, nil, &index)
	if err != nil {
		return nil, fmt.Errorf("listing pages (search_index.json): %w", err)
	}

	seen := make(map[string]bool)
	var pages []TechDocsPage
	for _, doc := range index.Docs {
		loc := doc.Location
		if loc == "" {
			loc = "/"
		}
		if seen[loc] {
			continue
		}
		seen[loc] = true
		pages = append(pages, TechDocsPage{
			Location: loc,
			Title:    doc.Title,
		})
	}
	return pages, nil
}
