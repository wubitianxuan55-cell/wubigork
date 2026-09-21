// hostguard.go — Host 头白名单中间件（v4.374，Reasonix internal/serve
// hostguard.go 的 HTTP 421 蒸馏）。动机：绑定在 loopback 上的本地 HTTP 服务
// 仍可被 DNS-rebinding 攻击——恶意网页先把自己的域名解析到 127.0.0.1，浏览器
// 侧即「同源」，可无预检地 POST 任意 Content-Type 驱动 /api/rpc。Host 校验
// 在 rebind 场景拿到的是攻击者域名，直接 421 拒绝。
package httpbridge

import (
	"net"
	"net/http"
	"strings"
)

// hostAllowlist 派生自监听地址：loopback 名单 + 精确监听 host。监听通配
// 地址（""、"0.0.0.0"、"::"、"[::]"）= 有意对外暴露，校验无意义，放行为
// no-op（与 Reasonix behind_proxy/通配豁免同口径）。
func hostAllowlist(listenAddr string) map[string]bool {
	host, _, err := net.SplitHostPort(listenAddr)
	if err != nil {
		host = strings.Trim(listenAddr, "[]")
	}
	host = strings.ToLower(strings.Trim(host, "[]"))
	switch host {
	case "", "0.0.0.0", "::", "any", "*":
		return nil // 通配绑定：跳过校验
	}
	allow := map[string]bool{"127.0.0.1": true, "::1": true, "localhost": true}
	allow[host] = true
	return allow
}

// requestHost 剥端口、去 IPv6 方括号、小写——Host 头的比对形态。
func requestHost(r *http.Request) string {
	h := strings.TrimSpace(r.Host)
	if h == "" {
		return "" // HTTP/1.0 无 Host：放行（无法路由的古老客户端，无 rebind 面）
	}
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
	}
	return strings.ToLower(strings.Trim(h, "[]"))
}

// HostGuard rejects requests whose Host is not loopback or the exact listen
// host (HTTP 421 Misdirected Request). allow == nil disables the check
// (wildcard bind). Empty request Host (HTTP/1.0) passes — a client that
// cannot say where it is going cannot be rebind-targeted either.
func HostGuard(listenAddr string, next http.Handler) http.Handler {
	allow := hostAllowlist(listenAddr)
	if allow == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := requestHost(r)
		if h != "" && !allow[h] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMisdirectedRequest)
			_, _ = w.Write([]byte(`{"error":"unrecognized Host header"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
