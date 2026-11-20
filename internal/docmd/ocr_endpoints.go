package docmd

// ocr_endpoints.go — 本地 OCR 服务的被动观测面（模型中心「本地服务目录」用）。
//
// 与 seam 主体（ocr.go）的分工：这里只读地址与健康，**绝不拉起服务**
//（ovisServerBase/uocr2ServerBase 的「探测→按需拉起」是 OCR 执行链行为；
// 目录列表只是看一眼，拉起会凭空吃显存/CPU）。显式引擎入口
// OCRImageTextWithEngine 供模型中心「设为 OCR（本地）」定点调用，语义与
// GAEA_OCR_ENGINE=<kind> 完全一致（仅该引擎，fail-closed）。

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// OvisBaseURL 被动解析 OvisOCR2 llama-server 地址（GAEA_OCR_URL 优先，其次
// GAEA_OCR_PORT，默认 8137）。只算地址不探测不拉起。
func OvisBaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("GAEA_OCR_URL")), "/")
	if base == "" {
		port := os.Getenv("GAEA_OCR_PORT")
		if port == "" {
			port = "8137"
		}
		base = "http://127.0.0.1:" + port
	}
	return base
}

// Uocr2LlamaAddr 被动解析 Unlimited-OCR-2 llama 后端地址（默认 8139）。
func Uocr2LlamaAddr() string { return uocr2LlamaBaseURL() }

// Uocr2PythonAddr 被动解析 Unlimited-OCR-2 python 后端地址（默认 8138）。
func Uocr2PythonAddr() string { return uocr2PythonBaseURL() }

// LocalServiceHealth 对本地服务 /health 做一次被动探测（800ms 超时，应答含
// "ok" 即就绪）。任何传输层失败=false；绝不触发服务拉起。
func LocalServiceHealth(base string) bool {
	if strings.TrimSpace(base) == "" {
		return false
	}
	client := &http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(strings.TrimRight(base, "/") + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return strings.Contains(string(body), "ok")
}

// OCRImageTextWithEngine 显式指定本地引擎 kind 跑单图 OCR（uocr2/ovis/
// tesseract，或注册表中的测试 kind）。语义与 GAEA_OCR_ENGINE=<kind> 一致：
// 仅该引擎、不可用 fail-closed、不静默回退 auto。缓存键含 kind，与 env 链
// 互不串味。模型中心「设为 OCR（本地）」链路的定点入口。
func OCRImageTextWithEngine(path, kind string) (string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	known := false
	for _, k := range OCRProviderKinds() {
		if k == kind {
			known = true
			break
		}
	}
	if !known {
		return "", fmt.Errorf("docmd: 未知本地 OCR 引擎 %q（可用：%s）", kind, strings.Join(OCRProviderKinds(), " / "))
	}
	fp, ferr := ocrFileFingerprint(path)
	if ferr != nil {
		return ocrImageTextUncached(path, []string{kind})
	}
	key := "img:" + fp + ":local:" + kind
	if text, ok := defaultOCRCache.get(key); ok {
		emitOCRProgress(OCRProgressEvent{Path: path, Stage: OCRStageCached, Total: 1, Chars: utf8.RuneCountInString(text)})
		return text, nil
	}
	return ocrCached(key, func() (string, error) {
		return ocrImageTextUncached(path, []string{kind})
	})
}
