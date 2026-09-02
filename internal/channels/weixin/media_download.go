// Package weixin — iLink 图片消息下载（v4.8 子项 b：图片→vision 管线的
// 离线可测部分）。下载即防线：dial-time SSRF 私网拒绝 + 20MiB 尺寸上限 +
// Content-Type/魔数白名单；错误一律显式返回，不 panic。临时文件由调用方
// 经 cleanup 删除（幂等）。协议背景见 docs/ilink-non-text-protocol.md。
package weixin

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	wxImageMaxBytes = 20 << 20 // 20 MiB：入站图片下载尺寸上限
	wxImageTimeout  = 30 * time.Second
	wxImageSniffLen = 512 // 魔数嗅探字节数

	wxFileMaxBytes = 50 << 20 // 50 MiB：入站文件下载尺寸上限（文件不能魔数白名单，尺寸+MD5 兜底）
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
//   - file:// 形态（v4.8.3）：仅用于读取本包 DownloadImageEncrypted 生成的
//     解密临时文件（限 TempDir 前缀 + 魔数终审），cleanup 为 no-op（文件
//     所有权归构造方 recognizeImage 的 defer）。
//
// cleanup 幂等（关闭并删除临时文件），调用方 defer 即可。
func DownloadImage(rawURL string) (string, func(), error) {
	// file:// 分支只服务本包内部解密产物流转（识别器注入方无需感知协议细节），
	// 放行面限定 TempDir——消息内下发的 URL 永远是 http(s)，不会进到这里。
	if strings.HasPrefix(rawURL, "file://") {
		return readLocalDecryptedImage(strings.TrimPrefix(rawURL, "file://"))
	}
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

// DownloadImageEncrypted 下载 iLink 加密 CDN 媒体并解密落盘（v4.8.3 真机协议）：
// GET full_url（SSRF/20MiB 防线同 DownloadImage；密文无魔数，终审移到解密后）
// → AES-128-ECB(hexdecode aeskey) 解密 + PKCS7 去填充（真实 CDN 明文在图片
// EOI 后可能带少量尾随字节，交由解码器按 EOI 截断）→ 魔数白名单 → 临时文件。
// cleanup 幂等；解密失败显式返回错误（密文不落盘）。
func DownloadImageEncrypted(rawURL, aesKeyHex string) (string, func(), error) {
	key, err := parseAESKeyHex(aesKeyHex)
	if err != nil {
		return "", nil, err
	}
	blob, err := fetchMediaBytes(rawURL)
	if err != nil {
		return "", nil, err
	}
	plain, err := aes128ECBDecrypt(blob, key)
	if err != nil {
		return "", nil, fmt.Errorf("媒体解密失败: %w", err)
	}
	return saveImageFile(plain)
}

// fetchMediaBytes 下载 URL 全量字节：绝对 http(s) 校验 + dial-time SSRF
// （mediaTransport，重定向每跳重新校验）+ 20MiB 双保险（Content-Length
// 预检 + LimitReader）。不校验魔数与 Content-Type（密文两者皆无意义，
// 由调用方解密后终审）。
func fetchMediaBytes(rawURL string) ([]byte, error) {
	return fetchMediaBytesLimit(rawURL, wxImageMaxBytes)
}

// fetchMediaBytesLimit fetchMediaBytes 的尺寸参数化版（图片 20MiB / 文件
// 50MiB 共用同一条 SSRF/双保险防线）。
func fetchMediaBytesLimit(rawURL string, maxBytes int64) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("媒体 URL 非法（须为绝对 http/https 地址）: %s", truncateRunes(rawURL, 120))
	}
	client := &http.Client{Timeout: wxImageTimeout, Transport: mediaTransport()}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造媒体请求失败: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载媒体失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载媒体 HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxBytes {
		return nil, fmt.Errorf("媒体超过尺寸上限 %d 字节（Content-Length=%d）", maxBytes, resp.ContentLength)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("下载媒体失败: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("媒体超过尺寸上限 %d 字节", maxBytes)
	}
	return data, nil
}

// saveImageFile 魔数白名单终审后落临时文件（解密产物路径；cleanup 幂等）。
func saveImageFile(data []byte) (string, func(), error) {
	if !sniffImageMagic(data) {
		return "", nil, fmt.Errorf("图片魔数不在白名单（png/jpeg/webp/gif）")
	}
	tmp, err := os.CreateTemp("", "gaea-wx-media-*.img")
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
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("写临时文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("关闭临时文件失败: %w", err)
	}
	return path, cleanup, nil
}

// readLocalDecryptedImage 读取本包 DownloadImageEncrypted 生成的解密临时文件
// （DownloadImage 的 file:// 分支）。安全面：路径必须位于 os.TempDir() 下
// （只接受我们自己构造的路径，绝不让外部输入指到任意本地文件），魔数白名单
// 终审；返回 no-op cleanup——文件所有权归构造方（recognizeImage 的 defer）。
func readLocalDecryptedImage(path string) (string, func(), error) {
	cleaned := filepath.Clean(path)
	if !strings.HasPrefix(strings.ToLower(cleaned), strings.ToLower(os.TempDir())) {
		return "", nil, fmt.Errorf("file:// 仅允许读取媒体解密临时目录下的文件")
	}
	st, err := os.Stat(cleaned)
	if err != nil {
		return "", nil, err
	}
	if st.Size() > wxImageMaxBytes {
		return "", nil, fmt.Errorf("图片超过尺寸上限 %d 字节", wxImageMaxBytes)
	}
	data, err := os.ReadFile(cleaned)
	if err != nil {
		return "", nil, err
	}
	if !sniffImageMagic(data) {
		return "", nil, fmt.Errorf("图片魔数不在白名单（png/jpeg/webp/gif）")
	}
	return cleaned, func() {}, nil
}

// ─── 入站文件下载（v4.9 真机抓包定稿 2026-09-02）─────────────────────────────

// resolveInboundFile 下载并解密入站文件到临时文件（file_item 与 image_item
// 完全同构：media.full_url CDN 密文 + media.aes_key base64-of-hex，解密复用
// DownloadImageEncrypted 同款 AES-128-ECB + PKCS7 实现）。
//
// 防线（文件不能魔数白名单，改为）：
//   - 声明大小预检：len/size 超 50MiB 直接拒绝，不发起下载；
//   - 下载走 fetchMediaBytesLimit：dial-time SSRF 同款防线 + 50MiB 双保险；
//   - 解密后长度复核 + 明文 MD5 与 file_item.md5 比对：对上则可信；对不上仅
//     Warn 保留（微信 CDN 偶发差异以实测为准，不拒收）；
//   - 落 os.TempDir()/gaea-wxfile-*.tmp，清理责任归调用方（用后即删）。
//
// fileName 取 file_name → name 留位；sizeBytes/md5sum 为解密后明文实测值。
func resolveInboundFile(fi fileItem) (localPath, fileName string, sizeBytes int64, md5sum string, err error) {
	name := fileItemDisplayName(fi)
	rawURL, key, err := resolveFileDownload(fi)
	if err != nil {
		return "", name, 0, "", err
	}
	if rawURL == "" {
		return "", name, 0, "", fmt.Errorf("file_item 无可用下载地址（无 media.full_url/url）")
	}
	// 声明大小预检（真机 len 为字符串防御解析成的数值；旧留位 size 兜底）
	if sizeBytes = fi.Len; sizeBytes == 0 {
		sizeBytes = fi.Size
	}
	if sizeBytes > wxFileMaxBytes {
		return "", name, 0, "", fmt.Errorf("文件声明大小 %d 超过上限 %d 字节", sizeBytes, wxFileMaxBytes)
	}
	blob, err := fetchMediaBytesLimit(rawURL, wxFileMaxBytes)
	if err != nil {
		return "", name, 0, "", err
	}
	plain := blob
	if key != nil {
		plain, err = aes128ECBDecrypt(blob, key)
		if err != nil {
			return "", name, 0, "", fmt.Errorf("文件解密失败: %w", err)
		}
	}
	if int64(len(plain)) > wxFileMaxBytes {
		return "", name, 0, "", fmt.Errorf("文件超过尺寸上限 %d 字节", wxFileMaxBytes)
	}
	md5sum = md5Hex(plain)
	if fi.MD5 != "" && !strings.EqualFold(fi.MD5, md5sum) {
		// 对不上不拒收：微信 CDN 偶发差异以实测为准，交上层消费者自行判断
		slog.Warn("[weixin] 入站文件 MD5 与 file_item.md5 不符，保留文件",
			"name", name, "declared", fi.MD5, "actual", md5sum)
	}
	path, err := saveInboundFile(plain)
	if err != nil {
		return "", name, 0, "", err
	}
	return path, name, int64(len(plain)), md5sum, nil
}

// resolveFileDownload 解析文件的下载地址与密钥（同 imageItem.resolveDownload
// 口径）：url = media.full_url 否则留位字段 url；key = media.aes_key
//（base64-of-hex）反解。aes_key 存在但解析失败是异常形态——显式报错（绝不把
// 密文当明文消费）；无 media/无 aes_key 按明文 URL 处理（留位形态，宁漏勿误）。
func resolveFileDownload(fi fileItem) (rawURL string, key []byte, err error) {
	rawURL = fi.URL
	if fi.Media == nil {
		return rawURL, nil, nil
	}
	if rawURL == "" {
		rawURL = fi.Media.FullURL
	}
	if fi.Media.AESKey == "" {
		return rawURL, nil, nil
	}
	key, err = aesKeyFromBase64Hex(fi.Media.AESKey)
	if err != nil {
		return rawURL, nil, fmt.Errorf("media.aes_key 解析失败: %w", err)
	}
	return rawURL, key, nil
}

// saveInboundFile 文件明文落临时文件（无魔数白名单——文件类型不限；
// gaea-wxfile-*.tmp 命名便于辨识，写入失败即删不留残片）。
func saveInboundFile(data []byte) (string, error) {
	tmp, err := os.CreateTemp("", "gaea-wxfile-*.tmp")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	path := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("写临时文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("关闭临时文件失败: %w", err)
	}
	return path, nil
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
