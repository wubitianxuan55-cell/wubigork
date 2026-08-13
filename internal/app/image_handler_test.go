package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
)

func TestGenerateFreeImage_sizeCleanup(t *testing.T) {
	tests := []struct {
		backendType string
		expectSize  bool
	}{
		{"comfyui", true},
		{"xai", false},
		{"herdsman", true},
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
			if tt.backendType != "comfyui" && tt.backendType != "herdsman" {
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

// TestFindPython_StandaloneEnv 验证 standalone-env 优先于系统 python（ROCm PyTorch 必需）
func TestFindPython_StandaloneEnv(t *testing.T) {
	root := t.TempDir()
	comfyPath := filepath.Join(root, "ComfyUI")
	os.MkdirAll(filepath.Join(comfyPath), 0o755)
	os.MkdirAll(filepath.Join(root, "standalone-env"), 0o755)

	// 模拟 standalone-env python 存在
	os.WriteFile(filepath.Join(root, "standalone-env", "python.exe"), []byte("x"), 0o755)

	// 配置路径为空 → 应自动找到 standalone-env
	got := findPython(comfyPath, "")
	want := filepath.Join(root, "standalone-env", "python.exe")
	if got != want {
		t.Errorf("findPython = %q, want %q（standalone-env 应优先，系统 Python 是 CPU-only）", got, want)
	}

	// 显式配置优先于自动查找
	explicit := filepath.Join(root, "my-python", "python.exe")
	os.MkdirAll(filepath.Join(root, "my-python"), 0o755)
	os.WriteFile(explicit, []byte("x"), 0o755)
	if got := findPython(comfyPath, explicit); got != explicit {
		t.Errorf("显式配置优先 = %q, want %q", got, explicit)
	}
}
