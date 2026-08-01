// Package asr 语音识别客户端 — 对接 Herdsman ASR API
//
// 对照 Ackem 的 asr_engine.py (faster-whisper) 设计：
//   - HTTP POST /v1/audio/transcriptions → 非流式（短音频/一次性识别）
//   - WebSocket /v1/audio/transcriptions/stream → 流式（实时连续识别）
//
// 与 Ackem 差异：Ackem 使用本地 Python faster-whisper 进程；
// Wubigrok 通过 Herdsman 网关统一调度 ASR 模型。
package asr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// HerdsmanASR 通过 Herdsman 的 OpenAI 兼容 ASR 端点进行语音识别
type HerdsmanASR struct {
	baseURL string
	model   string // sherpa-onnx-streaming-zipformer-zh-14m / whisper-base / funasr
	client  *http.Client
}

// TranscriptionResult ASR 识别结果（对齐 Ackem asr_engine.transcribe 返回格式）
type TranscriptionResult struct {
	Text       string  `json:"text"`       // 识别文本
	Language   string  `json:"language"`   // 语言代码，如 zh
	Duration   float64 `json:"duration"`   // 音频时长（秒）
	Confidence float64 `json:"confidence"` // 置信度（Herdsman 暂不返回，预留）
}

// NewHerdsmanASR 创建 Herdsman ASR 客户端
// model 推荐：whisper-base（非流式通用）、sherpa-onnx-streaming-zipformer-zh-14m（流式中英文）、funasr（流式中文）
func NewHerdsmanASR(baseURL, model string) *HerdsmanASR {
	if model == "" {
		model = "whisper-base"
	}
	return &HerdsmanASR{
		baseURL: baseURL,
		model:   model,
		client:  netclient.NewSimpleClient(60 * time.Second),
	}
}

// SetModel 动态切换 ASR 模型
func (h *HerdsmanASR) SetModel(model string) {
	h.model = model
}

// TranscribeBase64 通过 base64 编码的音频数据进行识别
// audioBase64: 不含前缀的纯 base64 字符串
// mimeType: 音频 MIME 类型，默认 "audio/wav"
func (h *HerdsmanASR) TranscribeBase64(audioBase64, mimeType string) (*TranscriptionResult, error) {
	if mimeType == "" {
		mimeType = "audio/wav"
	}
	body := map[string]interface{}{
		"model":    h.model,
		"audio":    fmt.Sprintf("data:%s;base64,%s", mimeType, audioBase64),
		"language": "zh",
	}
	return h.doTranscribe(body)
}

// TranscribeBytes 通过原始音频字节进行识别（multipart/form-data 上传）
func (h *HerdsmanASR) TranscribeBytes(audioData []byte, filename string) (*TranscriptionResult, error) {
	url := h.baseURL + "/v1/audio/transcriptions"

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("model", h.model)
	_ = writer.WriteField("language", "zh")

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("asr: create form file: %w", err)
	}
	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("asr: write audio: %w", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("asr: create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return h.doRequest(req)
}

// TranscribeRequest 通用转录请求（对齐 Ackem asr_engine 的参数设计）
type TranscribeRequest struct {
	AudioBase64 string // 纯 base64（不含前缀），与 MimeType 配合使用
	MimeType    string // 默认 "audio/wav"
	Language    string // 默认 "zh"
}

// doTranscribe 发送 JSON POST 请求并解析响应
func (h *HerdsmanASR) doTranscribe(body map[string]interface{}) (*TranscriptionResult, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("asr: marshal: %w", err)
	}

	url := h.baseURL + "/v1/audio/transcriptions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("asr: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return h.doRequest(req)
}

// doRequest 执行 HTTP 请求并解析转录结果
func (h *HerdsmanASR) doRequest(req *http.Request) (*TranscriptionResult, error) {
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("asr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("asr: HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var result TranscriptionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("asr: parse: %w", err)
	}
	return &result, nil
}

// ── 对齐 Ackem asr_engine.py 的 normalize 逻辑 ──

// NormalizeTranscription 对 ASR 结果做文本规范化
func NormalizeTranscription(text string) string {
	return trimSpace(text)
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// EncodeBase64 将字节编码为 base64 字符串
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
