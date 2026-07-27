package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestImageGenerationRequest_xAI_Cleanup(t *testing.T) {
	// 模拟前端发来的完整请求（所有字段都有值）
	req := &ImageGenerationRequest{
		Model:    "grok-imagine-image-quality",
		Prompt:   "一座悬浮在云端的东方仙侠城市",
		Negative: "模糊, 低质量",
		N:        1,
		Size:     "1024x1024",
		Seed:     12345,
	}

	// 应用 xAI 清理逻辑（与 generateImageXAI 中一致）
	req.Size = ""
	req.Negative = ""
	req.Seed = 0

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(body)
	t.Logf("xAI JSON: %s", jsonStr)

	// 必须存在的字段
	for _, must := range []string{`"model"`, `"prompt"`} {
		if !strings.Contains(jsonStr, must) {
			t.Errorf("缺少必要字段: %s", must)
		}
	}

	// 不能存在的字段（omitempty + 已清空）
	for _, banned := range []string{`"size"`, `"negative"`, `"seed"`} {
		if strings.Contains(jsonStr, banned) {
			t.Errorf("不应包含字段: %s (xAI 不支持)", banned)
		}
	}
}

func TestImageGenerationRequest_ComfyUI_KeepsAll(t *testing.T) {
	// ComfyUI 需要所有字段
	req := &ImageGenerationRequest{
		Model:    "flux",
		Prompt:   "一座悬浮在云端的东方仙侠城市",
		Negative: "模糊, 低质量",
		N:        1,
		Size:     "1024x1024",
		Seed:     12345,
	}

	// ComfyUI 不清除字段
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	jsonStr := string(body)
	t.Logf("ComfyUI JSON: %s", jsonStr)

	for _, must := range []string{`"size"`, `"negative"`, `"seed"`} {
		if !strings.Contains(jsonStr, must) {
			t.Errorf("ComfyUI 应包含字段: %s", must)
		}
	}
}
