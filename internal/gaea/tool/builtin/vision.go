package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/vision"
)

func init() { tool.RegisterBuiltin(visionTool{}) }

// visionTool 识图工具：用本地视觉模型识别图片内容。
type visionTool struct{}

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

func (visionTool) CompactDescription() string     { return compactDesc["vision"] }
func (visionTool) CompactSchema() json.RawMessage { return compactSchema["vision"] }

func (visionTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
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
	text, err := vision.RecognizeImage(ctx, p.ImagePath, p.Prompt)
	if err != nil {
		return "", err
	}
	return text, nil
}
