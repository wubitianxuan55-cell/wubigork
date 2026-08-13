// Package herdsman 提供对本地 Herdsman 模型服务的一次性健康检查能力。
//
// Herdsman 服务默认监听本机 8080 端口（OpenAI 兼容 API），gaea 的聊天/视觉/
// Embedding/Rerank/OCR/文档解析/ASR/TTS/生图/翻译等本地能力链均依赖它。
// 本文件实现 H0-2「服务健康检查」：端口占用探测、API 存活探测，以及按能力
// 归类「已装模型是否满足」各能力链。
package herdsman

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 默认探测参数：Herdsman 服务默认监听本机 8080。
const (
	// DefaultHost 默认探测主机。
	DefaultHost = "127.0.0.1"
	// DefaultPort 默认探测端口。
	DefaultPort = "8080"

	// portDialTimeout 端口探测超时（固定 1 秒）。
	portDialTimeout = 1 * time.Second
	// defaultAPITimeout API 探测默认超时（3 秒，可经 HealthCheck 的 timeout 参数覆盖）。
	defaultAPITimeout = 3 * time.Second
)

// ModelInfo 模型摘要信息，由调用方从模型中心引擎的模型列表转换而来。
type ModelInfo struct {
	ID         string `json:"id"`
	Capability string `json:"capability,omitempty"` // 能力归类；空时按 ID 关键词自动判断
	Status     string `json:"status,omitempty"`     // "running"/"stopped"/"unknown"；空视为可用
}

// HealthResult 一次健康检查的结果（JSON 可序列化，直接供前端展示）。
type HealthResult struct {
	PortOpen     bool            `json:"port_open"`            // TCP 端口是否可拨通
	PortError    string          `json:"port_error,omitempty"` // 端口探测失败原因
	APIReachable bool            `json:"api_reachable"`        // GET {baseURL}/models 是否成功
	APIError     string          `json:"api_error,omitempty"`  // API 探测失败原因（端口关闭时为 "port closed"）
	Capabilities map[string]bool `json:"capabilities"`         // 各能力是否已有可用模型
	Healthy      bool            `json:"healthy"`              // 端口 + API + 聊天模型齐备
	Summary      []string        `json:"summary,omitempty"`    // 中文问题清单
}

// capabilityKeys 全部能力键（固定顺序，结果 map 中恒存在，前端可稳定遍历）。
var capabilityKeys = []string{
	"chat", "vision", "embedding", "rerank", "ocr", "parse",
	"asr", "tts", "imagegen", "translation",
}

// capabilityLabels 能力键的中文展示名（用于 Summary 文案）。
var capabilityLabels = map[string]string{
	"chat":        "聊天",
	"vision":      "视觉",
	"embedding":   "Embedding",
	"rerank":      "Rerank",
	"ocr":         "OCR",
	"parse":       "文档解析",
	"asr":         "语音识别",
	"tts":         "语音合成",
	"imagegen":    "生图",
	"translation": "翻译",
}

// NewResult 返回预填全部能力键（均为 false）的空健康结果。
func NewResult() HealthResult {
	caps := make(map[string]bool, len(capabilityKeys))
	for _, k := range capabilityKeys {
		caps[k] = false
	}
	return HealthResult{Capabilities: caps}
}

// HealthCheck 执行一次 Herdsman 服务健康检查，步骤：
//  1. 端口探测：net.DialTimeout 拨 baseURL 的 host:port（1 秒超时；baseURL 为空或
//     解析失败时回退默认 127.0.0.1:8080）；
//  2. API 存活：GET {baseURL}/models（timeout 超时，<=0 时回退 3 秒）；
//     端口未开则跳过并置 APIError="port closed"；
//  3. 能力归类：模型 Status=running（或为空）且能力匹配即视为该能力就绪；
//     Capability 字段为空时按模型 ID 关键词归类（见 ClassifyModelCapability）。
//
// Healthy 要求端口、API 与聊天模型三者齐备。
func HealthCheck(baseURL string, models []ModelInfo, timeout time.Duration) HealthResult {
	result := NewResult()

	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "http://" + net.JoinHostPort(DefaultHost, DefaultPort)
	}
	addr := hostPortOf(baseURL)

	// 1. 端口探测
	conn, err := net.DialTimeout("tcp", addr, portDialTimeout)
	if err != nil {
		result.PortOpen = false
		result.PortError = err.Error()
	} else {
		result.PortOpen = true
		conn.Close()
	}

	// 2. API 存活探测（端口关闭则跳过）
	if !result.PortOpen {
		result.APIReachable = false
		result.APIError = "port closed"
	} else {
		apiTimeout := timeout
		if apiTimeout <= 0 {
			apiTimeout = defaultAPITimeout
		}
		result.APIReachable, result.APIError = pingModelsAPI(baseURL, apiTimeout)
	}

	// 3. 按能力归类已装模型
	for _, m := range models {
		if !modelUsable(m) {
			continue
		}
		cap := strings.TrimSpace(m.Capability)
		if cap == "" {
			cap = ClassifyModelCapability(m.ID)
		}
		if isKnownCapability(cap) {
			result.Capabilities[cap] = true
		}
	}

	result.Healthy = result.PortOpen && result.APIReachable && result.Capabilities["chat"]
	result.Summary = buildSummary(addr, result)
	return result
}

// modelUsable 判断模型是否可用：Status 为空视为可用，否则要求 running（大小写不敏感）。
func modelUsable(m ModelInfo) bool {
	s := strings.TrimSpace(m.Status)
	return s == "" || strings.EqualFold(s, "running")
}

// isKnownCapability 判断能力键是否在预定义列表内（避免未知键污染结果 map）。
func isKnownCapability(cap string) bool {
	for _, k := range capabilityKeys {
		if k == cap {
			return true
		}
	}
	return false
}

// hostPortOf 从 baseURL 提取 host:port；无端口时补默认端口 8080，
// 解析失败或 host 缺失时回退默认 127.0.0.1:8080。
func hostPortOf(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return net.JoinHostPort(DefaultHost, DefaultPort)
	}
	host := u.Hostname()
	if host == "" {
		host = DefaultHost
	}
	port := u.Port()
	if port == "" {
		port = DefaultPort
	}
	return net.JoinHostPort(host, port)
}

// pingModelsAPI GET {baseURL}/models 探测 API 存活。
// 返回 (是否可达, 失败原因)；传输错误或非 2xx 响应均视为不可达。
func pingModelsAPI(baseURL string, timeout time.Duration) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	endpoint := strings.TrimRight(baseURL, "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err.Error()
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return true, ""
}

// ClassifyModelCapability 按模型 ID 关键词推断能力归类（capability 字段为空时使用）。
// 匹配顺序按优先级排列，避免关键词互相覆盖：
//   - rerank（如 bge-reranker 必须先命中 rerank，再判断 embedding）
//   - bge / qwen3-embedding / embed → embedding
//   - paddleocr / ocr → ocr
//   - mineru / parse → parse
//   - sherpa / whisper / asr → asr
//   - tts / vox / cosy → tts
//   - image / zimage / flux → imagegen
//   - mt / translate / 翻译 → translation
//   - qwen / llama / gemma / hermes → chat
//
// 其余模型返回空字符串（未识别，不归入任何能力）。
func ClassifyModelCapability(modelID string) string {
	l := strings.ToLower(strings.TrimSpace(modelID))
	switch {
	case strings.Contains(l, "rerank"):
		return "rerank"
	case strings.Contains(l, "bge"), strings.Contains(l, "qwen3-embedding"), strings.Contains(l, "embed"):
		return "embedding"
	case strings.Contains(l, "paddleocr"), strings.Contains(l, "ocr"):
		return "ocr"
	case strings.Contains(l, "mineru"), strings.Contains(l, "parse"):
		return "parse"
	case strings.Contains(l, "sherpa"), strings.Contains(l, "whisper"), strings.Contains(l, "asr"):
		return "asr"
	case strings.Contains(l, "tts"), strings.Contains(l, "vox"), strings.Contains(l, "cosy"):
		return "tts"
	case strings.Contains(l, "image"), strings.Contains(l, "zimage"), strings.Contains(l, "flux"):
		return "imagegen"
	case strings.Contains(l, "mt"), strings.Contains(l, "translate"), strings.Contains(l, "翻译"):
		return "translation"
	case strings.Contains(l, "qwen"), strings.Contains(l, "llama"), strings.Contains(l, "gemma"), strings.Contains(l, "hermes"):
		return "chat"
	}
	return ""
}

// buildSummary 汇总人类可读的中文问题清单（全部就绪时为空）。
func buildSummary(addr string, r HealthResult) []string {
	var out []string
	if !r.PortOpen {
		out = append(out, fmt.Sprintf("Herdsman 端口 %s 未监听", addr))
	}
	if r.PortOpen && !r.APIReachable {
		out = append(out, fmt.Sprintf("Herdsman API 不可达（%s）", r.APIError))
	}
	for _, k := range capabilityKeys {
		if !r.Capabilities[k] {
			out = append(out, fmt.Sprintf("未发现可用 %s 模型", capabilityLabels[k]))
		}
	}
	return out
}
