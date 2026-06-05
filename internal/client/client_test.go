package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_GetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/test" {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"test"}`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ Name string }
	err = c.GetJSON(context.Background(), "/api/test", nil, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "test" {
		t.Errorf("got %q, want %q", result.Name, "test")
	}
}

func TestClient_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	var result any
	err = c.GetJSON(context.Background(), "/api/missing", nil, &result)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("got status %d, want 404", apiErr.StatusCode)
	}
	if !apiErr.IsNotFound() {
		t.Error("IsNotFound should be true")
	}
}

func TestClient_BearerAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Token: "my-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	var result any
	c.GetJSON(context.Background(), "/api/test", nil, &result)

	if gotAuth != "Bearer my-token" {
		t.Errorf("got auth %q, want %q", gotAuth, "Bearer my-token")
	}
}

func TestClient_Retry(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 10 * time.Second, MaxRetries: 3})
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ OK bool }
	err = c.GetJSON(context.Background(), "/api/test", nil, &result)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if !result.OK {
		t.Error("expected OK=true")
	}
	if attempts < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempts)
	}
}

func TestClient_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 100 * time.Millisecond, MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}

	var result any
	err = c.GetJSON(context.Background(), "/api/slow", nil, &result)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
