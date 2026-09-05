package ai

// CU1（蒸馏 unsloth §六-2）ComfyUI 预热回归：假 ComfyUI 端到端验证
// 空跑工作流的「极小尺寸/最低步数/正确模型文件」三要素与完成收敛。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestComfyUIWarmupQueuesTinyWorkflowAndCompletes(t *testing.T) {
	var gotWorkflow map[string]interface{}
	var gotViewHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/prompt":
			var body struct {
				Prompt map[string]interface{} `json:"prompt"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("解析 /prompt 失败: %v", err)
			}
			gotWorkflow = body.Prompt
			w.Write([]byte(`{"prompt_id":"warm-1"}`))
		case r.URL.Path == "/history/warm-1":
			w.Write([]byte(`{"warm-1":{"status":{"status_str":"success"},"outputs":{"12":{"images":[{"filename":"warm.png","subfolder":"","type":"output"}]}}}}`))
		case r.URL.Path == "/view":
			gotViewHits++
			w.Write([]byte("PNG"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := NewComfyUIBackend(srv.URL)
	// pollInterval 供 waitForResult 轮询（backend 默认值即可，历史已就绪首轮命中）
	err := b.Warmup(context.Background(), "krea2")
	if err != nil {
		t.Fatalf("Warmup: %v", err)
	}
	if gotViewHits != 1 {
		t.Errorf("应下载一次空跑产物，view=%d", gotViewHits)
	}

	// 三要素：模型文件正确 + 64×64 + 1 步（加载是目的，算力开销可忽略）
	if gotWorkflow == nil {
		t.Fatal("应提交空跑工作流")
	}
	unet, _ := gotWorkflow["4"].(map[string]interface{})
	if unet == nil || unet["class_type"] != "UNETLoader" {
		t.Fatalf("节点 4 应为 UNETLoader，got %v", gotWorkflow["4"])
	}
	unetInputs, _ := unet["inputs"].(map[string]interface{})
	if unetInputs["unet_name"] != "krea2_turbo_fp8_scaled.safetensors" {
		t.Errorf("预热应加载默认 krea2 权重，got %v", unetInputs["unet_name"])
	}
	latent, _ := gotWorkflow["9"].(map[string]interface{})
	if latent == nil || latent["class_type"] != "EmptyLatentImage" {
		t.Fatalf("节点 9 应为 EmptyLatentImage，got %v", gotWorkflow["9"])
	}
	latentInputs, _ := latent["inputs"].(map[string]interface{})
	if latentInputs["width"] != float64(64) || latentInputs["height"] != float64(64) {
		t.Errorf("空跑应为 64×64，got %v", latentInputs)
	}
	sampler, _ := gotWorkflow["10"].(map[string]interface{})
	samplerInputs, _ := sampler["inputs"].(map[string]interface{})
	if samplerInputs["steps"] != float64(1) {
		t.Errorf("空跑应为 1 步，got %v", samplerInputs["steps"])
	}
}

func TestComfyUIWarmupUnknownModelRejected(t *testing.T) {
	b := NewComfyUIBackend("http://127.0.0.1:1")
	err := b.Warmup(context.Background(), "ghost-model")
	if err == nil || !strings.Contains(err.Error(), "未登记预热工作流") {
		t.Fatalf("未知模型应诚实拒绝，got %v", err)
	}
}

func TestComfyUIWarmupSupportedGate(t *testing.T) {
	if !ComfyUIWarmupSupported("krea2") || !ComfyUIWarmupSupported("z-image-turbo") || !ComfyUIWarmupSupported("flux") {
		t.Error("三个登记模型都应可预热")
	}
	if ComfyUIWarmupSupported("ghost-model") {
		t.Error("未登记模型不应可预热")
	}
}

func TestComfyUIWarmupWaitTimeoutIsBounded(t *testing.T) {
	// 历史永不完成 → ctx 到期返回，不悬挂（用短 ctx 模拟）。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/prompt" {
			w.Write([]byte(`{"prompt_id":"warm-2"}`))
			return
		}
		if r.URL.Path == "/history/warm-2" {
			w.Write([]byte(`{}`)) // 未完成
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	b := NewComfyUIBackend(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if err := b.Warmup(ctx, "krea2"); err == nil {
		t.Fatal("超时应返回错误而非悬挂")
	}
}
