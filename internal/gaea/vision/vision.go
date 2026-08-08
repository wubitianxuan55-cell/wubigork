// Package vision 提供办公板块的本地图片识别（识图）能力。
// 默认调用本机 herdsman 的 OpenAI 兼容视觉端点（与 ds-vision-skill custom-1
// 使用同一本地模型），端点和模型可用环境变量覆盖。
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
	"strings"
	"time"
)

// 默认本地视觉端点与模型（本机 herdsman，Qwen3.6 视觉模型）。
const (
	DefaultBaseURL = "http://127.0.0.1:8080/v1"
	DefaultModel   = "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2"
)

// RecognizeImage 识别本地图片文件内容，返回文本描述。
// 端点与模型分别由环境变量 GAEA_VISION_BASE_URL / GAEA_VISION_MODEL 覆盖。
func RecognizeImage(ctx context.Context, imagePath, prompt string) (string, error) {
	baseURL := envOr("GAEA_VISION_BASE_URL", DefaultBaseURL)
	model := envOr("GAEA_VISION_MODEL", DefaultModel)
	if strings.TrimSpace(prompt) == "" {
		prompt = "请详细描述这张图片的内容，包括所有可见文字、布局和关键细节。"
	}
	return RecognizeImageAt(ctx, baseURL, model, imagePath, prompt, 90*time.Second)
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

func envOr(name, fallback string) string {
	if v := os.Getenv(name); strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
