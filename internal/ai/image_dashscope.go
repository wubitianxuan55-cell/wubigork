package ai

// image_dashscope.go — 阿里云百炼（DashScope）改图后端，qwen-image-edit 系列。
// 官方同步 HTTP 端点（无任务轮询）：POST {base}/api/v1/services/aigc/
// multimodal-generation/generation，Bearer 认证。请求体官方 schema 仅
// model/input.messages/parameters（n/watermark/negative_prompt/size）：
// content 数组 image 1-3 张 + text 恰好 1 个，这里改图固定 1 图 1 文。
// 响应 output.choices[].message.content[] 中含 "image" 字段的项为结果图
// URL（24h 有效）→ 统一下载转 data URL（同 GLM 后端，前端可显示可落盘）。
// 错误体官方扁平形态 {"code","message"} 原样透出。仅支持 img2img——
// txt2img/t2v 诚实报错，不静默降级。

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/netclient"
)

// ImageBackendKindDashScope 阿里云百炼改图后端 kind（常量锚在 image_backend.go）。

// DashScopeBaseURL 百炼 API 根地址（官方默认；TrimSuffix / 后拼端点路径）。
const DashScopeBaseURL = "https://dashscope.aliyuncs.com"

// DashScopeDefaultImageModel 改图缺省模型（官方 qwen-image-edit-plus；
// 另支持 qwen-image-edit / qwen-image-edit-max / qwen-image-2.0 系列）。
const DashScopeDefaultImageModel = "qwen-image-edit-plus"

// DashScopeMaxInputImageBytes 输入图官方上限 10MB（base64 前的原始字节）。
const DashScopeMaxInputImageBytes = 10 << 20

// DashScopeImageBackend 阿里云百炼改图后端。
type DashScopeImageBackend struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewDashScopeImageBackend 创建百炼图片后端。baseURL 为 API 根地址
// （默认 DashScopeBaseURL），apiKey 为百炼 API Key（必填）。
func NewDashScopeImageBackend(baseURL, apiKey string) *DashScopeImageBackend {
	return &DashScopeImageBackend{
		baseURL:    strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: netclient.NewSimpleClient(10 * time.Minute),
	}
}

// init 自注册：百炼后端经注册表提供（kind = ImageBackendKindDashScope）。
func init() {
	RegisterImageBackend(ImageBackendKindDashScope, func(cfg ImageBackendConfig) (ImageBackend, error) {
		if strings.TrimSpace(cfg.BaseURL) == "" {
			return nil, fmt.Errorf("ai: dashscope image backend requires base_url")
		}
		if strings.TrimSpace(cfg.APIKey) == "" {
			return nil, fmt.Errorf("ai: dashscope image backend requires api_key")
		}
		return NewDashScopeImageBackend(cfg.BaseURL, cfg.APIKey), nil
	})
}

// dashscopeContentPart 多模态 content 项：官方每项只有 image 或 text 一个键。
type dashscopeContentPart struct {
	Image string `json:"image,omitempty"`
	Text  string `json:"text,omitempty"`
}

// dashscopeImageRequest 官方 schema 内的请求体（多余字段一律不发）。
type dashscopeImageRequest struct {
	Model      string               `json:"model"`
	Input      dashscopeImageInput  `json:"input"`
	Parameters dashscopeImageParams `json:"parameters"`
}

type dashscopeImageInput struct {
	Messages []dashscopeImageMessage `json:"messages"`
}

type dashscopeImageMessage struct {
	Role    string                 `json:"role"`
	Content []dashscopeContentPart `json:"content"`
}

// dashscopeImageParams parameters 官方字段：n/watermark 恒发；
// negative_prompt（≤500 字）/size（改图默认不传，保持原图比例）非空才发。
type dashscopeImageParams struct {
	N              int    `json:"n"`
	Watermark      bool   `json:"watermark"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Size           string `json:"size,omitempty"`
}

// dashscopeImageResponse 响应：output.choices[].message.content[] 中
// 含 image 键的项 = 结果图 URL；错误时顶层 {"code","message"}。
type dashscopeImageResponse struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content []dashscopeContentPart `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// GenerateImage 调百炼多模态生成端点改图（仅 img2img）。
func (b *DashScopeImageBackend) GenerateImage(ctx context.Context, req *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	if req.Mode != "img2img" && strings.TrimSpace(req.InitImage) == "" {
		// txt2img/t2v 一律诚实报错，不静默降级。
		return nil, fmt.Errorf("百炼后端仅支持改图（img2img）")
	}
	initImage := strings.TrimSpace(req.InitImage)
	if initImage == "" {
		return nil, fmt.Errorf("百炼改图需要参考图（InitImage 为空）")
	}
	if raw, ok := dataURLByteSize(initImage); ok && raw > DashScopeMaxInputImageBytes {
		return nil, fmt.Errorf("参考图超过百炼 10MB 输入限制（当前约 %dMB），请压缩后再试", raw>>20)
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("百炼改图需要编辑指令（text 恰好 1 个）")
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = DashScopeDefaultImageModel
	}
	n := req.N
	if n < 1 {
		n = 1
	}
	params := dashscopeImageParams{N: n, Watermark: false}
	if neg := strings.TrimSpace(req.Negative); neg != "" {
		params.NegativePrompt = neg
	}
	if size := strings.TrimSpace(req.Size); size != "" {
		// 官方 size 为星号格式（1024*1536，16 的倍数）；gaea 内部 1024x1024 → 转换。
		params.Size = strings.Replace(size, "x", "*", 1)
	}
	body, err := json.Marshal(dashscopeImageRequest{
		Model: model,
		Input: dashscopeImageInput{
			Messages: []dashscopeImageMessage{{
				Role: "user",
				Content: []dashscopeContentPart{
					{Image: initImage},
					{Text: prompt},
				},
			}},
		},
		Parameters: params,
	})
	if err != nil {
		return nil, fmt.Errorf("构造百炼图片请求失败: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.baseURL+"/api/v1/services/aigc/multimodal-generation/generation", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建百炼图片请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)

	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("百炼图片 API 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("读取百炼图片响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		slog.Error("百炼生图失败", "status", resp.StatusCode, "body", trimStr(string(respBody), 500))
		if msg := dashscopeErrorBodyMessage(respBody); msg != "" {
			return nil, fmt.Errorf("百炼生图错误（HTTP %d）：%s", resp.StatusCode, msg)
		}
		return nil, fmt.Errorf("百炼生图错误（HTTP %d）", resp.StatusCode)
	}

	var parsed dashscopeImageResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		slog.Error("解析百炼图片响应失败", "body", trimStr(string(respBody), 300), "error", err)
		return nil, fmt.Errorf("解析百炼图片响应失败: %w", err)
	}
	// 遍历 content[] 取 image URL（与 text 项并存，顺序不敏感）。
	var urls []string
	for _, choice := range parsed.Output.Choices {
		for _, part := range choice.Message.Content {
			if u := strings.TrimSpace(part.Image); u != "" {
				urls = append(urls, u)
			}
		}
	}
	if len(urls) == 0 {
		// 200 但无图：官方 content_filter / code+message 错误形态——诚实提示。
		if parsed.Code != "" || parsed.Message != "" {
			return nil, fmt.Errorf("百炼未返回图片 [%s]：%s", parsed.Code, parsed.Message)
		}
		return nil, fmt.Errorf("百炼未返回图片（可能触发内容审核，请调整提示词）")
	}
	// 结果 URL 24h 有效：统一下载转 data URL（公网 CDN，不带认证头），
	// 下载失败保留原始 URL（同 GLM 后端口径，不让单图失败毁掉整次生成）。
	out := &ImageGenerationResponse{Created: time.Now().Unix()}
	for _, u := range urls {
		dataURL, err := fetchToDataURL(ctx, b.httpClient, u, "")
		if err != nil {
			slog.Warn("百炼图片 URL 下载失败，保留原始 URL", "url", u, "error", err)
			out.Data = append(out.Data, ImageData{URL: u})
			continue
		}
		out.Data = append(out.Data, ImageData{B64JSON: dataURL})
	}
	return out, nil
}

// dashscopeErrorBodyMessage 解析百炼错误体 {"code","message"}（官方扁平形态）。
func dashscopeErrorBodyMessage(body []byte) string {
	var parsed dashscopeImageResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	if parsed.Message == "" {
		return ""
	}
	if parsed.Code != "" {
		return fmt.Sprintf("[%s] %s", parsed.Code, parsed.Message)
	}
	return parsed.Message
}

// dataURLByteSize 解码 data URL 并返回原始字节长度（非 data URL 返回 false；
// 仅用于输入图 10MB 上限的 fail-fast 校验，避免把超大 base64 上传后才发现超限）。
func dataURLByteSize(s string) (int, bool) {
	if !strings.HasPrefix(s, "data:") {
		return 0, false
	}
	comma := strings.Index(s, ",")
	if comma < 0 {
		return 0, false
	}
	raw, err := base64.StdEncoding.DecodeString(s[comma+1:])
	if err != nil {
		return 0, false
	}
	return len(raw), true
}
