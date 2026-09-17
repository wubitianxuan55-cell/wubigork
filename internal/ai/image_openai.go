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
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
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
	if req.Mode == "edit" {
		return b.editImage(ctx, req)
	}
	var endpoint string
	var body []byte
	var err error
	if req.Mode == "img2img" {
		// Herdsman 图生图：JSON 请求，image 字段为参考图 base64 data URL
		endpoint = b.baseURL + "/images/img2img"
		// T2 参考槽 v0：未显式给 InitImage 时，取第一张参考图作图生图种子。
		r := *req
		if strings.TrimSpace(r.InitImage) == "" && len(req.RefImages) > 0 {
			r.InitImage = req.RefImages[0]
		}
		body, err = b.buildImg2ImgBody(&r)
	} else {
		if len(req.RefImages) > 0 {
			return nil, fmt.Errorf("该后端文生图端点不支持参考图（参考槽当前仅图生图可用）：%s", req.Model)
		}
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
	return b.parseImageResponse(ctx, respBody, resp.StatusCode)
}

// editImage 指令编辑（阶段一刀 C，规格 进度计划/gaea-instruct-edit-20260917.md）：
// multipart POST /images/edits（OpenAI 标准编辑端点；Qwen-Image-Edit 系云端
// 网关的暴露面）。InitImage=原图 data URL、Prompt=人话指令（Q1 复用既有字段）。
func (b *OpenAIImageBackend) editImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	imgBytes, ext, err := decodeDataURLBytes(req.InitImage)
	if err != nil {
		return nil, fmt.Errorf("指令编辑需要原图（data URL）：%w", err)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", req.Model)
	_ = w.WriteField("prompt", req.Prompt)
	if req.N > 0 {
		_ = w.WriteField("n", strconv.Itoa(req.N))
	}
	if req.Size != "" {
		_ = w.WriteField("size", req.Size)
	}
	fw, err := w.CreateFormFile("image", "source."+ext)
	if err != nil {
		return nil, fmt.Errorf("构造编辑请求失败: %w", err)
	}
	if _, err := fw.Write(imgBytes); err != nil {
		return nil, fmt.Errorf("构造编辑请求失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("构造编辑请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.baseURL+"/images/edits", &buf)
	if err != nil {
		return nil, fmt.Errorf("构造图片请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", w.FormDataContentType())
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
	return b.parseImageResponse(ctx, respBody, resp.StatusCode)
}

// parseImageResponse 响应解析共用（generation/img2img/edit 三分支同构）：
// JSON 解析 + url（含相对路径）统一下载转 data URL（原 GenerateImage 内联
// 逻辑原样提取，行为零变化）。
func (b *OpenAIImageBackend) parseImageResponse(ctx context.Context, respBody []byte, status int) (*ImageGenerationResponse, error) {
	if status != 200 {
		slog.Error("图片生成失败", "backend", b.baseURL, "status", status, "body", trimStr(string(respBody), 500))
		return nil, fmt.Errorf("图片 API 错误 (HTTP %d): %s", status, trimStr(string(respBody), 500))
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

// decodeDataURLBytes data URL → (字节, 扩展名)。MIME 按常见图片映射，未知
// 缺省 png（编辑端点宽容）；非 data URL 报错（编辑原图必须是 data URL——
// 与 img2img 的 InitImage 口径一致）。
func decodeDataURLBytes(dataURL string) ([]byte, string, error) {
	s := strings.TrimSpace(dataURL)
	if !strings.HasPrefix(s, "data:") {
		return nil, "", fmt.Errorf("原图须为 data URL")
	}
	comma := strings.Index(s, ",")
	if comma < 0 {
		return nil, "", fmt.Errorf("data URL 缺少负载")
	}
	mime := strings.TrimPrefix(strings.SplitN(s[5:comma], ";", 2)[0], "image/")
	b, err := base64.StdEncoding.DecodeString(s[comma+1:])
	if err != nil {
		return nil, "", fmt.Errorf("data URL base64 解码失败: %w", err)
	}
	switch mime {
	case "jpeg", "jpg":
		return b, "jpg", nil
	case "webp":
		return b, "webp", nil
	case "gif":
		return b, "gif", nil
	case "png", "":
		return b, "png", nil
	default:
		return b, "png", nil
	}
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

// fetchToDataURL 下载图片/视频并转为 data URL（openai 兼容后端专属封装，
// 自动附带实例认证头；共享实现在包级 fetchToDataURL，GLM 后端同样复用）。
func (b *OpenAIImageBackend) fetchToDataURL(ctx context.Context, rawURL string) (string, error) {
	return fetchToDataURL(ctx, b.httpClient, rawURL, b.apiKey)
}

// fetchToDataURL 下载图片/视频并转为 data URL。bearer 非空时附带认证头
// （部分服务如 Herdsman 的缓存 URL 需鉴权；公有 CDN 传空）。
func fetchToDataURL(ctx context.Context, client *http.Client, rawURL, bearer string) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("构造下载请求失败: %w", err)
	}
	if bearer != "" {
		httpReq.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := client.Do(httpReq)
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
