// Package weixin — iLink 图片消息下载（v4.8 子项 b：图片→vision 管线的
// 离线可测部分）。下载即防线：dial-time SSRF 私网拒绝 + 20MiB 尺寸上限 +
// Content-Type/魔数白名单；错误一律显式返回，不 panic。临时文件由调用方
// 经 cleanup 删除（幂等）。协议背景见 docs/ilink-non-text-protocol.md。
package weixin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	wxImageMaxBytes = 20 << 20 // 20 MiB：入站图片下载尺寸上限
	wxImageTimeout  = 30 * time.Second
	wxImageSniffLen = 512 // 魔数嗅探字节数
)

// mediaAllowLoopback 生产恒为 false（iLink 下发的 URL 是不可信输入，绝不
// 允许指向本机/内网）；测试翻成 true 让 httptest（127.0.0.1）可达。放行仅限
// 回环，RFC1918 私网段仍然拒绝。
var mediaAllowLoopback = false

// cgnatRange RFC 6598 共享地址段（100.64.0.0/10）：Go 的 IsPrivate 不覆盖它，
// 但部分云厂商把 instance metadata 放在这里（阿里云 100.100.100.200）。
var cgnatRange = mustCIDR("100.64.0.0/10")

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

// blockedMediaIP 报告 ip 是否为媒体下载必须拒绝的地址。对齐 webfetch 的
// dial-time 防线精神但更严：回环也拒绝（agent 的 web_fetch 保留回环是因
// bash 已可访问本机；这里的目标是聊天消息里下发的远程 URL，没有该豁免理由）。
func blockedMediaIP(ip net.IP) bool {
	if ip.IsLoopback() && !mediaAllowLoopback {
		return true
	}
	return ip.IsPrivate() || // RFC1918 + IPv6 unique-local (fc00::/7)
		ip.IsLinkLocalUnicast() || // 169.254.0.0/16（含云 metadata）+ fe80::/10
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || // 0.0.0.0 / ::
		cgnatRange.Contains(ip) // 100.64.0.0/10
}

// mediaTransport 直连（不走代理——代理会代替我们连内网，绕过 dial 校验）+
// dial-time SSRF 防线：解析出的每个 IP 都过 blockedMediaIP，公网域名被
// DNS rebinding 或重定向到内网也拦得住（重定向每一跳都会重新拨号校验）。
func mediaTransport() *http.Transport {
	dialer := &net.Dialer{Timeout: wxImageTimeout}
	return &http.Transport{
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
				if blockedMediaIP(ip.IP) {
					return nil, fmt.Errorf("拒绝下载内网/本机地址的媒体 %s（解析到 %s）", host, ip.IP)
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
		TLSHandshakeTimeout: wxImageTimeout,
	}
}

// DownloadImage 下载 iLink 下发的图片 URL 到临时文件，返回 (本地路径, cleanup, 错误)。
//
// 防线（离线可测部分）：
//   - URL 必须是绝对 http(s) 地址；
//   - dial-time SSRF：目标解析到私网/回环/链路本地/CGNAT 一律拒绝；
//   - 尺寸：Content-Length 预检 + io.LimitReader 双保险，超 20MiB 拒绝；
//   - 类型：Content-Type 为 text/* 直接拒绝；内容按魔数白名单
//     （png/jpeg/webp/gif）终审，白名单外不落盘。
//
// cleanup 幂等（关闭并删除临时文件），调用方 defer 即可。
func DownloadImage(rawURL string) (string, func(), error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", nil, fmt.Errorf("图片 URL 非法（须为绝对 http/https 地址）: %s", truncateRunes(rawURL, 120))
	}

	client := &http.Client{Timeout: wxImageTimeout, Transport: mediaTransport()}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("构造图片请求失败: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("下载图片失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("下载图片 HTTP %d", resp.StatusCode)
	}
	if ct := strings.ToLower(resp.Header.Get("Content-Type")); strings.HasPrefix(ct, "text/") {
		return "", nil, fmt.Errorf("图片 Content-Type %q 非白名单类型", ct)
	}
	if resp.ContentLength > wxImageMaxBytes {
		return "", nil, fmt.Errorf("图片超过尺寸上限 %d 字节（Content-Length=%d）", wxImageMaxBytes, resp.ContentLength)
	}

	// 先嗅探魔数再落盘：白名单外（含文本伪装）直接拒绝，不写临时文件。
	head := make([]byte, wxImageSniffLen)
	n, err := io.ReadFull(resp.Body, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", nil, fmt.Errorf("读取图片头部失败: %w", err)
	}
	head = head[:n]
	if !sniffImageMagic(head) {
		return "", nil, fmt.Errorf("图片魔数不在白名单（png/jpeg/webp/gif）")
	}

	tmp, err := os.CreateTemp("", "gaea-wx-img-*.png")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	path := tmp.Name()
	removed := false
	cleanup := func() {
		if !removed {
			removed = true
			_ = tmp.Close()
			_ = os.Remove(path)
		}
	}

	if _, err := tmp.Write(head); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("写临时文件失败: %w", err)
	}
	rest := int64(wxImageMaxBytes) - int64(len(head)) + 1 // +1 用于探测超限
	written, err := io.Copy(tmp, io.LimitReader(resp.Body, rest))
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("下载图片失败: %w", err)
	}
	if int64(len(head))+written > wxImageMaxBytes {
		cleanup()
		return "", nil, fmt.Errorf("图片超过尺寸上限 %d 字节", wxImageMaxBytes)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("关闭临时文件失败: %w", err)
	}
	return path, cleanup, nil
}

// sniffImageMagic 魔数白名单判定：png / jpeg / webp / gif。
func sniffImageMagic(head []byte) bool {
	switch {
	case len(head) >= 8 && string(head[:8]) == "\x89PNG\r\n\x1a\n":
		return true
	case len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		return true
	case len(head) >= 6 && (string(head[:6]) == "GIF87a" || string(head[:6]) == "GIF89a"):
		return true
	case len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		return true
	}
	return false
}

// OCRMediaRecognizer 把「URL → 下载 → OCR → 清理」打包成 Server.MediaRecognizer
// 可直接注入的适配器（v4.8 子项 b）：app 层注入点保持一行
// `srv.MediaRecognizer = weixin.OCRMediaRecognizer(a.GaeaOCRText)`。
// ocr 收本地图片路径返回识别文本；下载/OCR 失败原样返回错误，由 handle 层
// 保留占位提示行。
func OCRMediaRecognizer(ocr func(imagePath string) (string, error)) func(url string) (string, error) {
	return func(u string) (string, error) {
		path, cleanup, err := DownloadImage(u)
		if err != nil {
			return "", err
		}
		defer cleanup()
		return ocr(path)
	}
}
