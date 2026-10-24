package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"syscall"
	"time"
)

const defaultRequestSendBudget = 300 * time.Second

type sendGuardPhase string

const (
	sendGuardSending sendGuardPhase = "sending"
	sendGuardHeaders sendGuardPhase = "response_headers"
	sendGuardReading sendGuardPhase = "response_body"
)

// SendGuardError identifies which request phase failed and whether retry is
// safe. The original transport error remains available via errors.Is/As.
type SendGuardError struct {
	Phase     sendGuardPhase
	Reason    string
	Retryable bool
	Err       error
}

func (e *SendGuardError) Error() string {
	if e == nil {
		return ""
	}
	if e.Reason != "" {
		return fmt.Sprintf("request %s (%s): %v", e.Phase, e.Reason, e.Err)
	}
	return fmt.Sprintf("request %s: %v", e.Phase, e.Err)
}

func (e *SendGuardError) Unwrap() error { return e.Err }

func nonRetryableSendError(err error) bool {
	var guardErr *SendGuardError
	return errors.As(err, &guardErr) && !guardErr.Retryable
}

type sendGuardTransport struct {
	base         http.RoundTripper
	sendBudget   time.Duration
	headerBudget time.Duration
}

func newSendGuardTransport(base http.RoundTripper, sendBudget, headerBudget time.Duration) *sendGuardTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	if sendBudget <= 0 {
		sendBudget = defaultRequestSendBudget
	}
	if headerBudget <= 0 {
		headerBudget = defaultStreamIdleTimeout
	}
	return &sendGuardTransport{base: base, sendBudget: sendBudget, headerBudget: headerBudget}
}

func (t *sendGuardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	outerCtx := req.Context()
	attemptCtx, cancel := context.WithCancel(outerCtx)

	var mu sync.Mutex
	phase := sendGuardSending
	timedOutSend := false
	wroteRequest := false
	var headerTimer *time.Timer

	sendTimer := time.AfterFunc(t.sendBudget, func() {
		mu.Lock()
		if phase == sendGuardSending {
			timedOutSend = true
		}
		mu.Unlock()
		cancel()
	})

	trace := &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			mu.Lock()
			defer mu.Unlock()
			if wroteRequest || phase != sendGuardSending {
				return
			}
			wroteRequest = info.Err == nil
			if info.Err != nil {
				return
			}
			sendTimer.Stop()
			phase = sendGuardHeaders
			headerTimer = time.AfterFunc(t.headerBudget, cancel)
		},
	}

	attemptReq := req.Clone(httptrace.WithClientTrace(attemptCtx, trace))
	resp, err := t.base.RoundTrip(attemptReq)
	sendTimer.Stop()
	if headerTimer != nil {
		headerTimer.Stop()
	}
	if err != nil {
		cancel()
		if outerCtx.Err() != nil {
			return nil, err
		}
		mu.Lock()
		phaseNow := phase
		wrote := wroteRequest
		sendTimedOut := timedOutSend
		mu.Unlock()
		if sendTimedOut {
			return nil, &SendGuardError{
				Phase: sendGuardSending, Reason: "request_send_timeout", Retryable: true, Err: err,
			}
		}
		if phaseNow == sendGuardHeaders {
			return nil, &SendGuardError{
				Phase: sendGuardHeaders, Retryable: false, Err: err,
			}
		}
		return nil, &SendGuardError{
			Phase: sendGuardSending, Reason: classifySendReason(err),
			Retryable: !wrote, Err: err,
		}
	}

	mu.Lock()
	phase = sendGuardReading
	mu.Unlock()
	if resp.Body != nil {
		resp.Body = &sendGuardBody{ReadCloser: resp.Body, cancel: cancel}
	} else {
		cancel()
	}
	return resp, nil
}

type sendGuardBody struct {
	io.ReadCloser
	cancel context.CancelFunc
	once   sync.Once
}

func (b *sendGuardBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(b.cancel)
	return err
}

func classifySendReason(err error) string {
	if err == nil {
		return ""
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns"
	}
	if errors.Is(err, io.EOF) {
		return "eof"
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return "reset"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "http2"):
		return "http2_protocol"
	case strings.Contains(msg, "connection reset"):
		return "reset"
	case strings.Contains(msg, "eof"):
		return "eof"
	case strings.Contains(msg, "timeout"):
		return "timeout"
	case strings.Contains(msg, "dns"):
		return "dns"
	default:
		return ""
	}
}
