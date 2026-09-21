package netclient

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// v4.374（Reasonix installsource SSRF parity 蒸馏）：内部地址判据与守卫 client。
func TestBlockedInternalIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},            // loopback（全禁变体）
		{"127.8.8.8", true},            // loopback 网段
		{"::1", true},                  // IPv6 loopback
		{"10.0.0.5", true},             // RFC1918
		{"192.168.1.1", true},          // RFC1918
		{"169.254.169.254", true},      // 云元数据
		{"100.100.100.200", true},      // 阿里云元数据（CGNAT 段）
		{"0.0.0.0", true},              // 未指定
		{"8.8.8.8", false},             // 公网
		{"1.1.1.1", false},             // 公网
	}
	for _, tt := range tests {
		if got := BlockedInternalIP(net.ParseIP(tt.ip)); got != tt.want {
			t.Errorf("BlockedInternalIP(%s) = %v, want %v", tt.ip, got, tt.want)
		}
	}
	// webfetch 变体放行 loopback（本地 dev server 可抓）。
	if BlockedSensitiveIP(net.ParseIP("127.0.0.1")) {
		t.Error("BlockedSensitiveIP must allow loopback (webfetch semantics)")
	}
	if !BlockedSensitiveIP(net.ParseIP("169.254.169.254")) {
		t.Error("BlockedSensitiveIP must refuse cloud metadata")
	}
}

// GuardedClient 在拨号层拒绝内网目标（含重定向/rebind 后解析出的内网 IP）。
func TestGuardedClientRefusesLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("internal address must not be reached")
	}))
	defer srv.Close()

	client := GuardedClient(2 * time.Second)
	resp, err := client.Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("guarded client must refuse loopback target")
	}
	if !strings.Contains(err.Error(), "internal address") {
		t.Fatalf("error should name the guard, got: %v", err)
	}
}
