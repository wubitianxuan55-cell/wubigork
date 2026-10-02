// ssrf.go — 共享 SSRF 守卫（v4.374，Reasonix installsource/ssrf.go 的
// parity 蒸馏）：把「哪些 IP 是内部地址、不许自动抓取」的判据收敛为单一
// 来源，供 webfetch（放行 loopback 的变体）与 booksource（内网全禁）共用。
package netclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// cgnatRange is RFC 6598 shared address space (100.64.0.0/10). Go's IsPrivate
// doesn't cover it, yet some clouds host instance metadata there (Alibaba Cloud
// at 100.100.100.200), so it's an SSRF target fetchers must refuse.
var cgnatRange = mustCIDR("100.64.0.0/10")

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

// BlockedSensitiveIP reports whether ip is a sensitive internal address that
// automated fetchers must not reach: RFC1918 + IPv6 unique-local, link-local
// (incl. cloud metadata 169.254.169.254), link-local multicast, unspecified,
// and CGNAT 100.64.0.0/10. Loopback is NOT included — web_fetch deliberately
// allows localhost (local dev servers); use BlockedInternalIP for the
// loopback-included variant.
func BlockedSensitiveIP(ip net.IP) bool {
	return ip.IsPrivate() || // RFC1918 + IPv6 unique-local (fc00::/7)
		ip.IsLinkLocalUnicast() || // 169.254.0.0/16 (incl. cloud metadata) + fe80::/10
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || // 0.0.0.0 / ::
		cgnatRange.Contains(ip) // 100.64.0.0/10 (incl. Alibaba Cloud metadata)
}

// BlockedInternalIP 是 BlockedSensitiveIP 的全禁变体：loopback（含云元数据
// 之外的 127.0.0.0/8 与 ::1）同样拒绝。用于抓取「任意外部 URL」且无本地
// 合理目标的通道（书源等）——被提示注入的规则不该摸到本机服务。
func BlockedInternalIP(ip net.IP) bool {
	return ip.IsLoopback() || BlockedSensitiveIP(ip)
}

// GuardedClient builds an HTTP client whose dialer refuses to connect to any
// internal address (BlockedInternalIP). The check runs at dial time on every
// resolved IP, so a public host that redirects or DNS-rebinds to an internal
// address is caught too. Proxying follows the environment (same semantics as
// the http.DefaultClient this replaces).
func GuardedClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				for _, ip := range ips {
					if BlockedInternalIP(ip.IP) {
						return nil, fmt.Errorf("refusing to fetch internal address %s (resolves to %s)", host, ip.IP)
					}
				}
				// 审计 P1 IN2-04：拨号目标必须取自已校验集合，逐个尝试——
				// 此前固定拨 ips[0]，单一目标连接失败即整体失败（无回退）。
				var lastErr error
				for _, ip := range ips {
					conn, derr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
					if derr == nil {
						return conn, nil
					}
					lastErr = derr
				}
				if lastErr != nil {
					return nil, lastErr
				}
				return nil, fmt.Errorf("no usable address to dial for %s", host)
			},
		},
	}
}
