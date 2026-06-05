package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jsolana/backctl/internal/backstage"
	"github.com/jsolana/backctl/internal/client"
)

func TestResolver_Resolve_EnrichedNodes(t *testing.T) {
	entities := map[string]string{
		"/api/catalog/entities/by-name/component/default/my-service": `{
			"apiVersion": "backstage.io/v1alpha1",
			"kind": "Component",
			"metadata": {
				"name": "my-service",
				"namespace": "default",
				"labels": {"tier": "critical"}
			},
			"spec": {
				"type": "service",
				"lifecycle": "production",
				"owner": "group:default/team-payments"
			},
			"relations": [
				{"type": "dependsOn", "targetRef": "resource:default/my-db"},
				{"type": "providesApi", "targetRef": "api:default/my-api"}
			]
		}`,
		"/api/catalog/entities/by-name/resource/default/my-db": `{
			"apiVersion": "backstage.io/v1alpha1",
			"kind": "Resource",
			"metadata": {
				"name": "my-db",
				"namespace": "default",
				"labels": {"tier": "critical"}
			},
			"spec": {
				"type": "database",
				"owner": "group:default/team-infra"
			},
			"relations": []
		}`,
		"/api/catalog/entities/by-name/api/default/my-api": `{
			"apiVersion": "backstage.io/v1alpha1",
			"kind": "API",
			"metadata": {
				"name": "my-api",
				"namespace": "default",
				"labels": {}
			},
			"spec": {
				"type": "openapi",
				"lifecycle": "production",
				"owner": "group:default/team-payments"
			},
			"relations": []
		}`,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := entities[r.URL.Path]
		if !ok {
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"not found"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	catalog := backstage.NewCatalogService(c)
	r := New(catalog)

	tree, err := r.Resolve(context.Background(), "component", "default", "my-service", Options{
		Depth:     2,
		Direction: "outbound",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Root node checks
	if tree.Ref != "component:default/my-service" {
		t.Errorf("root ref = %q", tree.Ref)
	}
	if tree.Kind != "Component" {
		t.Errorf("root kind = %q", tree.Kind)
	}
	if tree.Owner != "group:default/team-payments" {
		t.Errorf("root owner = %q", tree.Owner)
	}
	if tree.Lifecycle != "production" {
		t.Errorf("root lifecycle = %q", tree.Lifecycle)
	}
	if tree.Tier != "critical" {
		t.Errorf("root tier = %q", tree.Tier)
	}

	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(tree.Children))
	}

	// Find the db child and api child (order is non-deterministic due to goroutines)
	var dbNode, apiNode *struct {
		ref, relType, kind, owner, lifecycle, tier string
	}

	for _, child := range tree.Children {
		switch {
		case strings.Contains(child.Ref, "my-db"):
			dbNode = &struct{ ref, relType, kind, owner, lifecycle, tier string }{
				child.Ref, child.RelationType, child.Kind, child.Owner, child.Lifecycle, child.Tier,
			}
		case strings.Contains(child.Ref, "my-api"):
			apiNode = &struct{ ref, relType, kind, owner, lifecycle, tier string }{
				child.Ref, child.RelationType, child.Kind, child.Owner, child.Lifecycle, child.Tier,
			}
		}
	}

	if dbNode == nil {
		t.Fatal("db child not found")
	}
	if dbNode.relType != "dependsOn" {
		t.Errorf("db relationType = %q", dbNode.relType)
	}
	if dbNode.kind != "Resource" {
		t.Errorf("db kind = %q", dbNode.kind)
	}
	if dbNode.owner != "group:default/team-infra" {
		t.Errorf("db owner = %q", dbNode.owner)
	}
	if dbNode.tier != "critical" {
		t.Errorf("db tier = %q", dbNode.tier)
	}

	if apiNode == nil {
		t.Fatal("api child not found")
	}
	if apiNode.relType != "providesApi" {
		t.Errorf("api relationType = %q", apiNode.relType)
	}
	if apiNode.kind != "API" {
		t.Errorf("api kind = %q", apiNode.kind)
	}
	if apiNode.owner != "group:default/team-payments" {
		t.Errorf("api owner = %q", apiNode.owner)
	}
	if apiNode.lifecycle != "production" {
		t.Errorf("api lifecycle = %q", apiNode.lifecycle)
	}
}

func TestResolver_Resolve_UnresolvedNode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "my-service") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"apiVersion": "backstage.io/v1alpha1",
				"kind": "Component",
				"metadata": {"name": "my-service", "namespace": "default"},
				"spec": {"owner": "team-x"},
				"relations": [
					{"type": "dependsOn", "targetRef": "resource:default/missing-db"}
				]
			}`))
			return
		}
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	catalog := backstage.NewCatalogService(c)
	r := New(catalog)

	tree, err := r.Resolve(context.Background(), "component", "default", "my-service", Options{
		Depth:     2,
		Direction: "outbound",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(tree.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(tree.Children))
	}

	child := tree.Children[0]
	if !strings.Contains(child.Label, "unresolved") {
		t.Errorf("expected unresolved label, got %q", child.Label)
	}
	if child.Owner != "" {
		t.Errorf("unresolved node should have empty owner, got %q", child.Owner)
	}
}

func TestResolver_ResolveFlat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"apiVersion": "backstage.io/v1alpha1",
			"kind": "Component",
			"metadata": {"name": "my-service", "namespace": "default"},
			"spec": {},
			"relations": [
				{"type": "dependsOn", "targetRef": "resource:default/my-db"},
				{"type": "ownedBy", "targetRef": "group:default/team-a"},
				{"type": "ownerOf", "targetRef": "component:default/other"}
			]
		}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	catalog := backstage.NewCatalogService(c)
	r := New(catalog)

	relations, err := r.ResolveFlat(context.Background(), "component", "default", "my-service", Options{
		Direction: "outbound",
	})
	if err != nil {
		t.Fatal(err)
	}

	// outbound: dependsOn + ownedBy (ownerOf is inbound)
	if len(relations) != 2 {
		t.Fatalf("expected 2 outbound relations, got %d", len(relations))
	}
}

func TestFormatNodeLabel(t *testing.T) {
	tests := []struct {
		relType   string
		ref       string
		owner     string
		lifecycle string
		want      string
	}{
		{
			"dependsOn", "resource:default/my-db", "team-a", "production",
			"[dependsOn] resource:default/my-db (owner=team-a, lifecycle=production)",
		},
		{
			"consumesApi", "api:default/my-api", "team-b", "",
			"[consumesApi] api:default/my-api (owner=team-b)",
		},
		{
			"ownedBy", "group:default/team-a", "", "",
			"[ownedBy] group:default/team-a",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.relType, tt.ref), func(t *testing.T) {
			got := formatNodeLabel(tt.relType, tt.ref, tt.owner, tt.lifecycle)
			if got != tt.want {
				t.Errorf("formatNodeLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTreeToJSON_FullSerialization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "my-service") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"apiVersion": "backstage.io/v1alpha1",
				"kind": "Component",
				"metadata": {"name": "my-service", "namespace": "default", "labels": {"tier": "tier-1"}},
				"spec": {"lifecycle": "production", "owner": "group:default/platform"},
				"relations": [{"type": "dependsOn", "targetRef": "resource:default/cache"}]
			}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"apiVersion": "backstage.io/v1alpha1",
			"kind": "Resource",
			"metadata": {"name": "cache", "namespace": "default", "labels": {"tier": "tier-2"}},
			"spec": {"owner": "group:default/infra"},
			"relations": []
		}`))
	}))
	defer srv.Close()

	c, err := client.New(client.Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	catalog := backstage.NewCatalogService(c)
	r := New(catalog)

	tree, err := r.Resolve(context.Background(), "component", "default", "my-service", Options{
		Depth:     2,
		Direction: "outbound",
	})
	if err != nil {
		t.Fatal(err)
	}

	jsonData, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	// Verify JSON contains the enriched fields
	jsonStr := string(jsonData)
	for _, field := range []string{`"ref"`, `"kind"`, `"owner"`, `"lifecycle"`, `"tier"`} {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing field %s:\n%s", field, jsonStr)
		}
	}
}
