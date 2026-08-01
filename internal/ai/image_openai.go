package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
	"io"
	"log/slog"
	"net/http"
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

// GenerateImage 通过 OpenAI 兼容 API 生成图片
func (b *OpenAIImageBackend) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	endpoint := b.baseURL + "/images/generations"

	body, err := json.Marshal(req)
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

	return &imgResp, nil
}
