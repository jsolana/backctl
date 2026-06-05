package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestETagCache_SecondRequestSends304(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		if etag := r.Header.Get("If-None-Match"); etag == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"cached-entity","version":1}`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	// First request: full response with ETag
	var result1 map[string]any
	err = c.GetJSON(context.Background(), "/api/test", nil, &result1)
	if err != nil {
		t.Fatal(err)
	}
	if result1["name"] != "cached-entity" {
		t.Errorf("first request: got %v", result1)
	}

	// Second request: should get 304 and serve from cache
	var result2 map[string]any
	err = c.GetJSON(context.Background(), "/api/test", nil, &result2)
	if err != nil {
		t.Fatal(err)
	}
	if result2["name"] != "cached-entity" {
		t.Errorf("second request (cached): got %v", result2)
	}

	if got := requestCount.Load(); got != 2 {
		t.Errorf("expected 2 server requests, got %d", got)
	}
}

func TestETagCache_UpdatedResourceInvalidatesCache(t *testing.T) {
	var version atomic.Int32
	version.Store(1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentVersion := version.Load()
		currentETag := `"v1"`
		if currentVersion == 2 {
			currentETag = `"v2"`
		}

		if etag := r.Header.Get("If-None-Match"); etag == currentETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("ETag", currentETag)
		w.Header().Set("Content-Type", "application/json")
		if currentVersion == 1 {
			w.Write([]byte(`{"value":"original"}`))
		} else {
			w.Write([]byte(`{"value":"updated"}`))
		}
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	// First request
	var r1 map[string]string
	if err := c.GetJSON(context.Background(), "/api/entity", nil, &r1); err != nil {
		t.Fatal(err)
	}
	if r1["value"] != "original" {
		t.Fatalf("expected 'original', got %q", r1["value"])
	}

	// Second request with same ETag → 304 → cached
	var r2 map[string]string
	if err := c.GetJSON(context.Background(), "/api/entity", nil, &r2); err != nil {
		t.Fatal(err)
	}
	if r2["value"] != "original" {
		t.Fatalf("expected cached 'original', got %q", r2["value"])
	}

	// Simulate resource update
	version.Store(2)

	// Third request: server returns new ETag → full response
	var r3 map[string]string
	if err := c.GetJSON(context.Background(), "/api/entity", nil, &r3); err != nil {
		t.Fatal(err)
	}
	if r3["value"] != "updated" {
		t.Fatalf("expected 'updated' after invalidation, got %q", r3["value"])
	}
}

func TestETagCache_DifferentURLsAreCachedSeparately(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"etag-`+r.URL.Path+`"`)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"path":"` + r.URL.Path + `"}`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	var r1, r2 map[string]string
	if err := c.GetJSON(context.Background(), "/api/a", nil, &r1); err != nil {
		t.Fatal(err)
	}
	if err := c.GetJSON(context.Background(), "/api/b", nil, &r2); err != nil {
		t.Fatal(err)
	}

	if r1["path"] != "/api/a" {
		t.Errorf("r1 path = %q", r1["path"])
	}
	if r2["path"] != "/api/b" {
		t.Errorf("r2 path = %q", r2["path"])
	}
}

func TestETagCache_NoETagHeader_NoCaching(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":` + string(rune('0'+requestCount.Load())) + `}`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	var r1, r2 map[string]any
	c.GetJSON(context.Background(), "/api/nocache", nil, &r1)
	c.GetJSON(context.Background(), "/api/nocache", nil, &r2)

	if got := requestCount.Load(); got != 2 {
		t.Errorf("expected 2 requests (no caching), got %d", got)
	}
}

func TestETagCache_GetRawAlsoCaches(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		if etag := r.Header.Get("If-None-Match"); etag == `"html-v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("ETag", `"html-v1"`)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<h1>Hello</h1>`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	body1, err := c.GetRaw(context.Background(), "/page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(body1) != "<h1>Hello</h1>" {
		t.Errorf("first GetRaw = %q", body1)
	}

	body2, err := c.GetRaw(context.Background(), "/page", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(body2) != "<h1>Hello</h1>" {
		t.Errorf("second GetRaw (cached) = %q", body2)
	}

	if got := requestCount.Load(); got != 2 {
		t.Errorf("expected 2 server hits, got %d", got)
	}
}
