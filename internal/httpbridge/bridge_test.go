package httpbridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeApp mirrors the Wails binding surface shape: plain args, (T, error)
// returns, error-only returns, and no-arg methods.
type fakeApp struct{}

func (f *fakeApp) Greet(name string) string  { return "hi " + name }
func (f *fakeApp) Add(a, b int) (int, error) { return a + b, nil }
func (f *fakeApp) Fail() (string, error)     { return "", errors.New("boom") }
func (f *fakeApp) Noop()                     {}
func (f *fakeApp) Stats() map[string]int     { return map[string]int{"tools": 3} }

func rpc(t *testing.T, srv *httptest.Server, method string, args any) rpcResponse {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"method": method, "args": args})
	resp, err := http.Post(srv.URL+"/api/rpc", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("POST /api/rpc: %v", err)
	}
	defer resp.Body.Close()
	var out rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestRPCDispatch(t *testing.T) {
	srv := httptest.NewServer(New(&fakeApp{}).Handler())
	defer srv.Close()

	if got := rpc(t, srv, "Greet", []any{"world"}); got.Error != "" || got.Result != "hi world" {
		t.Fatalf("Greet = %+v, want hi world", got)
	}
	if got := rpc(t, srv, "Add", []any{2, 3}); got.Error != "" || got.Result != float64(5) {
		t.Fatalf("Add = %+v, want 5", got)
	}
	if got := rpc(t, srv, "Noop", nil); got.Error != "" || got.Result != nil {
		t.Fatalf("Noop = %+v, want empty result", got)
	}
	// map result survives JSON round-trip
	if got := rpc(t, srv, "Stats", nil); got.Error != "" {
		t.Fatalf("Stats error: %+v", got)
	}
	// error convention: (T, error) with non-nil error surfaces in Error field
	if got := rpc(t, srv, "Fail", nil); got.Error != "boom" {
		t.Fatalf("Fail = %+v, want error boom", got)
	}
	// unknown method
	if got := rpc(t, srv, "Nope", nil); !strings.Contains(got.Error, "unknown method") {
		t.Fatalf("Nope = %+v, want unknown method error", got)
	}
	// missing trailing args → zero values
	if got := rpc(t, srv, "Add", []any{7}); got.Error != "" || got.Result != float64(7) {
		t.Fatalf("Add(7) = %+v, want 7", got)
	}
}

func TestSSEPublish(t *testing.T) {
	srv := httptest.NewServer(New(&fakeApp{}).Handler())
	defer srv.Close()

	// Subscribe first, then publish — the event must arrive on the stream.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/stream?id=chat", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open SSE: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE status = %d", resp.StatusCode)
	}

	// consume the initial "connected" frame
	br := bufio.NewReader(resp.Body)
	if _, err := br.ReadString('\n'); err != nil {
		t.Fatalf("read connected frame: %v", err)
	}
	if _, err := br.ReadString('\n'); err != nil {
		t.Fatalf("read connected data: %v", err)
	}
	if _, err := br.ReadString('\n'); err != nil {
		t.Fatalf("read connected blank: %v", err)
	}

	Publish("chat", map[string]interface{}{"kind": "text", "text": "hello"})
	line, err := br.ReadString('\n')
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	if !strings.HasPrefix(line, "data: ") {
		t.Fatalf("unexpected frame: %q", line)
	}
	if !strings.Contains(line, "hello") {
		t.Fatalf("payload missing text: %q", line)
	}
}
