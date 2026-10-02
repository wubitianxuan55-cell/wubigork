package app

// resolveImageBackend 单测（IN2-03）：app 层五份手写「backend 名 → 构造图片
// 后端」switch 收敛后的单一来源，分支语义在此锁口——fail-closed xAI 兜底、
// 结构化不可用原因（引擎未启用 / GLM 缺 Key / ComfyUI 缺地址）、构造类型与
// BaseURL 口径。

import (
	"errors"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// newImageBackendTestEnv 独立 Manager + 配置（statePath 为空，saveState 落空，
// 不触碰真实用户状态）。
func newImageBackendTestEnv(t *testing.T) (*config.Config, *modelengine.Manager) {
	t.Helper()
	mgr := modelengine.NewManager("", "")
	// herdsman 预置即存在，这里显式关停供「未启用」分支用例
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "herdsman", Enabled: false}); err != nil {
		t.Fatalf("SaveEngine herdsman: %v", err)
	}
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "ollama", BaseURL: "http://127.0.0.1:9/ollama/v1", Enabled: true}); err != nil {
		t.Fatalf("SaveEngine ollama: %v", err)
	}
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "glm", BaseURL: "https://glm.example/api/paas/v4", Enabled: true}); err != nil {
		t.Fatalf("SaveEngine glm: %v", err)
	}
	cfg := &config.Config{ComfyUIURL: "http://127.0.0.1:8188"}
	return cfg, mgr
}

func TestResolveImageBackend(t *testing.T) {
	cfg, mgr := newImageBackendTestEnv(t)

	t.Run("xai 空名与未知名 fail-closed 到 xAI 原生管线", func(t *testing.T) {
		for _, name := range []string{"xai", "", "wat"} {
			r, err := resolveImageBackend(name, cfg, mgr)
			if err != nil {
				t.Fatalf("%q: 期望无错误，得 %v", name, err)
			}
			if r.Backend != nil || r.Kind != "xai" {
				t.Fatalf("%q: 期望 (nil,\"xai\")，得 (%v,%q)", name, r.Backend, r.Kind)
			}
		}
	})

	t.Run("comfyui 地址缺失报结构化原因且保留规范名", func(t *testing.T) {
		empty := &config.Config{}
		r, err := resolveImageBackend("comfyui", empty, mgr)
		var re *imageBackendResolveError
		if !errors.As(err, &re) || re.Reason != reasonComfyURLMissing {
			t.Fatalf("期望 reasonComfyURLMissing，得 %v", err)
		}
		if r.Kind != "comfyui" {
			t.Fatalf("错误态 Kind 应保留规范名 comfyui，得 %q", r.Kind)
		}
	})

	t.Run("comfyui 地址齐备构造实例", func(t *testing.T) {
		r, err := resolveImageBackend("comfyui", cfg, mgr)
		if err != nil {
			t.Fatalf("期望成功，得 %v", err)
		}
		if _, ok := r.Backend.(*ai.ComfyUIBackend); !ok {
			t.Fatalf("期望 *ai.ComfyUIBackend，得 %T", r.Backend)
		}
		if r.BaseURL != cfg.ComfyUIURL {
			t.Fatalf("BaseURL 口径不符：得 %q", r.BaseURL)
		}
	})

	t.Run("herdsman 未启用报引擎原因 启用后走 openai 兼容构造", func(t *testing.T) {
		_, err := resolveImageBackend("herdsman", cfg, mgr)
		var re *imageBackendResolveError
		if !errors.As(err, &re) || re.Reason != reasonEngineDisabled {
			t.Fatalf("期望 reasonEngineDisabled，得 %v", err)
		}
		if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "herdsman", Enabled: true}); err != nil {
			t.Fatalf("SaveEngine herdsman 启用: %v", err)
		}
		r, err := resolveImageBackend("herdsman", cfg, mgr)
		if err != nil {
			t.Fatalf("启用后期望成功，得 %v", err)
		}
		if _, ok := r.Backend.(*ai.OpenAIImageBackend); !ok {
			t.Fatalf("期望 *ai.OpenAIImageBackend，得 %T", r.Backend)
		}
		if r.BaseURL != "http://localhost:8080/v1" {
			t.Fatalf("BaseURL 应取自引擎配置，得 %q", r.BaseURL)
		}
	})

	t.Run("glm 缺 Key 报 Key 原因 配 Key 后构造", func(t *testing.T) {
		_, err := resolveImageBackend("glm", cfg, mgr)
		var re *imageBackendResolveError
		if !errors.As(err, &re) || re.Reason != reasonGLMKeyMissing {
			t.Fatalf("期望 reasonGLMKeyMissing，得 %v", err)
		}
		mgr.UpdateGLMKey("glm-test-key")
		r, err := resolveImageBackend("glm", cfg, mgr)
		if err != nil {
			t.Fatalf("配 Key 后期望成功，得 %v", err)
		}
		if _, ok := r.Backend.(*ai.GLMImageBackend); !ok {
			t.Fatalf("期望 *ai.GLMImageBackend，得 %T", r.Backend)
		}
	})

	t.Run("engineMgr 为 nil 一律按引擎不可用", func(t *testing.T) {
		for _, name := range []string{"herdsman", "ollama", "glm"} {
			_, err := resolveImageBackend(name, cfg, nil)
			var re *imageBackendResolveError
			if !errors.As(err, &re) || re.Reason != reasonEngineDisabled {
				t.Fatalf("%q: 期望 reasonEngineDisabled，得 %v", name, err)
			}
		}
	})

	t.Run("报错文案显示名映射", func(t *testing.T) {
		for name, want := range map[string]string{"herdsman": "Herdsman", "ollama": "Ollama", "glm": "GLM", "xai": "xai"} {
			if got := imageEngineDisplayName(name); got != want {
				t.Fatalf("imageEngineDisplayName(%q)=%q，期望 %q", name, got, want)
			}
		}
	})
}
