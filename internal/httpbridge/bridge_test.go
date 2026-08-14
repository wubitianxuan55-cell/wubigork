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
)// fakeApp mirrors the Wails binding surface shape: plain args, (T, error)
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

// rpcWithAuth 携带指定 token 调用 /api/rpc；token 为空则不携带任何凭证。
func rpcWithAuth(t *testing.T, srv *httptest.Server, method string, args any, token string) (int, rpcResponse) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"method": method, "args": args})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/rpc", strings.NewReader(string(body)))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/rpc: %v", err)
	}
	defer resp.Body.Close()
	var out rpcResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

// statusOf 返回一次 GET 请求的状态码（SSE 场景只关心鉴权结果）。
func statusOf(t *testing.T, srv *httptest.Server, path string) int {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// statusOfHeader 返回一次携带指定请求头的 GET 状态码（Bearer / X-Gaea-Token 鉴权路径）。
func statusOfHeader(t *testing.T, srv *httptest.Server, path, header, value string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	if value != "" {
		req.Header.Set(header, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestTokenAuth(t *testing.T) {
	const tok = "s3cr3t-token"
	srv := httptest.NewServer(NewWithToken(&fakeApp{}, tok).Handler())
	defer srv.Close()

	// 无 token → 401，且不泄露结果。
	if code, _ := rpcWithAuth(t, srv, "Greet", []any{"world"}, ""); code != http.StatusUnauthorized {
		t.Fatalf("no-token rpc status = %d, want 401", code)
	}
	// 错误 token → 401。
	if code, _ := rpcWithAuth(t, srv, "Greet", []any{"world"}, "wrong"); code != http.StatusUnauthorized {
		t.Fatalf("bad-token rpc status = %d, want 401", code)
	}
	// 正确 token（Authorization: Bearer）→ 200 且结果正常。
	if code, got := rpcWithAuth(t, srv, "Greet", []any{"world"}, tok); code != http.StatusOK || got.Result != "hi world" {
		t.Fatalf("bearer rpc = %d %+v, want 200 hi world", code, got)
	}
	// X-Gaea-Token 头同样可用。
	body, _ := json.Marshal(map[string]any{"method": "Add", "args": []any{2, 3}})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/rpc", strings.NewReader(string(body)))
	req.Header.Set("X-Gaea-Token", tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("x-token rpc: %v", err)
	}
	defer resp.Body.Close()
	var out rpcResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != http.StatusOK || out.Result != float64(5) {
		t.Fatalf("x-token rpc = %d %+v, want 200 5", resp.StatusCode, out)
	}
	// SSE：无 token 401；URL ?token= 已废弃——即使携带正确 token 也必须 401
	// （前端已改用 fetch 流式 SSE 携带 Authorization 头，T6-9.6）。
	if code := statusOf(t, srv, "/api/stream?id=chat"); code != http.StatusUnauthorized {
		t.Fatalf("sse no-token status = %d, want 401", code)
	}
	if code := statusOf(t, srv, "/api/stream?id=chat&token="+tok); code != http.StatusUnauthorized {
		t.Fatalf("sse url-token status = %d, want 401（URL token 已废弃）", code)
	}
	// Authorization: Bearer 仍 200。
	if code := statusOfHeader(t, srv, "/api/stream?id=chat", "Authorization", "Bearer "+tok); code != http.StatusOK {
		t.Fatalf("sse bearer status = %d, want 200", code)
	}
	// X-Gaea-Token 头仍 200。
	if code := statusOfHeader(t, srv, "/api/stream?id=chat", "X-Gaea-Token", tok); code != http.StatusOK {
		t.Fatalf("sse x-token status = %d, want 200", code)
	}
	// /api/health 保持开放（存活探针）。
	if code := statusOf(t, srv, "/api/health"); code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", code)
	}
}

func TestSessionToken(t *testing.T) {
	// 显式 token 原样返回。
	if got := SessionToken("abc"); got != "abc" {
		t.Fatalf("SessionToken(abc) = %q", got)
	}
	// 自动生成：两次不同且为 32 位十六进制。
	a, b := SessionToken(""), SessionToken("")
	if a == b {
		t.Fatalf("auto tokens must differ: %q", a)
	}
	if len(a) != 32 {
		t.Fatalf("auto token length = %d, want 32", len(a))
	}
	for _, r := range a {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("auto token not hex: %q", a)
		}
	}
}

