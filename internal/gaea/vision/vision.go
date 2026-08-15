// Package vision 提供办公板块的本地图片识别（识图）能力。
// 默认调用本机 herdsman 的 OpenAI 兼容视觉端点（与 ds-vision-skill custom-1
// 使用同一本地模型）；端点/模型/后端由 config 驱动（SetVisionRuntime 注入），
// 不再读环境变量。
package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 默认本地视觉端点与模型（本机 herdsman，Qwen3.6 视觉模型）。
const (
	DefaultBaseURL = "http://127.0.0.1:8080/v1"
	DefaultModel   = "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2"
)

// ── VisionProvider seam（3.0 Step 3d #4：端点硬编码/环境变量 → 注册表 + config）──
// 范式见 internal/gaea/provider/provider.go 与 internal/ai/image_backend.go 的
// Register/New/Kinds。消费者（RecognizeImage）只依赖 Provider 接口与 config
// 驱动的 kind；切换视觉后端只改配置、代码零改动。

// Provider 本地图片识别能力接口（OpenAI 兼容 /v1/chat/completions 视觉端点）。
type Provider interface {
	// Recognize 识别图片文件内容，返回文本描述。
	Recognize(ctx context.Context, imagePath, prompt string) (string, error)
}

// VisionProviderKindOpenAI OpenAI 兼容视觉后端 kind（覆盖 Herdsman/Ollama
// 等 /v1/chat/completions 视觉模型服务）。
const VisionProviderKindOpenAI = "openai"

// VisionProviderConfig 是视觉后端实例配置（注册表 New 入参）。
type VisionProviderConfig struct {
	BaseURL string // API 地址（如 "http://127.0.0.1:8080/v1"）
	Model   string // 视觉模型名
}

// VisionProviderFactory 按实例配置构建视觉后端（kind → 实例）。
type VisionProviderFactory func(cfg VisionProviderConfig) (Provider, error)

// visionProviderRegistry kind → 工厂注册表。各实现 init() 自注册；互斥注册，
// 重复即 panic（编译期接线错误）。
var visionProviderRegistry = map[string]VisionProviderFactory{}

func init() {
	RegisterVisionProvider(VisionProviderKindOpenAI, func(cfg VisionProviderConfig) (Provider, error) {
		return &openAIProvider{baseURL: cfg.BaseURL, model: cfg.Model}, nil
	})
}

// RegisterVisionProvider 注册视觉后端 kind（如 "openai"）。供各实现 init()
// 自注册；kind 为空或重复注册直接 panic。
func RegisterVisionProvider(kind string, factory VisionProviderFactory) {
	if kind == "" {
		panic("vision: provider kind must not be empty")
	}
	if _, dup := visionProviderRegistry[kind]; dup {
		panic("vision: duplicate provider kind " + kind)
	}
	visionProviderRegistry[kind] = factory
}

// NewVisionProvider 按 kind 经注册表构建视觉后端；未知 kind 返回错误
// （fail-closed，附已注册 kind 列表）。
func NewVisionProvider(kind string, cfg VisionProviderConfig) (Provider, error) {
	factory, ok := visionProviderRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("vision: unknown provider kind %q (registered: %v)", kind, VisionProviderKinds())
	}
	p, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("vision: provider factory %q returned nil", kind)
	}
	return p, nil
}

// VisionProviderKinds 返回已注册视觉后端 kind 列表（排序，供诊断/校验）。
func VisionProviderKinds() []string {
	out := make([]string, 0, len(visionProviderRegistry))
	for k := range visionProviderRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// openAIProvider OpenAI 兼容视觉端点实现（POST /v1/chat/completions，image_url
// base64 data URL）。
type openAIProvider struct {
	baseURL string
	model   string
}

func (p *openAIProvider) Recognize(ctx context.Context, imagePath, prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		prompt = "请详细描述这张图片的内容，包括所有可见文字、布局和关键细节。"
	}
	return RecognizeImageAt(ctx, p.baseURL, p.model, imagePath, prompt, 90*time.Second)
}

// ── 运行时配置注入 ─────────────────────────────────────────────

// VisionRuntime 是视觉后端的运行时配置，由 boot 从 gaea.toml 注入。
// 零值 = 全默认（kind=openai，DefaultBaseURL + DefaultModel）。
type VisionRuntime struct {
	Kind    string // 注册表 kind（默认 VisionProviderKindOpenAI）
	BaseURL string // API 地址（默认 DefaultBaseURL）
	Model   string // 视觉模型（默认 DefaultModel）
}

// visionRuntime 保存 SetVisionRuntime 注入的视觉后端配置。
var visionRuntime VisionRuntime

// SetVisionRuntime 注入视觉后端配置（boot 装配调用）。
// 切换视觉后端只改配置（kind/base/model），消费方代码零改动。
func SetVisionRuntime(cfg VisionRuntime) { visionRuntime = cfg }

// resolveVisionRuntime 返回生效的视觉后端配置：注入值优先，空字段回落默认
// （等价旧 GAEA_VISION_BASE_URL/MODEL 缺省行为）。
func resolveVisionRuntime() VisionRuntime {
	r := visionRuntime
	if r.Kind == "" {
		r.Kind = VisionProviderKindOpenAI
	}
	if r.BaseURL == "" {
		r.BaseURL = DefaultBaseURL
	}
	if r.Model == "" {
		r.Model = DefaultModel
	}
	return r
}

// RecognizeImage 识别本地图片文件内容，返回文本描述。
// 后端由 config 驱动（SetVisionRuntime 注入），未注入时回落默认值。
func RecognizeImage(ctx context.Context, imagePath, prompt string) (string, error) {
	r := resolveVisionRuntime()
	p, err := NewVisionProvider(r.Kind, VisionProviderConfig{BaseURL: r.BaseURL, Model: r.Model})
	if err != nil {
		// 未知 kind fail-closed：拒绝运行而非静默降级。
		return "", err
	}
	return p.Recognize(ctx, imagePath, prompt)
}

// RecognizeImageAt 向指定 OpenAI 兼容端点发起视觉识别请求（可测试注入）。
func RecognizeImageAt(ctx context.Context, baseURL, model, imagePath, prompt string, timeout time.Duration) (string, error) {
	dataURL, err := imageDataURL(imagePath)
	if err != nil {
		return "", fmt.Errorf("识图：读取图片失败: %w", err)
	}
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": prompt},
					{"type": "image_url", "image_url": map[string]string{"url": dataURL}},
				},
			},
		},
		"max_tokens":  1024,
		"temperature": 0.2,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("识图：构造请求失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("识图：创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("识图：本地视觉服务不可用: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("识图：读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("识图：视觉服务返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("识图：解析响应失败: %w", err)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("识图：模型未返回内容")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

// imageDataURL 把本地图片转成 base64 data URL（供 OpenAI 兼容 image_url 使用）。
func imageDataURL(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mime := mimeByExt(path)
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
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

