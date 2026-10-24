package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestSendGuardTimesOutBeforeRequestWriteCompletes(t *testing.T) {
	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	guard := newSendGuardTransport(base, 20*time.Millisecond, time.Second)
	req := httptest.NewRequest(http.MethodPost, "http://example.test", strings.NewReader("payload"))

	_, err := guard.RoundTrip(req)
	var guardErr *SendGuardError
	if !errors.As(err, &guardErr) {
		t.Fatalf("error = %v, want SendGuardError", err)
	}
	if guardErr.Phase != sendGuardSending || guardErr.Reason != "request_send_timeout" || !guardErr.Retryable {
		t.Fatalf("guard error = %+v", guardErr)
	}
}

func TestSendGuardResponseHeaderTimeoutIsNotRetryable(t *testing.T) {
	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if trace := httptrace.ContextClientTrace(req.Context()); trace != nil && trace.WroteRequest != nil {
			trace.WroteRequest(httptrace.WroteRequestInfo{})
		}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	guard := newSendGuardTransport(base, time.Second, 25*time.Millisecond)
	req := httptest.NewRequest(http.MethodPost, "http://example.test", strings.NewReader("payload"))

	_, err := guard.RoundTrip(req)
	var guardErr *SendGuardError
	if !errors.As(err, &guardErr) {
		t.Fatalf("error = %v, want SendGuardError", err)
	}
	if guardErr.Phase != sendGuardHeaders || guardErr.Retryable {
		t.Fatalf("guard error = %+v", guardErr)
	}
}

func TestSendGuardStopsSendTimerAfterRequestIsWritten(t *testing.T) {
	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if trace := httptrace.ContextClientTrace(req.Context()); trace != nil && trace.WroteRequest != nil {
			trace.WroteRequest(httptrace.WroteRequestInfo{})
		}
		time.Sleep(40 * time.Millisecond)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("data: ok\n\n")),
		}, nil
	})
	guard := newSendGuardTransport(base, 10*time.Millisecond, time.Second)
	req := httptest.NewRequest(http.MethodPost, "http://example.test", strings.NewReader("payload"))

	resp, err := guard.RoundTrip(req)
	if err != nil {
		t.Fatalf("healthy response interrupted by send timeout: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || string(body) != "data: ok\n\n" {
		t.Fatalf("body = %q err=%v", body, err)
	}
}

func TestSendGuardUserCancellationWins(t *testing.T) {
	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	guard := newSendGuardTransport(base, time.Second, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "http://example.test", strings.NewReader("payload")).WithContext(ctx)
	cancel()

	_, err := guard.RoundTrip(req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestStreamRequestRetriesIncompleteSendOnlyOnce(t *testing.T) {
	var calls atomic.Int32
	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		n := calls.Add(1)
		if n == 1 {
			<-req.Context().Done()
			return nil, req.Context().Err()
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: [DONE]\n\n"))}, nil
	})
	client := &Client{
		httpClient:         &http.Client{Transport: newSendGuardTransport(base, 10*time.Millisecond, time.Second)},
		streamRetryBackoff: []time.Duration{time.Millisecond},
	}
	resp, err := client.doStreamRequest(context.Background(), "http://example.test", "", []byte(`{}`), "xai", &ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got := calls.Load(); got != 2 {
		t.Fatalf("transport calls = %d, want one safe resend", got)
	}
}

func TestStreamRequestDoesNotRetryResponseHeaderTimeout(t *testing.T) {
	var calls atomic.Int32
	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if trace := httptrace.ContextClientTrace(req.Context()); trace != nil && trace.WroteRequest != nil {
			trace.WroteRequest(httptrace.WroteRequestInfo{})
		}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	client := &Client{
		httpClient:         &http.Client{Transport: newSendGuardTransport(base, time.Second, 10*time.Millisecond)},
		streamRetryBackoff: []time.Duration{time.Millisecond},
	}
	_, err := client.doStreamRequest(context.Background(), "http://example.test", "", []byte(`{}`), "xai", &ChatRequest{})
	if err == nil || !nonRetryableSendError(err) {
		t.Fatalf("error = %v, want non-retryable response-header timeout", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("transport calls = %d, want no resend", got)
	}
}
