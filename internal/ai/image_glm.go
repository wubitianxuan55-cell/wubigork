package ai

// image_glm.go — 智谱 GLM 图片生成后端（官方 docs.bigmodel.cn「图像生成」API）。
// 端点 POST {base}/images/generations，HTTP Bearer 认证。官方请求 schema 仅
// model/prompt/size/quality/watermark_enabled/user_id：不接受 response_format
// （响应只回 URL）、negative/seed 等 OpenAI 扩展字段，故不复用通用
// OpenAIImageBackend（其序列化整个 ImageGenerationRequest），这里只发官方
// 字段、其余诚实忽略。错误体官方形态 {"error":{"code","message"}} 原样透出。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/netclient"
)

// ImageBackendKindGLM 智谱 GLM 云端图片后端（/api/paas/v4/images/generations）。
const ImageBackendKindGLM = "glm"

// GLMDefaultImageModel 生图缺省模型：cogview-4 锁定版（官方图像生成 API 枚举；
// glm-image 仅 hd 档约 20s，cogview-4 standard 约 5–10s 更适合默认交互节奏）。
const GLMDefaultImageModel = "cogview-4-250304"

// GLMImageBackend 智谱图片生成后端。
type GLMImageBackend struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewGLMImageBackend 创建 GLM 图片后端。baseURL 为引擎预置地址
// （https://open.bigmodel.cn/api/paas/v4），apiKey 为智谱 API Key（必填）。
func NewGLMImageBackend(baseURL, apiKey string) *GLMImageBackend {
	return &GLMImageBackend{
		baseURL:    strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: netclient.NewSimpleClient(10 * time.Minute),
	}
}

// init 自注册：GLM 后端经注册表提供（kind = ImageBackendKindGLM）。
func init() {
	RegisterImageBackend(ImageBackendKindGLM, func(cfg ImageBackendConfig) (ImageBackend, error) {
		if strings.TrimSpace(cfg.BaseURL) == "" {
			return nil, fmt.Errorf("ai: glm image backend requires base_url")
		}
		if strings.TrimSpace(cfg.APIKey) == "" {
			return nil, fmt.Errorf("ai: glm image backend requires api_key")
		}
		return NewGLMImageBackend(cfg.BaseURL, cfg.APIKey), nil
	})
}

// glmImageRequest 官方 schema 内的请求体（多余字段一律不发）。
type glmImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
}

// GenerateImage 调智谱 /images/generations 生成图片。
func (b *GLMImageBackend) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	if req.Mode == "img2img" || strings.TrimSpace(req.InitImage) != "" || len(req.RefImages) > 0 {
		// 官方端点无图生图参数——诚实报错，不静默丢弃参考图。
		return nil, fmt.Errorf("GLM 生图端点仅支持文生图（官方 images/generations 无图生图/参考图参数）")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = GLMDefaultImageModel
	}
	body, err := json.Marshal(glmImageRequest{
		Model:  model,
		Prompt: req.Prompt,
		Size:   strings.TrimSpace(req.Size),
	})
	if err != nil {
		return nil, fmt.Errorf("构造 GLM 图片请求失败: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 GLM 图片请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GLM 图片 API 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("读取 GLM 图片响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		slog.Error("GLM 生图失败", "status", resp.StatusCode, "body", trimStr(string(respBody), 500))
		if msg := zhipuErrorBodyMessage(respBody); msg != "" {
			return nil, fmt.Errorf("GLM 生图错误（HTTP %d）：%s", resp.StatusCode, msg)
		}
		return nil, fmt.Errorf("GLM 生图错误（HTTP %d）：", resp.StatusCode)
	}

	var imgResp ImageGenerationResponse
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		slog.Error("解析 GLM 图片响应失败", "body", trimStr(string(respBody), 300), "error", err)
		return nil, fmt.Errorf("解析 GLM 图片响应失败: %w", err)
	}
	if len(imgResp.Data) == 0 {
		// 200 但无图：官方 content_filter 拦截等场景——诚实提示而非空成功。
		return nil, fmt.Errorf("GLM 未返回图片（可能触发内容审核，请调整提示词）")
	}
	// 官方只回 URL（30 天有效）：统一下载转 data URL，保证前端可显示、可落盘。
	for i := range imgResp.Data {
		rawURL := strings.TrimSpace(imgResp.Data[i].URL)
		if rawURL == "" || imgResp.Data[i].B64JSON != "" || strings.HasPrefix(rawURL, "data:") {
			continue
		}
		dataURL, err := fetchToDataURL(ctx, b.httpClient, rawURL, "")
		if err != nil {
			slog.Warn("GLM 图片 URL 下载失败，保留原始 URL", "url", rawURL, "error", err)
			continue
		}
		imgResp.Data[i].B64JSON = dataURL
		imgResp.Data[i].URL = ""
	}
	return &imgResp, nil
}

// zhipuErrorBodyMessage 解析智谱错误体 {"error":{"code","message"}}（官方形态）。
// ai 包与 modelengine 各有一份：跨包复用会引入依赖环，形态极简两处各自锚定。
func zhipuErrorBodyMessage(body []byte) string {
	var parsed struct {
		Error struct {
			Code    interface{} `json:"code"`
			Message string      `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	if parsed.Error.Message == "" {
		return ""
	}
	// code 官方可能回字符串或数字，两态兼容。
	if code := fmt.Sprint(parsed.Error.Code); code != "" && code != "<nil>" {
		return fmt.Sprintf("[%s] %s", code, parsed.Error.Message)
	}
	return parsed.Error.Message
}
