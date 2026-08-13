package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
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

// TestGetImageBackendInfo_FullConfig 验证模型中心恢复表单所需的完整配置字段。
func TestGetImageBackendInfo_FullConfig(t *testing.T) {
	ms := &mediaState{
		core: &core{
			cfg: &config.Config{
				ImageBackend:      "comfyui",
				ImageModel:        "z-image-turbo",
				ComfyUIURL:        "http://127.0.0.1:8188",
				ImageSaveDir:      `D:\pics`,
				ComfyUIPath:       `C:\ComfyUI`,
				ComfyUIPythonPath: `C:\ComfyUI\python.exe`,
			},
		},
	}
	got := ms.GetImageBackendInfo()
	if got["model"] != "z-image-turbo" || got["image_model"] != "z-image-turbo" {
		t.Fatalf("model/image_model = %q/%q, want z-image-turbo", got["model"], got["image_model"])
	}
	if got["comfyui_url"] != "http://127.0.0.1:8188" {
		t.Fatalf("comfyui_url = %q", got["comfyui_url"])
	}
	if got["image_save_dir"] != `D:\pics` {
		t.Fatalf("image_save_dir = %q", got["image_save_dir"])
	}
	if got["comfyui_path"] != `C:\ComfyUI` {
		t.Fatalf("comfyui_path = %q", got["comfyui_path"])
	}
	if got["comfyui_python_path"] != `C:\ComfyUI\python.exe` {
		t.Fatalf("comfyui_python_path = %q", got["comfyui_python_path"])
	}
}

// TestGetImageBackendInfo_Defaults 验证空模型/空保存目录的兜底值。
func TestGetImageBackendInfo_Defaults(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\test`)

	t.Run("xai 空模型默认高质量", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "xai"}}}
		if got := ms.GetImageBackendInfo()["image_model"]; got != "grok-imagine-image-quality" {
			t.Fatalf("image_model = %q", got)
		}
	})

	t.Run("comfyui 空模型默认 krea2", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "comfyui"}}}
		if got := ms.GetImageBackendInfo()["image_model"]; got != "krea2" {
			t.Fatalf("image_model = %q, want krea2", got)
		}
	})

	t.Run("空保存目录回退默认路径", func(t *testing.T) {
		ms := &mediaState{core: &core{cfg: &config.Config{ImageBackend: "xai"}}}
		want := filepath.Join(`C:\Users\test`, "Pictures", "gaea")
		if got := ms.GetImageBackendInfo()["image_save_dir"]; got != want {
			t.Fatalf("image_save_dir = %q, want %q", got, want)
		}
	})
}
