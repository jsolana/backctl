package client

import (
	"fmt"
	"io"
	"net/http"
)

type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("backstage API error (HTTP %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("backstage API error (HTTP %d)", e.StatusCode)
}

func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

func (e *APIError) IsAuthError() bool {
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	const maxErrorBodySize = 1024 * 1024 // 1 MB
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    http.StatusText(resp.StatusCode),
		Body:       string(body),
	}
}
