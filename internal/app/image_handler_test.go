package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/wubigork/wubigork/internal/ai"
)

func TestGenerateFreeImage_sizeCleanup(t *testing.T) {
	tests := []struct {
		backendType string
		expectSize  bool
	}{
		{"comfyui", true},
		{"xai", false},
		{"herdsman", false},
		{"ollama", false},
	}

	for _, tt := range tests {
		t.Run(tt.backendType, func(t *testing.T) {
			req := &ai.ImageGenerationRequest{
				Model:    "test-model",
				Prompt:   "test prompt",
				Negative: "bad quality",
				N:        1,
				Size:     "1024x1024",
				Seed:     42,
			}

			// 模拟 image_handler.go 中的清理逻辑
			if tt.backendType != "comfyui" {
				req.Size = ""
			}

			body, _ := json.Marshal(req)
			jsonStr := string(body)

			if tt.expectSize && !strings.Contains(jsonStr, `"size"`) {
				t.Errorf("%s 应包含 size，实际: %s", tt.backendType, jsonStr)
			}
			if !tt.expectSize && strings.Contains(jsonStr, `"size"`) {
				t.Errorf("%s 不应包含 size，实际: %s", tt.backendType, jsonStr)
			}
		})
	}
}
