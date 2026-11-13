package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/vision"
)

func init() { tool.RegisterBuiltin(visionTool{}) }

// visionTool 识图工具：用本地视觉模型识别图片内容。
type visionTool struct{ workDir string }

func (visionTool) Name() string { return "vision" }

func (visionTool) Description() string {
	return "识别图片内容（识图）：读取本地图片文件，用本地视觉模型描述图片中的内容、文字、布局和关键细节。支持自定义提示词。通常几秒，冷启动（模型未加载）约 20 秒+；不消耗主模型 token。"
}

func (visionTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "image_path":{"type":"string","description":"图片文件路径（相对工作区或绝对路径）"},
  "prompt":{"type":"string","description":"可选：识别提示词，例如\"提取图中所有文字\"、\"描述图表内容\"，默认详细描述整张图片"}
},
"required":["image_path"]
}`)
}

func (visionTool) ReadOnly() bool { return true }

func (v visionTool) ReadTarget(args json.RawMessage) (string, bool) {
	var p struct {
		ImagePath string `json:"image_path"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.ImagePath) == "" {
		return "", false
	}
	return resolveIn(v.workDir, p.ImagePath), true
}

// ModelBacked 声明该工具调用本地视觉模型——桌面端把一次调用当作
// 「变相子代理」打开 mt_ 运行记录（与子代理同一会话 UI）。
func (visionTool) ModelBacked() bool { return true }

func (visionTool) CompactDescription() string     { return compactDesc["vision"] }
func (visionTool) CompactSchema() json.RawMessage { return compactSchema["vision"] }

func (v visionTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		ImagePath string `json:"image_path"`
		Prompt    string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.ImagePath == "" {
		return "", fmt.Errorf("image_path 不能为空")
	}
	p.ImagePath = resolveIn(v.workDir, p.ImagePath)
	text, err := vision.RecognizeImage(ctx, p.ImagePath, p.Prompt)
	if err != nil {
		return "", visionWrapErr(err)
	}
	return text, nil
}

// visionWrapErr 失败语义族（batch2-plan B2）：识图失败后的行为指令——
// 模型看到原始错误容易「凭记忆描述图片」编造内容，必须显式禁止。
func visionWrapErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "不可用") || strings.Contains(msg, "provider") || strings.Contains(msg, "冷启动"):
		return fmt.Errorf("%v——本地视觉模型不可用（未配置或未启动）：如实告知用户识图暂不可用；不要凭图片文件名或想象描述图片内容；用户坚持要识图时提示在模型中心配置视觉模型后重试", err)
	case strings.Contains(msg, "读取图片失败"):
		return fmt.Errorf("%v——图片路径读不到：确认文件存在且为图片格式（可先 ls 看目录）；不要对不存在的图片编造描述", err)
	case strings.Contains(msg, "超时") || strings.Contains(msg, "timeout") || strings.Contains(msg, "context"):
		return fmt.Errorf("%v——识图超时（冷启动约 20 秒+）：可重试一次；仍超时就如实告知，不要编造图片内容", err)
	}
	return fmt.Errorf("%v——识图失败：如实转述该错误；不要编造图片内容", err)
}
