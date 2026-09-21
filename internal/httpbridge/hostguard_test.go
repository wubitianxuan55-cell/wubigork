package httpbridge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// v4.374（Reasonix serve/hostguard 蒸馏）：Host 白名单挡 DNS-rebinding。
func TestHostGuard(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		listenAddr string
		host       string
		wantCode   int
	}{
		{"loopback ok", "127.0.0.1:8080", "127.0.0.1:8080", 200},
		{"localhost ok", "127.0.0.1:8080", "localhost:8080", 200},
		{"ipv6 loopback ok", "127.0.0.1:8080", "[::1]:8080", 200},
		{"rebind host 421", "127.0.0.1:8080", "evil.example.com:8080", 421},
		{"rebind host no port 421", "127.0.0.1:8080", "evil.example.com", 421},
		{"empty host passes", "127.0.0.1:8080", "", 200},
		{"listen host ok", "192.168.1.5:8080", "192.168.1.5:8080", 200},
		{"foreign host on lan bind 421", "192.168.1.5:8080", "evil.example.com", 421},
		{"wildcard bind skips check", "0.0.0.0:8080", "evil.example.com", 200},
		{"empty addr skips check", "", "evil.example.com", 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(HostGuard(tt.listenAddr, next))
			defer srv.Close()
			req := httptest.NewRequest(http.MethodGet, srv.URL, nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()
			srv.Config.Handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantCode {
				t.Fatalf("Host %q on %q: code = %d, want %d", tt.host, tt.listenAddr, rec.Code, tt.wantCode)
			}
		})
	}
}

// 桥 Handler 自带 loopback 守卫（httptest 自身绑定 127.0.0.1，正常请求不受影响）。
func TestBridgeHandlerRejectsForeignHost(t *testing.T) {
	b := New(&fakeApp{})
	srv := httptest.NewServer(b.Handler())
	defer srv.Close()
	req := httptest.NewRequest(http.MethodPost, srv.URL+"/api/rpc", nil)
	req.Host = "evil.example.com"
	rec := httptest.NewRecorder()
	b.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMisdirectedRequest {
		t.Fatalf("foreign Host should be 421, got %d", rec.Code)
	}
}
