package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Config struct {
	BaseURL    string
	Token      string
	Timeout    time.Duration
	MaxRetries int
	UserAgent  string
}

func (c *Config) defaults() {
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.UserAgent == "" {
		c.UserAgent = "backctl"
	}
}

// TODO: Replace sync.Map-based cache with a bounded LRU to prevent unbounded
// memory growth in long-running MCP server sessions.
type etagEntry struct {
	etag string
	body []byte
}

type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	userAgent  string
	cache      sync.Map // key: request URL string → value: *etagEntry
}

func New(cfg Config) (*Client, error) {
	cfg.defaults()

	base, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	transport := &http.Transport{
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	var rt http.RoundTripper = transport
	if cfg.Token != "" {
		rt = &bearerTransport{token: cfg.Token, base: rt}
	}
	rt = &retryTransport{base: rt, maxRetries: cfg.MaxRetries, baseDelay: time.Second}

	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout, Transport: rt},
		baseURL:    base,
		userAgent:  cfg.UserAgent,
	}, nil
}

func (c *Client) Get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, path, query, nil)
}

func (c *Client) Post(ctx context.Context, path string, body io.Reader) (*http.Response, error) {
	return c.do(ctx, http.MethodPost, path, nil, body)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	u := c.baseURL.JoinPath(path)
	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	cacheKey := req.URL.String()
	if method == http.MethodGet {
		if entry, ok := c.cache.Load(cacheKey); ok {
			req.Header.Set("If-None-Match", entry.(*etagEntry).etag)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if method == http.MethodGet && resp.StatusCode == http.StatusNotModified {
		resp.Body.Close()
		if entry, ok := c.cache.Load(cacheKey); ok {
			cached := entry.(*etagEntry)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     resp.Header,
				Body:       io.NopCloser(bytes.NewReader(cached.body)),
			}, nil
		}
	}

	if err := checkResponse(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	if method == http.MethodGet {
		if etag := resp.Header.Get("ETag"); etag != "" {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return nil, readErr
			}
			c.cache.Store(cacheKey, &etagEntry{etag: etag, body: bodyBytes})
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	return resp, nil
}

func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, v any) error {
	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// ResponseWithHeaders wraps the decoded result alongside response headers.
type ResponseWithHeaders struct {
	Header http.Header
}

func (c *Client) GetJSONWithHeaders(ctx context.Context, path string, query url.Values, v any) (*ResponseWithHeaders, error) {
	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return nil, err
	}
	return &ResponseWithHeaders{Header: resp.Header}, nil
}

// ParseLinkNext extracts the cursor from a Link header with rel="next".
// Format: </api/catalog/entities?after=CURSOR>; rel="next"
func ParseLinkNext(header string) string {
	for _, part := range splitLinks(header) {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, `rel="next"`) {
			continue
		}
		start := strings.Index(part, "<")
		end := strings.Index(part, ">")
		if start < 0 || end < 0 || end <= start {
			continue
		}
		link := part[start+1 : end]
		u, err := url.Parse(link)
		if err != nil {
			continue
		}
		if after := u.Query().Get("after"); after != "" {
			return after
		}
	}
	return ""
}

func splitLinks(s string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<':
			depth++
			current.WriteRune(r)
		case '>':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
				continue
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func (c *Client) GetRaw(ctx context.Context, path string, query url.Values) ([]byte, error) {
	resp, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	const maxResponseSize = 50 * 1024 * 1024 // 50 MB
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
}
