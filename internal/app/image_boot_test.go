package app

// v4.387 ComfyUI 未运行自动拉起测试：生成链对「连接 ComfyUI 失败」（dial
// 连接被拒）先 ensureComfyUIRunning（拉起+就绪等待）再重试一次；原罪插图
// 与绘梦生成共用该链。测试不拉真进程——就绪探测指向 httptest 假端点，启动
// 路径用「无 main.py 的空目录」钉住 StartComfyUI 的失败分支。

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
)

// flakyImageBackend 首次调用返回 errOnce，之后成功——钉「自动拉起后重试
// 恰好一次且成功」的接线形状。
type flakyImageBackend struct {
	errOnce error
	calls   int
}

func (f *flakyImageBackend) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	f.calls++
	if f.calls == 1 && f.errOnce != nil {
		return nil, f.errOnce
	}
	return &ai.ImageGenerationResponse{Data: []ai.ImageData{{B64JSON: pngDataURLApp("boot-ok")}}}, nil
}
func (f *flakyImageBackend) Interrupt(ctx context.Context) error { return nil }
func (f *flakyImageBackend) ResetCancel()                        {}

// TestGenerateImage_AutoBootRetriesOnConnectRefused：连接被拒 →
// ensureComfyUIRunning（此处以假 /system_stats 端点直接就绪）→ 重试成功。
func TestGenerateImage_AutoBootRetriesOnConnectRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	fake := &flakyImageBackend{errOnce: errors.New(`连接 ComfyUI 失败 (http://127.0.0.1:8188): Post "http://127.0.0.1:8188/prompt": dial tcp 127.0.0.1:8188: connectex: No connection could be made because the target machine actively refused it`)}
	c := &ai.Client{}
	c.SetImageBackend(fake, "comfyui")
	ms := &mediaState{core: &core{cfg: &config.Config{
		ImageBackend: "comfyui",
		ComfyUIURL:   srv.URL,
		// ComfyUIPath 留空也行：isComfyUIRunning 命中假端点即 true，不走启动分支。
		// ImageSaveDir 指向临时目录：跳过 saveToNovelImages（其依赖装配态路径）。
		ImageSaveDir: t.TempDir(),
	}, client: c}}

	res, err := ms.generateImageInternal(imageGenInternal{prompt: "夜景", n: 1})
	if err != nil {
		t.Fatalf("generateImageInternal: %v", err)
	}
	if res["error"] != nil {
		t.Fatalf("自动拉起重试后不应失败: %v", res["error"])
	}
	if fake.calls != 2 {
		t.Fatalf("GenerateImage 调用次数 = %d, want 2（首败+重试成功）", fake.calls)
	}
}

// TestGenerateImage_ConnectRefusedNoPathKeepsError：未配置安装路径且服务
// 未运行 → 不尝试启动，原始连接错误如实上抛（不吞错不换错）。
func TestGenerateImage_ConnectRefusedNoPathKeepsError(t *testing.T) {
	fake := &flakyImageBackend{errOnce: errors.New("连接 ComfyUI 失败 (http://127.0.0.1:1): dial tcp: connection refused")}
	c := &ai.Client{}
	c.SetImageBackend(fake, "comfyui")
	ms := &mediaState{core: &core{cfg: &config.Config{
		ImageBackend: "comfyui",
		ComfyUIURL:   "http://127.0.0.1:1", // 无监听端口：连接立刻被拒
	}, client: c}}

	res, err := ms.generateImageInternal(imageGenInternal{prompt: "夜景", n: 1})
	if err != nil {
		t.Fatalf("generateImageInternal: %v", err)
	}
	msg, _ := res["error"].(string)
	if msg == "" || !strings.Contains(msg, "连接 ComfyUI 失败") {
		t.Fatalf("应保留原始连接错误, got %q", msg)
	}
	if fake.calls != 1 {
		t.Fatalf("无路径时不重试, calls = %d, want 1", fake.calls)
	}
}

// TestEnsureComfyUIRunning 契约三分支：已运行即真 / 未配置路径即假 /
// 启动失败（无 main.py）即假——都不拉真进程。
func TestEnsureComfyUIRunning(t *testing.T) {
	// 已运行：假端点 200。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	ms := &mediaState{core: &core{cfg: &config.Config{ComfyUIURL: srv.URL}}}
	if !ms.ensureComfyUIRunning() {
		t.Fatal("服务已运行应返回 true")
	}

	// 未配置安装路径：不尝试启动。
	ms = &mediaState{core: &core{cfg: &config.Config{ComfyUIURL: "http://127.0.0.1:1"}}}
	if ms.ensureComfyUIRunning() {
		t.Fatal("未配置路径应返回 false")
	}

	// 路径存在但目录里没有 main.py：StartComfyUI 失败 → false（快速返回，
	// 不进入 120s 等待循环）。
	ms = &mediaState{core: &core{cfg: &config.Config{
		ComfyUIURL:  "http://127.0.0.1:1",
		ComfyUIPath: t.TempDir(),
	}}}
	if ms.ensureComfyUIRunning() {
		t.Fatal("启动失败应返回 false")
	}
}
