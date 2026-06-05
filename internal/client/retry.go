package client

import (
	"math"
	"math/rand/v2"
	"net/http"
	"time"
)

type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
	baseDelay  time.Duration
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !isIdempotent(req.Method) {
		return t.base.RoundTrip(req)
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			delay := t.backoff(attempt)
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(delay):
			}
		}

		resp, err = t.base.RoundTrip(req)
		if err != nil {
			continue
		}

		if !isRetryable(resp.StatusCode) {
			return resp, nil
		}

		if attempt < t.maxRetries {
			resp.Body.Close()
		}
	}

	if err != nil {
		return nil, err
	}
	return resp, nil
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

func (t *retryTransport) backoff(attempt int) time.Duration {
	d := float64(t.baseDelay) * math.Pow(2, float64(attempt-1))
	jitter := d * 0.2 * rand.Float64()
	return time.Duration(d + jitter)
}

func isRetryable(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}
