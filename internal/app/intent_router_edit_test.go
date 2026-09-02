package app

// intent_router_edit_test.go — 对话式改图执行层（v4.9）：缓存未命中不接管、
// 命中走 editImageFromCard 契约（wxEditImageInvoker seam 替身）、失败诚实
// 回复、能力未装配降级、助手上下文边界与 dry-run 预览。

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
)

// withEditInvoker 临时替换改图执行 seam（测毕恢复），并记录调用入参。
func withEditInvoker(t *testing.T, fn func(a *App, initImage, prompt string) (string, error)) (gotInit, gotPrompt *string) {
	t.Helper()
	orig := wxEditImageInvoker
	init, prompt := "", ""
	wxEditImageInvoker = func(a *App, initImage, p string) (string, error) {
		init, prompt = initImage, p
		return fn(a, initImage, p)
	}
	t.Cleanup(func() { wxEditImageInvoker = orig })
	return &init, &prompt
}

// seedWxEditCache 向测试 App 的缓存注入一张 PNG（走真实 Set：复制自持+魔数）。
func seedWxEditCache(t *testing.T, a *App, assistantID string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "inbound.png")
	if err := os.WriteFile(p, wxEditTestPNG, 0o644); err != nil {
		t.Fatalf("写源图片: %v", err)
	}
	e, err := wxEditImageCache(a.whisperState.whisperDataRoot).Set(assistantID, p)
	if err != nil {
		t.Fatalf("Set 缓存: %v", err)
	}
	return e.Path
}

// 缓存未命中=不接管（决策口径）：回落聊天管道，让对话自然处理「没有图」。
func TestExecEditImage_CacheMissNotHandled(t *testing.T) {
	a := newChatServiceTestApp(t)

	res := a.routeIntentWithResultForAssistant("把这张图的背景换成海边", "ast-none")
	if res.Handled || res.Reply != "" || res.CardPath != "" {
		t.Fatalf("缓存未命中应不接管: %+v", res)
	}
}

// assistantID 为空（语音/命令面板入口）不接管——GaeaRouteIntent 签名零变更
// 且永不从面板执行改图。
func TestExecEditImage_EmptyAssistantNotHandled(t *testing.T) {
	a := newChatServiceTestApp(t)
	seedWxEditCache(t, a, "ast-panel") // 即便缓存里有图，无助手上下文也不接管

	if res := a.routeIntentWithResultForAssistant("把这张图的背景换成海边", ""); res.Handled {
		t.Fatalf("assistantID 为空应不接管: %+v", res)
	}
	if res := a.GaeaRouteIntent("把这张图的背景换成海边", false); res.Handled {
		t.Fatalf("面板入口应不接管: %+v", res)
	}
	if res := a.GaeaRouteIntent("把这张图的背景换成海边", true); res.Handled {
		t.Fatalf("面板 dry-run 也应不预览改图: %+v", res)
	}
}

// 命中路：读自持副本转 data URL → editImageFromCard（seam 捕获入参）→
// CardPath=产物路径 + 简短确认。
func TestExecEditImage_HitCallsContract(t *testing.T) {
	a := newChatServiceTestApp(t)
	cached := seedWxEditCache(t, a, "ast-hit")
	gotInit, gotPrompt := withEditInvoker(t, func(a *App, initImage, prompt string) (string, error) {
		return `D:\out\edited.png`, nil
	})

	res := a.routeIntentWithResultForAssistant("把这张图的背景换成海边", "ast-hit")
	if !res.Handled {
		t.Fatal("命中应接管")
	}
	if res.CardPath != `D:\out\edited.png` {
		t.Errorf("CardPath = %q, want 产物路径", res.CardPath)
	}
	if !strings.Contains(res.Reply, "改好") {
		t.Errorf("确认语应含「改好」: %q", res.Reply)
	}
	if *gotPrompt != "背景换成海边" {
		t.Errorf("prompt = %q, want 去掉指代前缀的编辑指令", *gotPrompt)
	}
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(*gotInit, prefix) {
		t.Fatalf("initImage 应为 png data URL: %q", (*gotInit)[:min(60, len(*gotInit))])
	}
	dec, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(*gotInit, prefix))
	if err != nil {
		t.Fatalf("data URL 解码: %v", err)
	}
	want, err := os.ReadFile(cached)
	if err != nil || !bytes.Equal(dec, want) {
		t.Errorf("data URL 应与缓存副本逐字节一致: err=%v", err)
	}
}

// 引擎失败：诚实回错误摘要（Handled=true，不坠回聊天）。
func TestExecEditImage_EngineErrorHonestReply(t *testing.T) {
	a := newChatServiceTestApp(t)
	seedWxEditCache(t, a, "ast-err")
	withEditInvoker(t, func(a *App, initImage, prompt string) (string, error) {
		return "", errors.New("模型超时")
	})
	res := a.routeIntentWithResultForAssistant("刚才那张图调成黑白", "ast-err")
	if !res.Handled {
		t.Fatal("引擎失败仍应接管（失败要说出口）")
	}
	if !strings.Contains(res.Reply, "改图失败") || !strings.Contains(res.Reply, "模型超时") {
		t.Errorf("应诚实回错误摘要: %q", res.Reply)
	}
	if res.CardPath != "" {
		t.Errorf("失败不应带产物: %+v", res)
	}
}

// 能力未装配（媒体域缺失/引擎契约未落地）：对齐 execGenerateImage 先例降级
// 为未命中——seam 生产实现返回 errEditImageUnavailable。
func TestExecEditImage_CapabilityMissingDegrades(t *testing.T) {
	a := newChatServiceTestApp(t) // 此测试 App 无媒体域（a.mediaState == nil）
	seedWxEditCache(t, a, "ast-cap")
	// 不替换 seam：生产实现走接口断言 → mediaState 缺失 → 不可用。

	res := a.routeIntentWithResultForAssistant("把这张图的背景换成海边", "ast-cap")
	if res.Handled {
		t.Fatalf("能力未装配应降级为未命中: %+v", res)
	}
}

// 副本丢失（外部清走缓存目录内容）：Get 按未命中处理 → 不接管。
func TestExecEditImage_MissingFileNotHandled(t *testing.T) {
	a := newChatServiceTestApp(t)
	cached := seedWxEditCache(t, a, "ast-gone")
	os.Remove(cached)

	if res := a.routeIntentWithResultForAssistant("把这张图的背景换成海边", "ast-gone"); res.Handled {
		t.Fatalf("副本丢失应按未命中处理: %+v", res)
	}
}

// 助手感知与无感知入口同源一致：传空串时 routeIntentWithResult 与
// routeIntentWithResultForAssistant 行为一致（包装关系回归护栏）。
func TestRouteIntentWithResultWrapsForAssistant(t *testing.T) {
	a := newChatServiceTestApp(t)

	direct := a.routeIntentWithResult("现在用什么模型")
	wrapped := a.routeIntentWithResultForAssistant("现在用什么模型", "")
	if direct != wrapped {
		t.Fatalf("routeIntentWithResult = %+v, ForAssistant(\"\") = %+v（应一致）", direct, wrapped)
	}
	if res := a.routeIntentWithResultForAssistant("今天天气怎么样", "ast-x"); res.Handled {
		t.Fatalf("闲聊不应接管: %+v", res)
	}
}

// 命中路 → 真实契约全链（集成）：接口断言命中 *mediaState 的
// editImageFromCard（引擎线已落地），产物落盘路径作 CardPath 回推。
func TestExecEditImage_IntegrationRealContract(t *testing.T) {
	a := newChatServiceTestApp(t)
	seedWxEditCache(t, a, "ast-e2e")

	rec := &editCardImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("edited")}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(rec, "dashscope")
	a.mediaState = &mediaState{core: &core{cfg: &config.Config{
		ImageBackend: "dashscope",
		ImageModel:   "qwen-image-edit-max",
		ImageSaveDir: t.TempDir(),
	}, client: c}, app: a}

	res := a.routeIntentWithResultForAssistant("把这张图的背景换成海边", "ast-e2e")
	if !res.Handled || res.CardPath == "" {
		t.Fatalf("全链应命中且带产物: %+v", res)
	}
	if _, err := os.Stat(res.CardPath); err != nil {
		t.Fatalf("产物应已落盘: %v", err)
	}
	if rec.req == nil {
		t.Fatal("应发出生成请求")
	}
	if rec.req.Mode != "img2img" || rec.req.N != 1 {
		t.Errorf("请求契约应锚定 img2img/N=1: %+v", rec.req)
	}
	if rec.req.Prompt != "背景换成海边" {
		t.Errorf("Prompt = %q, want 去掉指代前缀的编辑指令", rec.req.Prompt)
	}
	if !strings.HasPrefix(rec.req.InitImage, "data:image/png;base64,") {
		t.Errorf("initImage 应为缓存副本的 data URL: %q", rec.req.InitImage[:min(60, len(rec.req.InitImage))])
	}
}

// dry-run 预览（助手上下文内）：缓存命中给诚实预览；未命中不预览。
func TestExecEditImage_DryRunPreview(t *testing.T) {
	a := newChatServiceTestApp(t)

	if res := a.routeIntentModeForAssistant("把这张图的背景换成海边", true, "ast-prev"); res.Handled {
		t.Fatalf("缓存未命中不应预览: %+v", res)
	}

	seedWxEditCache(t, a, "ast-prev")
	res := a.routeIntentModeForAssistant("把这张图的背景换成海边", true, "ast-prev")
	if !res.Handled || !strings.Contains(res.Reply, "将编辑") {
		t.Fatalf("命中应预览「将编辑…」: %+v", res)
	}
	if res.Action != "edit_image" {
		t.Errorf("预览应带 Action=edit_image: %+v", res)
	}
}
