package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StreamHTTPClient wraps an http.Client with a retry loop for streaming POST
// requests. It uses the shared RetryPolicy and BackoffStrategy from the provider
// package so all provider implementations get consistent retry behaviour.
type StreamHTTPClient struct {
	Name       string
	HTTPClient *http.Client
	Policy     RetryPolicy
	RLPolicy   RetryPolicy
}

// Do sends an HTTP request and retries on transient network errors and retryable
// HTTP statuses (408, 429, 5xx) with exponential backoff + jitter.
//
// authCheck, when non-nil, is called on 401/403 responses. It should return nil
// to allow retry (transient auth failure) or an error (typically *AuthError) to
// abort. The caller is responsible for setting the Authorization header before
// calling Do.
//
// Do handles the connection + header phase only; once it returns a response the
// caller must read and close resp.Body.
func (c *StreamHTTPClient) Do(ctx context.Context, method, url string, headers map[string]string, body []byte, authCheck func(code int, body string) error) (*http.Response, error) {
	var lastErr error
	rateLimitCount := 0

	maxAttempts := c.Policy.MaxAttempts
	if c.RLPolicy.MaxAttempts > maxAttempts {
		maxAttempts = c.RLPolicy.MaxAttempts
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			var delay time.Duration
			if isRateLimitStatus(lastErr) {
				rateLimitCount++
				if rateLimitCount >= c.RLPolicy.MaxAttempts {
					return nil, fmt.Errorf("%s: rate limited after %d attempts", c.Name, rateLimitCount)
				}
				delay = c.RLPolicy.Backoff.Duration(rateLimitCount - 1)
			} else {
				delay = c.Policy.Backoff.Duration(attempt - 1)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("%s: build request: %w", c.Name, err)
		}
		for k, v := range headers {
			httpReq.Header.Set(k, v)
		}
		if headers == nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.HTTPClient.Do(httpReq)
		if err != nil {
			if !IsTransientNetErr(err) {
				return nil, fmt.Errorf("%s: request failed: %w", c.Name, err)
			}
			lastErr = fmt.Errorf("%s: request failed: %w", c.Name, err)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		bodyStr := strings.TrimSpace(string(msg))

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			if authCheck != nil {
				if err := authCheck(resp.StatusCode, bodyStr); err != nil {
					return nil, err
				}
				lastErr = fmt.Errorf("%s: auth retry", c.Name)
				continue
			}
			return nil, fmt.Errorf("%s: status %d: %s", c.Name, resp.StatusCode, bodyStr)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = &httpStatusError{name: c.Name, code: resp.StatusCode, body: bodyStr}
			if d := ParseRetryAfter(resp, 120*time.Second); d > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(d):
				}
			}
			continue
		}

		statusErr := fmt.Errorf("%s: status %d: %s", c.Name, resp.StatusCode, bodyStr)
		if !IsRetryableStatus(resp.StatusCode) {
			return nil, statusErr
		}
		if d := ParseRetryAfter(resp, 120*time.Second); d > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(d):
			}
		}
		lastErr = &httpStatusError{name: c.Name, code: resp.StatusCode, body: bodyStr}
	}
	return nil, lastErr
}

// httpStatusError carries an HTTP status code so retry-classification helpers
// can inspect it directly.
type httpStatusError struct {
	name string
	code int
	body string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("%s: HTTP %d: %s", e.name, e.code, e.body)
}

// isRateLimitStatus reports whether err is a rate-limit (429) httpStatusError.
func isRateLimitStatus(err error) bool {
	var se *httpStatusError
	if errors.As(err, &se) {
		return se.code == http.StatusTooManyRequests
	}
	return false
}
