package testutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func NewServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	root := repoRoot()

	mux.HandleFunc("GET /api/catalog/entities", func(w http.ResponseWriter, r *http.Request) {
		if after := r.URL.Query().Get("after"); after != "" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
			return
		}
		if limit := r.URL.Query().Get("limit"); limit != "" {
			w.Header().Set("Link", `</api/catalog/entities?after=eyJsaW1pdCI6MSwib2Zmc2V0IjoxfQ>; rel="next"`)
		}
		serveFixture(t, w, filepath.Join(root, "testdata/catalog/entities_list.json"), "application/json")
	})

	mux.HandleFunc("GET /api/catalog/entities/by-name/{kind}/{ns}/{name}/ancestry", func(w http.ResponseWriter, r *http.Request) {
		serveFixture(t, w, filepath.Join(root, "testdata/catalog/ancestry.json"), "application/json")
	})

	mux.HandleFunc("GET /api/catalog/entities/by-name/{kind}/{ns}/{name}", func(w http.ResponseWriter, r *http.Request) {
		kind := strings.ToLower(r.PathValue("kind"))
		name := r.PathValue("name")
		file := resolveEntityFixture(kind, name)
		serveFixture(t, w, filepath.Join(root, "testdata/catalog", file), "application/json")
	})

	mux.HandleFunc("GET /api/catalog/entity-facets", func(w http.ResponseWriter, r *http.Request) {
		serveFixture(t, w, filepath.Join(root, "testdata/catalog/facets_kind.json"), "application/json")
	})

	mux.HandleFunc("GET /api/search/query", func(w http.ResponseWriter, r *http.Request) {
		term := r.URL.Query().Get("term")
		file := "testdata/search/query_components.json"
		if term == "empty" {
			file = "testdata/search/query_empty.json"
		}
		serveFixture(t, w, filepath.Join(root, file), "application/json")
	})

	mux.HandleFunc("GET /api/techdocs/static/docs/{rest...}", func(w http.ResponseWriter, r *http.Request) {
		rest := r.PathValue("rest")
		if strings.HasSuffix(rest, "search/search_index.json") {
			serveFixture(t, w, filepath.Join(root, "testdata/techdocs/search_index.json"), "application/json")
			return
		}
		serveFixture(t, w, filepath.Join(root, "testdata/techdocs/page_content.html"), "text/html")
	})

	mux.HandleFunc("GET /api/techdocs/metadata/techdocs/{ns}/{kind}/{name}", func(w http.ResponseWriter, r *http.Request) {
		serveFixture(t, w, filepath.Join(root, "testdata/techdocs/metadata_techdocs.json"), "application/json")
	})

	mux.HandleFunc("POST /api/catalog/validate-entity", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"valid":true,"errors":[]}`))
	})

	mux.HandleFunc("POST /api/catalog/refresh", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := authMiddleware(t, root, mux)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func authMiddleware(t *testing.T, root string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-No-Auth") != "" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			data, err := os.ReadFile(filepath.Join(root, "testdata/errors/401_unauthorized.json"))
			if err != nil {
				t.Fatalf("testutil: read 401 fixture: %v", err)
			}
			w.Write(data)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func resolveEntityFixture(kind, name string) string {
	switch {
	case kind == "component" && name == "payment-service":
		return "entity_component.json"
	case kind == "api" && name == "payment-api":
		return "entity_api.json"
	case kind == "resource" && name == "payments-db":
		return "entity_resource.json"
	default:
		return "entity_component.json"
	}
}

func serveFixture(t *testing.T, w http.ResponseWriter, path, contentType string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"name":"NotFoundError","message":"Entity not found"}}`))
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

func repoRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}
