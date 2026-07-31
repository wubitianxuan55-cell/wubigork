package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStreamHTTPClient_RetriesOnTransientNetErr(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			hijack, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server does not support hijack")
			}
			conn, _, _ := hijack.Hijack()
			conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &StreamHTTPClient{
		Name:       "test",
		HTTPClient: srv.Client(),
		Policy:     DefaultRetryPolicy(),
		RLPolicy:   RateLimitRetryPolicy(),
	}
	resp, err := c.Do(context.Background(), http.MethodPost, srv.URL, nil, []byte(`{}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestStreamHTTPClient_AuthFailsFast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := &StreamHTTPClient{
		Name:       "test",
		HTTPClient: srv.Client(),
		Policy:     DefaultRetryPolicy(),
		RLPolicy:   RateLimitRetryPolicy(),
	}
	_, err := c.Do(context.Background(), http.MethodPost, srv.URL, nil, []byte(`{}`), func(code int, body string) error {
		return &AuthError{Provider: "test", Status: code}
	})
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	var ae *AuthError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *AuthError, got %T", err)
	}
	if ae.Status != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", ae.Status)
	}
}

func TestStreamHTTPClient_RateLimitExhausted(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	rlPolicy := RateLimitRetryPolicy()
	rlPolicy.MaxAttempts = 2
	rlPolicy.Backoff.Base = time.Millisecond
	rlPolicy.Backoff.Max = time.Millisecond

	c := &StreamHTTPClient{
		Name:       "test",
		HTTPClient: srv.Client(),
		Policy:     DefaultRetryPolicy(),
		RLPolicy:   rlPolicy,
	}
	_, err := c.Do(context.Background(), http.MethodPost, srv.URL, nil, []byte(`{}`), nil)
	if err == nil {
		t.Fatal("expected rate-limit error, got nil")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestStreamHTTPClient_CtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := &StreamHTTPClient{
		Name:       "test",
		HTTPClient: &http.Client{Timeout: time.Second},
		Policy:     DefaultRetryPolicy(),
		RLPolicy:   RateLimitRetryPolicy(),
	}
	_, err := c.Do(ctx, http.MethodPost, srv.URL, nil, []byte(`{}`), nil)
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}

func TestStreamHTTPClient_RetryAfterHonored(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &StreamHTTPClient{
		Name:       "test",
		HTTPClient: srv.Client(),
		Policy:     DefaultRetryPolicy(),
		RLPolicy:   RateLimitRetryPolicy(),
	}
	start := time.Now()
	resp, err := c.Do(context.Background(), http.MethodPost, srv.URL, nil, []byte(`{}`), nil)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if elapsed < 900*time.Millisecond {
		t.Errorf("Retry-After not honored: elapsed=%v", elapsed)
	}
}

func TestStreamHTTPClient_NonRetryableStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := &StreamHTTPClient{
		Name:       "test",
		HTTPClient: srv.Client(),
		Policy:     DefaultRetryPolicy(),
		RLPolicy:   RateLimitRetryPolicy(),
	}
	_, err := c.Do(context.Background(), http.MethodPost, srv.URL, nil, []byte(`{}`), nil)
	if err == nil {
		t.Fatal("expected error for 400, got nil")
	}
}

func TestStreamHTTPClient_EmptyHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &StreamHTTPClient{
		Name:       "test",
		HTTPClient: srv.Client(),
		Policy:     DefaultRetryPolicy(),
		RLPolicy:   RateLimitRetryPolicy(),
	}
	resp, err := c.Do(context.Background(), http.MethodPost, srv.URL, nil, []byte(`{}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

func TestStreamHTTPClient_AuthCheckRetried(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &StreamHTTPClient{
		Name:       "test",
		HTTPClient: srv.Client(),
		Policy:     RetryPolicy{MaxAttempts: 4, Backoff: BackoffStrategy{Base: time.Millisecond, Max: time.Millisecond, Multiplier: 1}},
		RLPolicy:   RetryPolicy{MaxAttempts: 4, Backoff: BackoffStrategy{Base: time.Millisecond, Max: time.Millisecond, Multiplier: 1}},
	}
	authRetries := 0
	resp, err := c.Do(context.Background(), http.MethodPost, srv.URL, nil, []byte(`{}`), func(code int, body string) error {
		if authRetries < 2 {
			authRetries++
			return nil
		}
		return &AuthError{Provider: "test", Status: code}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if attempts != 3 {
		t.Errorf("expected 3 attempts (2 auth retries + success), got %d", attempts)
	}
}
