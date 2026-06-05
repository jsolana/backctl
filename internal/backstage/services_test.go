package backstage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jsolana/backctl/internal/client"
)

func TestCatalogService_ListEntities_CursorPagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		after := r.URL.Query().Get("after")

		if after == "page2cursor" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"apiVersion":"backstage.io/v1alpha1","kind":"Component","metadata":{"name":"svc-c","namespace":"default"},"spec":{}}]`))
			return
		}

		w.Header().Set("Link", `</api/catalog/entities?after=page2cursor>; rel="next"`)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"apiVersion":"backstage.io/v1alpha1","kind":"Component","metadata":{"name":"svc-a","namespace":"default"},"spec":{}},{"apiVersion":"backstage.io/v1alpha1","kind":"Component","metadata":{"name":"svc-b","namespace":"default"},"spec":{}}]`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewCatalogService(c)

	// First page
	result, err := svc.ListEntities(context.Background(), ListEntitiesOptions{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(result.Entities))
	}
	if result.NextCursor != "page2cursor" {
		t.Errorf("expected next cursor 'page2cursor', got %q", result.NextCursor)
	}

	// Second page using cursor
	result, err = svc.ListEntities(context.Background(), ListEntitiesOptions{Limit: 2, After: result.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entities) != 1 {
		t.Fatalf("expected 1 entity on second page, got %d", len(result.Entities))
	}
	if result.NextCursor != "" {
		t.Errorf("expected empty next cursor on last page, got %q", result.NextCursor)
	}
	if result.Entities[0].Metadata.Name != "svc-c" {
		t.Errorf("expected svc-c, got %q", result.Entities[0].Metadata.Name)
	}
}

func TestCatalogService_ListEntities_NoLinkHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"apiVersion":"backstage.io/v1alpha1","kind":"Component","metadata":{"name":"only-one","namespace":"default"},"spec":{}}]`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewCatalogService(c)

	result, err := svc.ListEntities(context.Background(), ListEntitiesOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.NextCursor != "" {
		t.Errorf("expected empty cursor, got %q", result.NextCursor)
	}
	if len(result.Entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(result.Entities))
	}
}

func TestTechDocsService_ListPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"config": {"lang": ["en"]},
			"docs": [
				{"location": "", "title": "Home", "text": "Welcome"},
				{"location": "getting-started/", "title": "Getting Started", "text": "Setup guide"},
				{"location": "guides/integration/", "title": "Integration", "text": "How to integrate"},
				{"location": "getting-started/", "title": "Getting Started", "text": "Duplicate entry"}
			]
		}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewTechDocsService(c)

	pages, err := svc.ListPages(context.Background(), "default", "component", "my-service")
	if err != nil {
		t.Fatal(err)
	}

	if len(pages) != 3 {
		t.Fatalf("expected 3 unique pages, got %d", len(pages))
	}

	expected := []struct {
		location string
		title    string
	}{
		{"/", "Home"},
		{"getting-started/", "Getting Started"},
		{"guides/integration/", "Integration"},
	}

	for i, exp := range expected {
		if pages[i].Location != exp.location {
			t.Errorf("page[%d].Location = %q, want %q", i, pages[i].Location, exp.location)
		}
		if pages[i].Title != exp.title {
			t.Errorf("page[%d].Title = %q, want %q", i, pages[i].Title, exp.title)
		}
	}
}

func TestTechDocsService_ListPages_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewTechDocsService(c)

	_, err = svc.ListPages(context.Background(), "default", "component", "no-docs")
	if err == nil {
		t.Fatal("expected error for entity without TechDocs")
	}
}

func TestCatalogService_GetAncestry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"rootEntityRef": "component:default/my-service",
			"items": [
				{
					"entity": {
						"apiVersion": "backstage.io/v1alpha1",
						"kind": "Component",
						"metadata": {"name": "my-service", "namespace": "default"},
						"spec": {}
					},
					"parentEntityRefs": ["location:default/generated-123"]
				},
				{
					"entity": {
						"apiVersion": "backstage.io/v1alpha1",
						"kind": "Location",
						"metadata": {"name": "generated-123", "namespace": "default"},
						"spec": {"type": "url", "target": "https://github.com/org/repo/catalog-info.yaml"}
					},
					"parentEntityRefs": ["location:default/root-location"]
				},
				{
					"entity": {
						"apiVersion": "backstage.io/v1alpha1",
						"kind": "Location",
						"metadata": {"name": "root-location", "namespace": "default"},
						"spec": {"type": "url", "target": "https://github.com/org/repo"}
					},
					"parentEntityRefs": []
				}
			]
		}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewCatalogService(c)

	result, err := svc.GetAncestry(context.Background(), "component", "default", "my-service")
	if err != nil {
		t.Fatal(err)
	}

	if result.RootEntityRef != "component:default/my-service" {
		t.Errorf("rootEntityRef = %q", result.RootEntityRef)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 ancestry items, got %d", len(result.Items))
	}

	// First item is the entity itself
	if result.Items[0].Entity.Kind != "Component" {
		t.Errorf("items[0].kind = %q", result.Items[0].Entity.Kind)
	}
	if len(result.Items[0].ParentEntityRefs) != 1 {
		t.Errorf("items[0] should have 1 parent, got %d", len(result.Items[0].ParentEntityRefs))
	}

	// Last item is the root (no parents)
	if len(result.Items[2].ParentEntityRefs) != 0 {
		t.Errorf("root item should have no parents, got %d", len(result.Items[2].ParentEntityRefs))
	}
	if result.Items[2].Entity.Kind != "Location" {
		t.Errorf("root item kind = %q", result.Items[2].Entity.Kind)
	}
}

func TestCatalogService_GetAncestry_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"name":"NotFoundError","message":"Entity not found"}}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewCatalogService(c)

	_, err = svc.GetAncestry(context.Background(), "component", "default", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent entity")
	}
}

func TestCatalogService_ValidateEntity_Valid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/catalog/validate-entity" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"valid":true,"errors":[]}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewCatalogService(c)

	entity := Entity{
		APIVersion: "backstage.io/v1alpha1",
		Kind:       "Component",
		Metadata:   EntityMeta{Name: "test-service", Namespace: "default"},
		Spec:       map[string]any{"type": "service", "lifecycle": "production", "owner": "group:default/team-a"},
	}

	result, err := svc.ValidateEntity(context.Background(), entity, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Errorf("expected valid=true, got false with errors: %v", result.Errors)
	}
}

func TestCatalogService_ValidateEntity_Invalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"valid":false,"errors":["spec.owner must be a valid entity reference","metadata.name contains invalid characters"]}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewCatalogService(c)

	entity := Entity{
		APIVersion: "backstage.io/v1alpha1",
		Kind:       "Component",
		Metadata:   EntityMeta{Name: "bad service!!", Namespace: "default"},
		Spec:       map[string]any{"owner": "not-a-ref"},
	}

	result, err := svc.ValidateEntity(context.Background(), entity, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid {
		t.Error("expected valid=false")
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}

func TestCatalogService_ValidateEntity_WithLocation(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"valid":true,"errors":[]}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewCatalogService(c)

	entity := Entity{
		APIVersion: "backstage.io/v1alpha1",
		Kind:       "Component",
		Metadata:   EntityMeta{Name: "svc", Namespace: "default"},
		Spec:       map[string]any{},
	}

	_, err = svc.ValidateEntity(context.Background(), entity, "url:https://github.com/org/repo/blob/main/catalog-info.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(gotBody), "catalog-info.yaml") {
		t.Errorf("expected location in request body, got: %s", gotBody)
	}
}

func TestCatalogService_RefreshEntity(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/catalog/refresh" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewCatalogService(c)

	err = svc.RefreshEntity(context.Background(), "component:default/my-service")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(gotBody), "component:default/my-service") {
		t.Errorf("expected entityRef in body, got: %s", gotBody)
	}
}

func TestCatalogService_RefreshEntity_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"name":"NotFoundError","message":"Entity not found"}}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	svc := NewCatalogService(c)

	err = svc.RefreshEntity(context.Background(), "component:default/nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent entity")
	}
}
