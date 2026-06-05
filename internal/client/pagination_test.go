package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetJSONWithHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `</api/catalog/entities?after=abc123>; rel="next"`)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name":"test"}]`))
	}))
	defer srv.Close()

	c, err := New(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	var result []map[string]string
	resp, err := c.GetJSONWithHeaders(context.Background(), "/api/catalog/entities", nil, &result)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Header.Get("Link") == "" {
		t.Error("expected Link header in response")
	}
	if len(result) != 1 || result[0]["name"] != "test" {
		t.Errorf("unexpected body: %v", result)
	}
}

func TestParseLinkNext(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "standard next link",
			header: `</api/catalog/entities?after=eyJsaW1pdCI6MSwib2Zmc2V0IjoxfQ>; rel="next"`,
			want:   "eyJsaW1pdCI6MSwib2Zmc2V0IjoxfQ",
		},
		{
			name:   "multiple links",
			header: `</api/catalog/entities?after=cursor123>; rel="next", </api/catalog/entities?before=cursor000>; rel="prev"`,
			want:   "cursor123",
		},
		{
			name:   "no next link",
			header: `</api/catalog/entities?before=cursor000>; rel="prev"`,
			want:   "",
		},
		{
			name:   "empty header",
			header: "",
			want:   "",
		},
		{
			name:   "full url in link",
			header: `<https://backstage.example.com/api/catalog/entities?after=full_url_cursor&limit=20>; rel="next"`,
			want:   "full_url_cursor",
		},
		{
			name:   "no after param",
			header: `</api/catalog/entities?offset=10>; rel="next"`,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLinkNext(tt.header)
			if got != tt.want {
				t.Errorf("ParseLinkNext(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}
