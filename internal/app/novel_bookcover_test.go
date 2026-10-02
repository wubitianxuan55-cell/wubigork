package app

// GaeaGenerateBookCover 测试（v4.3g）。
//
// 通过 ai.Client.SetImageBackend 注入假图片后端（ai.ImageBackend 接口，离线程路），
// 使校验分支 + 落盘 + 下载全链路可离线测试，不触真实远端 API：
//   - 假后端可返回 b64（解码落盘）或 URL（经 httptest 本地 HTTP 下载）；
//   - 假后端返回 error 时验证「生成书封失败」错误透传；
//   - 成功路径断言 3:4 尺寸、落盘 .gaea/play/exports、不建 .gaea/work（play 红线）。

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/project"
)

// tinyPNG 1x1 透明 PNG（base64）。
const tinyPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

// recordingImageBackend 复用 image_handler_test.go 的 fakeImageBackend（不修改既有
// 文件），叠加记录最近一次请求与调用次数，便于断言 3:4 尺寸/提示词契约。
type recordingImageBackend struct {
	*fakeImageBackend
	lastReq *ai.ImageGenerationRequest
	callN   int
}

func (r *recordingImageBackend) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	r.lastReq = req
	r.callN++
	return r.fakeImageBackend.GenerateImage(ctx, req)
}

func newTestAppWithProject(t *testing.T, dirName string) (*App, *project.Manager) {
	t.Helper()
	a := New()
	pm, err := project.Create(filepath.Join(t.TempDir(), dirName), "测试书", "玄幻", "热血", "")
	if err != nil {
		t.Fatal(err)
	}
	a.setPM(pm)
	return a, pm
}

func installFakeBackend(t *testing.T, a *App, resp *ai.ImageGenerationResponse, err error) *recordingImageBackend {
	t.Helper()
	rb := &recordingImageBackend{fakeImageBackend: &fakeImageBackend{result: resp, err: err}}
	a.client.SetImageBackend(rb, "openai")
	t.Cleanup(func() { a.client.SetImageBackend(nil, "") }) // 恢复 xAI 缺省，避免污染后续用例
	return rb
}

// TestGaeaGenerateBookCoverNoProject 未打开项目 → 报中文错误。
func TestGaeaGenerateBookCoverNoProject(t *testing.T) {
	a := New()
	_, err := a.GaeaGenerateBookCover("demo", "")
	if err == nil {
		t.Fatal("未打开项目时应报错")
	}
	if !strings.Contains(err.Error(), "请先打开项目") {
		t.Fatalf("错误应含「请先打开项目」，got: %v", err)
	}
}

// TestGaeaGenerateBookCoverProjectMismatch 已打开项目但 projectID 不匹配 → 报错。
func TestGaeaGenerateBookCoverProjectMismatch(t *testing.T) {
	a, _ := newTestAppWithProject(t, "novel-a")
	if _, err := a.GaeaGenerateBookCover("other-project", ""); err == nil {
		t.Fatal("projectID 不匹配时应报错")
	} else if !strings.Contains(err.Error(), "项目不匹配") {
		t.Fatalf("错误应含「项目不匹配」，got: %v", err)
	}
}

// TestGaeaGenerateBookCoverBackendError 生成失败 → 错误透传（含「生成书封失败」）。
func TestGaeaGenerateBookCoverBackendError(t *testing.T) {
	a, _ := newTestAppWithProject(t, "novel-b")
	fb := installFakeBackend(t, a, nil, context.DeadlineExceeded)
	if _, err := a.GaeaGenerateBookCover("novel-b", ""); err == nil {
		t.Fatal("生成失败时应报错")
	} else if !strings.Contains(err.Error(), "生成书封失败") {
		t.Fatalf("错误应含「生成书封失败」，got: %v", err)
	}
	if fb.callN != 1 {
		t.Fatalf("应恰好调用 1 次图片后端，got %d", fb.callN)
	}
}

// TestGaeaGenerateBookCoverSuccessB64 假后端返回 b64 → 解码落盘 play exports。
func TestGaeaGenerateBookCoverSuccessB64(t *testing.T) {
	a, pm := newTestAppWithProject(t, "novel-c")
	fb := installFakeBackend(t, a, &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: tinyPNG}},
	}, nil)

	out, err := a.GaeaGenerateBookCover("novel-c", "红色主色调")
	if err != nil {
		t.Fatalf("生成书封失败: %v", err)
	}

	// 请求契约：3:4 严格尺寸、N=1、prompt 含竖版书封构图与补充要求
	if fb.lastReq == nil {
		t.Fatal("应记录到图片请求")
	}
	if fb.lastReq.Size != "768x1024" {
		t.Fatalf("Size = %q, want 768x1024（3:4）", fb.lastReq.Size)
	}
	if fb.lastReq.N != 1 {
		t.Fatalf("N = %d, want 1", fb.lastReq.N)
	}
	if !strings.Contains(fb.lastReq.Prompt, "竖版书籍封面构图") {
		t.Fatalf("prompt 应含竖版书封构图指令，got: %s", fb.lastReq.Prompt)
	}
	if !strings.Contains(fb.lastReq.Prompt, "红色主色调") {
		t.Fatalf("prompt 应含 promptHint，got: %s", fb.lastReq.Prompt)
	}
	if !strings.Contains(fb.lastReq.Prompt, "测试书") {
		t.Fatalf("prompt 应含项目名，got: %s", fb.lastReq.Prompt)
	}

	// 落盘路径：<项目根>/.gaea/play/exports/cover-novel-c.png（绝对路径）
	wantDir := filepath.Join(pm.Dir, ".gaea", "play", "exports")
	if filepath.Dir(out) != wantDir {
		t.Fatalf("落盘目录 = %q, want %q", filepath.Dir(out), wantDir)
	}
	if filepath.Base(out) != "cover-novel-c.png" {
		t.Fatalf("文件名 = %q, want cover-novel-c.png", filepath.Base(out))
	}
	if !filepath.IsAbs(out) {
		t.Fatalf("应返回绝对路径，got %q", out)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("读取封面失败: %v", err)
	}
	want, _ := base64.StdEncoding.DecodeString(tinyPNG)
	if string(got) != string(want) {
		t.Fatal("落盘字节与 b64 解码不一致")
	}
	// play 红线：不得创建 work 目录
	if _, err := os.Stat(filepath.Join(pm.Dir, ".gaea", "work")); !os.IsNotExist(err) {
		t.Fatalf("不应创建 .gaea/work（play 红线），err=%v", err)
	}
}

// TestGaeaGenerateBookCoverSuccessURL 假后端返回远端 URL → HTTP 下载后落盘。
func TestGaeaGenerateBookCoverSuccessURL(t *testing.T) {
	want, _ := base64.StdEncoding.DecodeString(tinyPNG)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(want)
	}))
	defer srv.Close()

	a, pm := newTestAppWithProject(t, "novel-d")
	installFakeBackend(t, a, &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{URL: srv.URL + "/cover.png"}},
	}, nil)

	out, err := a.GaeaGenerateBookCover("novel-d", "")
	if err != nil {
		t.Fatalf("生成书封失败: %v", err)
	}
	if !strings.HasPrefix(filepath.ToSlash(out), filepath.ToSlash(filepath.Join(pm.Dir, ".gaea", "play", "exports"))) {
		t.Fatalf("落盘路径不在 play exports: %q", out)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("读取封面失败: %v", err)
	}
	if string(got) != string(want) {
		t.Fatal("下载落盘字节与服务端不一致")
	}
}

// TestGaeaGenerateBookCoverSafeModePrompt 是 AP2-04 的护栏契约锁
// （审计 2026-10-02 批次十四）：
//   - 安全模式**关闭**（零配置）→ 请求提示词不含安全段（改造前逐字节形态）；
//   - 安全模式**开启** → 请求提示词 == 『关闭时的基线提示词 + imageSafePromptSuffix』
//     逐字节相等——既证明注入了安全段，也证明注入是改造前后的**唯一**差异
//     （没有顺手改动书封 prompt 其它部分）；
//   - 3:4 尺寸契约不受护栏影响（防止日后为接护栏改走统一入口而悄悄丢 size）。
func TestGaeaGenerateBookCoverSafeModePrompt(t *testing.T) {
	a, _ := newTestAppWithProject(t, "novel-safe")
	fb := installFakeBackend(t, a, &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: tinyPNG}},
	}, nil)

	// ① 关闭（playGuardrails 零值）：基线提示词。
	if _, err := a.GaeaGenerateBookCover("novel-safe", "夜色蓝调"); err != nil {
		t.Fatalf("生成书封失败: %v", err)
	}
	offPrompt := fb.lastReq.Prompt
	if strings.Contains(offPrompt, "内容安全要求") {
		t.Fatalf("安全模式关闭时不得注入安全段: %s", offPrompt)
	}

	// ② 开启：安全段注入。
	withGaeaCfg(t, guardrailsCfg(gaeaConfig.PlayGuardrails{Enabled: true, ImageSafeMode: true}))
	if _, err := a.GaeaGenerateBookCover("novel-safe", "夜色蓝调"); err != nil {
		t.Fatalf("生成书封失败: %v", err)
	}
	onPrompt := fb.lastReq.Prompt
	if want := offPrompt + imageSafePromptSuffix; onPrompt != want {
		t.Fatalf("安全模式提示词应为「基线+安全段」逐字节相等:\ngot  %q\nwant %q", onPrompt, want)
	}
	if fb.callN != 2 {
		t.Fatalf("两次调用应各发一次请求，got %d", fb.callN)
	}
	if fb.lastReq.Size != "768x1024" {
		t.Fatalf("护栏不得改变严格 3:4 尺寸契约，got %q", fb.lastReq.Size)
	}
}

// TestSanitizeCoverID 封面文件名段清洗。
func TestSanitizeCoverID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"demo", "demo"},
		{"我的第一本书", "我的第一本书"},
		{"a/b/c", "c"},         // 只取末段
		{"a:b*c?d", "a_b_c_d"}, // Windows 非法字符替换
		{"cover 123", "cover 123"},
		{"..", "project"},  // 空
		{"", "project"},    // 空
		{"CON", "project"}, // 保留设备名
		{"com1", "project"},
	}
	for _, c := range cases {
		if got := sanitizeCoverID(c.in); got != c.want {
			t.Errorf("sanitizeCoverID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
