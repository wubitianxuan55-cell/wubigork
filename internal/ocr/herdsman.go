// Package ocr 提供 Herdsman 的 OpenAI 兼容 OCR / 文档解析客户端。
//
// 端点：
//   - POST /v1/ocr           PaddleOCR PP-OCRv5 文本检测与识别
//   - POST /v1/documents/parse MinerU Pipeline/Hybrid 文档解析
package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/netclient"
)

const (
	// DefaultOCRModel 是 Herdsman /v1/ocr 的默认模型。
	DefaultOCRModel = "paddleocr-ppocrv5-server"
	// DefaultParseModel 是 Herdsman /v1/documents/parse 的默认模型。
	DefaultParseModel = "minerU"
)

// Client 是 Herdsman OCR/文档解析客户端。
type Client struct {
	baseURL string
	model   string
	client  *http.Client
}

// New 创建客户端。baseURL 可以带 /v1 也可以不带；model 为空时使用默认 OCR 模型。
func New(baseURL, model string) *Client {
	if strings.TrimSpace(model) == "" {
		model = DefaultOCRModel
	}
	return &Client{
		baseURL: normalizeBaseURL(baseURL),
		model:   model,
		client:  netclient.NewSimpleClient(180 * time.Second),
	}
}

// SetModel 动态切换模型名。
func (c *Client) SetModel(model string) {
	if strings.TrimSpace(model) != "" {
		c.model = model
	}
}

// Line 是单行 OCR 结果。
type Line struct {
	Text  string      `json:"text"`
	Score float64     `json:"score"`
	Box   [][]float64 `json:"box"`
}

// Result 是 POST /v1/ocr 的响应。
type Result struct {
	Text        string `json:"text"`
	Lines       []Line `json:"lines"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
	ElapsedMS   int64  `json:"elapsed_ms"`
}

// RecognizeImageFile 读取本地图片并调用 /v1/ocr。
func (c *Client) RecognizeImageFile(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ocr: 读取图片失败: %w", err)
	}
	return c.RecognizeImageBytes(data, mimeByExt(path))
}

// RecognizeImageBytes 把图片字节编码为 data URI 后调用 /v1/ocr。
func (c *Client) RecognizeImageBytes(data []byte, mimeType string) (*Result, error) {
	if mimeType == "" {
		mimeType = "image/png"
	}
	imageBase64 := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
	return c.RecognizeImageBase64(imageBase64)
}

// RecognizeImageBase64 调用 /v1/ocr。imageBase64 支持纯 base64 或 data URI。
func (c *Client) RecognizeImageBase64(imageBase64 string) (*Result, error) {
	body, err := json.Marshal(map[string]any{
		"model":        c.model,
		"image_base64": imageBase64,
	})
	if err != nil {
		return nil, fmt.Errorf("ocr: marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint("/ocr"), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ocr: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ocr: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("ocr: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var result Result
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("ocr: parse: %w", err)
	}
	return &result, nil
}

// ParseOptions 是 POST /v1/documents/parse 的请求参数。
type ParseOptions struct {
	Model     string // 默认 minerU
	Path      string // 本地文件路径（JSON 请求必填）
	Mode      string // pipeline | hybrid，默认 pipeline
	Format    string // json | text | markdown | md，默认 json
	DPI       int    // PDF 渲染 DPI，默认 200
	Formula   bool   // Pipeline 公式识别，默认 true
	Effort    string // Hybrid 推理强度，low | medium | high，默认 medium
	MaxTokens int    // Hybrid 最大生成 token 数，默认 2048
}

// ParsePage 是文档解析结果中的单页。
type ParsePage struct {
	PageNumber int    `json:"page_number"`
	Text       string `json:"text"`
}

// ParseResult 是 POST /v1/documents/parse 的 JSON 响应。
type ParseResult struct {
	Model    string      `json:"model"`
	Text     string      `json:"text"`
	Markdown string      `json:"markdown"`
	Pages    []ParsePage `json:"pages"`
	Metadata struct {
		PageCount     int    `json:"page_count"`
		ElapsedMS     int64  `json:"elapsed_ms"`
		OCREnabled    bool   `json:"ocr_enabled"`
		Runtime       string `json:"runtime"`
		InputFormat   string `json:"input_format"`
		Parser        string `json:"parser"`
		OCRImageCount int    `json:"ocr_image_count"`
	} `json:"metadata"`
}

// ParseDocument 调用 /v1/documents/parse，返回结构化结果。
// 当 Format 为 text/markdown/md 时，服务返回纯文本，Text 会保存该内容。
func (c *Client) ParseDocument(opts ParseOptions) (*ParseResult, error) {
	if strings.TrimSpace(opts.Path) == "" {
		return nil, fmt.Errorf("ocr: parse path 不能为空")
	}
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = DefaultParseModel
	}
	mode := strings.TrimSpace(opts.Mode)
	if mode == "" {
		mode = "pipeline"
	}
	format := strings.TrimSpace(opts.Format)
	if format == "" {
		format = "json"
	}
	dpi := opts.DPI
	if dpi <= 0 {
		dpi = 200
	}

	payload := map[string]any{
		"model":  model,
		"path":   opts.Path,
		"mode":   mode,
		"format": format,
		"dpi":    dpi,
	}
	if mode == "pipeline" && format == "json" {
		payload["formula"] = opts.Formula
	}
	if mode == "hybrid" {
		effort := strings.TrimSpace(opts.Effort)
		if effort == "" {
			effort = "medium"
		}
		payload["effort"] = effort
		maxTokens := opts.MaxTokens
		if maxTokens <= 0 {
			maxTokens = 2048
		}
		payload["max_tokens"] = maxTokens
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ocr: parse marshal: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, c.endpoint("/documents/parse"), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ocr: parse create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ocr: parse: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("ocr: parse read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ocr: parse HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	trimmed := strings.TrimSpace(string(raw))
	if format != "json" {
		return &ParseResult{Model: model, Text: trimmed, Markdown: trimmed}, nil
	}
	var out ParseResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("ocr: parse response: %w", err)
	}
	return &out, nil
}

func (c *Client) endpoint(path string) string {
	return strings.TrimRight(c.baseURL, "/") + path
}

func normalizeBaseURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "http://localhost:8080"
	}
	base = strings.TrimSuffix(base, "/v1")
	return base + "/v1"
}

func mimeByExt(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/png"
	}
}
