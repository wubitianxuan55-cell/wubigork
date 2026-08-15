package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OpenAIImageBackend 通用 OpenAI 兼容图片生成后端
// 支持任何提供 /v1/images/generations 端点的服务（Herdsman、Ollama 等）
type OpenAIImageBackend struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewOpenAIImageBackend 创建 OpenAI 兼容图片后端
// baseURL 应包含 /v1 后缀（如 http://localhost:8080/v1）
// apiKey 为空表示无需认证
func NewOpenAIImageBackend(baseURL string, apiKey string) *OpenAIImageBackend {
	return &OpenAIImageBackend{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: netclient.NewSimpleClient(10 * time.Minute),
	}
}

// init 自注册：OpenAI 兼容后端经注册表提供（kind = ImageBackendKindOpenAI）。
// 覆盖 xAI / Herdsman / Ollama 等提供 /v1/images/generations 的服务。
func init() {
	RegisterImageBackend(ImageBackendKindOpenAI, func(cfg ImageBackendConfig) (ImageBackend, error) {
		if strings.TrimSpace(cfg.BaseURL) == "" {
			return nil, fmt.Errorf("ai: openai image backend requires base_url")
		}
		return NewOpenAIImageBackend(cfg.BaseURL, cfg.APIKey), nil
	})
}

// GenerateImage 通过 OpenAI 兼容 API 生成图片
func (b *OpenAIImageBackend) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	var endpoint string
	var body []byte
	var err error
	if req.Mode == "img2img" {
		// Herdsman 图生图：JSON 请求，image 字段为参考图 base64 data URL
		endpoint = b.baseURL + "/images/img2img"
		body, err = b.buildImg2ImgBody(req)
	} else {
		endpoint = b.baseURL + "/images/generations"
		body, err = json.Marshal(req)
	}
	if err != nil {
		return nil, fmt.Errorf("marshal image request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造图片请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if b.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
	}

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("图片 API 请求失败 (%s): %w", b.baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取图片响应失败: %w", err)
	}

	if resp.StatusCode != 200 {
		slog.Error("图片生成失败", "backend", b.baseURL, "status", resp.StatusCode, "body", trimStr(string(respBody), 500))
		return nil, fmt.Errorf("图片 API 错误 (HTTP %d): %s", resp.StatusCode, trimStr(string(respBody), 500))
	}

	var imgResp ImageGenerationResponse
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		slog.Error("解析图片响应失败", "backend", b.baseURL, "body", trimStr(string(respBody), 300), "error", err)
		return nil, fmt.Errorf("解析图片响应失败: %w", err)
	}

	// 服务端可能返回 url（含相对路径，如 /v1/images/cache/xxx.png）而不是 b64_json：
	// 统一下载并转成 data URL，保证前端可显示、可落盘、历史可复用。
	for i := range imgResp.Data {
		if imgResp.Data[i].B64JSON != "" || strings.HasPrefix(imgResp.Data[i].URL, "data:") {
			continue
		}
		rawURL := strings.TrimSpace(imgResp.Data[i].URL)
		if rawURL == "" {
			continue
		}
		if strings.HasPrefix(rawURL, "/") {
			if base, err := url.Parse(b.baseURL); err == nil {
				if rel, err := url.Parse(rawURL); err == nil {
					rawURL = base.ResolveReference(rel).String()
				}
			}
		}
		dataURL, err := b.fetchToDataURL(ctx, rawURL)
		if err != nil {
			slog.Warn("图片 URL 下载失败，保留原始 URL", "backend", b.baseURL, "url", rawURL, "error", err)
			continue
		}
		imgResp.Data[i].B64JSON = dataURL
		imgResp.Data[i].URL = ""
	}

	return &imgResp, nil
}

// img2imgRequest Herdsman /v1/images/img2img 请求体（JSON，image 为参考图 base64）
type img2imgRequest struct {
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt"`
	Image  string `json:"image"`
	N      int    `json:"n,omitempty"`
	Size   string `json:"size,omitempty"`
}

// buildImg2ImgBody 构造图生图 JSON 请求体
func (b *OpenAIImageBackend) buildImg2ImgBody(req *ImageGenerationRequest) ([]byte, error) {
	if strings.TrimSpace(req.InitImage) == "" {
		return nil, fmt.Errorf("图生图需要提供参考图")
	}
	return json.Marshal(img2imgRequest{
		Model:  req.Model,
		Prompt: req.Prompt,
		Image:  req.InitImage,
		N:      req.N,
		Size:   req.Size,
	})
}

// fetchToDataURL 下载图片/视频并转为 data URL
func (b *OpenAIImageBackend) fetchToDataURL(ctx context.Context, rawURL string) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("构造下载请求失败: %w", err)
	}
	if b.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
	}
	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("下载图片失败 (%s): %w", rawURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("下载图片 HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取图片数据失败: %w", err)
	}
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || strings.HasPrefix(mimeType, "text/") {
		mimeType = http.DetectContentType(data)
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
