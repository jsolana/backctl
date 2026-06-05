package backstage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/jsolana/backctl/internal/client"
)

type CatalogService struct {
	client *client.Client
}

func NewCatalogService(c *client.Client) *CatalogService {
	return &CatalogService{client: c}
}

type ListEntitiesOptions struct {
	Filters []string
	Fields  []string
	Order   []string
	Limit   int
	Offset  int
	After   string
}

type ListEntitiesResponse struct {
	Entities   []Entity `json:"entities"`
	NextCursor string   `json:"nextCursor,omitempty"`
}

func (s *CatalogService) ListEntities(ctx context.Context, opts ListEntitiesOptions) (*ListEntitiesResponse, error) {
	q := url.Values{}
	for _, f := range opts.Filters {
		q.Add("filter", f)
	}
	for _, f := range opts.Fields {
		q.Add("fields", f)
	}
	for _, o := range opts.Order {
		q.Add("order", o)
	}
	if opts.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.After != "" {
		q.Set("after", opts.After)
	} else if opts.Offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", opts.Offset))
	}

	var entities []Entity
	resp, err := s.client.GetJSONWithHeaders(ctx, "/api/catalog/entities", q, &entities)
	if err != nil {
		return nil, err
	}

	result := &ListEntitiesResponse{Entities: entities}
	if link := resp.Header.Get("Link"); link != "" {
		result.NextCursor = client.ParseLinkNext(link)
	}
	return result, err
}

func (s *CatalogService) GetEntityByName(ctx context.Context, kind, namespace, name string) (*Entity, error) {
	path := fmt.Sprintf("/api/catalog/entities/by-name/%s/%s/%s",
		strings.ToLower(kind), namespace, name)
	var entity Entity
	err := s.client.GetJSON(ctx, path, nil, &entity)
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (s *CatalogService) GetEntityByUID(ctx context.Context, uid string) (*Entity, error) {
	path := fmt.Sprintf("/api/catalog/entities/by-uid/%s", uid)
	var entity Entity
	err := s.client.GetJSON(ctx, path, nil, &entity)
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (s *CatalogService) GetFacets(ctx context.Context, facet string) (*FacetsResponse, error) {
	q := url.Values{"facet": {facet}}
	var result FacetsResponse
	err := s.client.GetJSON(ctx, "/api/catalog/entity-facets", q, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *CatalogService) GetAncestry(ctx context.Context, kind, namespace, name string) (*AncestryResponse, error) {
	path := fmt.Sprintf("/api/catalog/entities/by-name/%s/%s/%s/ancestry",
		strings.ToLower(kind), namespace, name)
	var result AncestryResponse
	err := s.client.GetJSON(ctx, path, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *CatalogService) ValidateEntity(ctx context.Context, entity Entity, location string) (*ValidateEntityResponse, error) {
	payload, err := json.Marshal(ValidateEntityRequest{
		Entity:   entity,
		Location: location,
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling validate request: %w", err)
	}
	resp, err := s.client.Post(ctx, "/api/catalog/validate-entity", strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ValidateEntityResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *CatalogService) RefreshEntity(ctx context.Context, entityRef string) error {
	payload, err := json.Marshal(struct {
		EntityRef string `json:"entityRef"`
	}{EntityRef: entityRef})
	if err != nil {
		return fmt.Errorf("marshaling refresh request: %w", err)
	}
	resp, err := s.client.Post(ctx, "/api/catalog/refresh", strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
