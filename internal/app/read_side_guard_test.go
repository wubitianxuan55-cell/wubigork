package app

// 2026-09-19 审计读侧收口回归：①GaeaReadFileB64 契约强制化（仅系统对话框
// 选择过的文件可读）②GaeaReadFile 拒绝穿越/绝对路径③快照 sceneID 白名单。

import (
	"os"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/snapshot"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileB64RequiresPick(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	a := &App{core: &core{cfg: &config.Config{}}}

	dir := t.TempDir()
	p := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 未登记：拒绝（fail-closed——此前任意绝对路径 ≤64MB 可读）。
	if _, err := a.GaeaReadFileB64(p); err == nil || !strings.Contains(err.Error(), "文件对话框") {
		t.Fatalf("未登记路径应拒绝, got %v", err)
	}

	// 经 GaeaPickFiles 登记后可读——直接调登记助手模拟对话框结果。
	registerPickedFile(p)
	b64, err := a.GaeaReadFileB64(p)
	if err != nil {
		t.Fatalf("登记后应可读: %v", err)
	}
	if !strings.Contains(b64, "aGVsbG8=") { // base64("hello")
		t.Fatalf("内容不符: %s", b64)
	}
}

func TestGaeaReadFileTraversalGuard(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	a := &App{core: &core{cfg: &config.Config{}}}

	for _, rel := range []string{`..\..\secret.txt`, "../../secret.txt", `C:\Windows\win.ini`} {
		got := a.GaeaReadFile(rel)
		if !strings.Contains(got.Markdown, "非法") && !strings.Contains(got.Markdown, "不在可读范围") {
			t.Fatalf("穿越路径 %q 应被拒绝, got %+v", rel, got)
		}
	}
	// 工作区内正常文件不受影响。
	if err := os.WriteFile("note.md", []byte("# ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := a.GaeaReadFile("note.md"); !strings.Contains(got.Markdown, "# ok") {
		t.Fatalf("工作区内文件应可读, got %+v", got)
	}
}

func TestSnapshotSceneIDGuard(t *testing.T) {
	s := snapshot.NewStore(t.TempDir())
	if _, err := s.Capture(`..\evil`, "正文", "标签", "manual"); err == nil || !strings.Contains(err.Error(), "非法场景") {
		t.Fatalf("穿越 sceneID 应拒绝, got %v", err)
	}
	if _, err := s.Capture("001-evil", "正文", "标签", "manual"); err != nil {
		t.Fatalf("合法 sceneID 应通过: %v", err)
	}
}

// v4.363：ConvertToPdf 读侧收口——相对穿越拒绝、任意绝对路径 fail-closed、
// 登记路径与工作区路径放行（放行断言用 .doc 扩展名拒绝分支，零外部进程）。
func TestConvertToPdfPathGuard(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	a := &App{core: &core{cfg: &config.Config{}}}

	// 相对穿越拒绝（Join 会 Clean 掉 ..，必须显式拦）。
	for _, rel := range []string{`..\..\secret.docx`, "../../secret.docx", `..\secret.docx`} {
		if _, err := a.GaeaConvertToPdf(rel); err == nil || !strings.Contains(err.Error(), "非法") {
			t.Fatalf("穿越路径 %q 应拒绝, got %v", rel, err)
		}
	}

	// 未登记的任意绝对路径拒绝（fail-closed——此前绝对路径直通）。
	outside := filepath.Join(t.TempDir(), "outside.docx")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaConvertToPdf(outside); err == nil || !strings.Contains(err.Error(), "文件对话框") {
		t.Fatalf("未登记绝对路径应拒绝, got %v", err)
	}

	// 经 GaeaPickFiles 登记后过守卫（用 .doc 命中扩展名拒绝分支——证明已过
	// 路径门且不触发真实 LibreOffice 转换；.docx 会真转，测试机装了
	// soffice 时反而无法断言）。
	outsideDoc := filepath.Join(t.TempDir(), "outside.doc")
	if err := os.WriteFile(outsideDoc, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	registerPickedFile(outsideDoc)
	if _, err := a.GaeaConvertToPdf(outsideDoc); err == nil || !strings.Contains(err.Error(), ".doc") {
		t.Fatalf("登记路径应过守卫并命中扩展名分支, got %v", err)
	}

	// 工作区内相对路径照常过守卫（同样以 .doc 分支验证，不触发真实转换）。
	if err := os.WriteFile("local.doc", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaConvertToPdf("local.doc"); err == nil || !strings.Contains(err.Error(), ".doc") {
		t.Fatalf("工作区文件应过守卫并命中扩展名分支, got %v", err)
	}
}

// v4.363 真机走查补：绘梦历史图落 cfg.ImageSaveDir / 默认 Pictures\gaea，
// 不在「工作区+数据根」两根内——withinReadRoots 必须覆盖第三根（v4.358 误伤修复）。
func TestWithinReadRootsCoversImageSaveDir(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	a := &App{core: &core{cfg: &config.Config{}}}
	dir := filepath.Join(t.TempDir(), "Pictures", "gaea")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "gen.png")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 未配置 ImageSaveDir：默认兜底根（真实 %USERPROFILE%\Pictures\gaea）纯前缀
	// 判断命中（不写用户真实目录）。
	if !a.withinReadRoots(filepath.Join(os.Getenv("USERPROFILE"), "Pictures", "gaea", "gen.png")) {
		t.Fatalf("默认图片目录兜底根应命中")
	}

	// 显式配置 ImageSaveDir：配置根命中。
	custom := filepath.Join(t.TempDir(), "custom-img")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	a.cfg.ImageSaveDir = custom
	if !a.withinReadRoots(filepath.Join(custom, "x.png")) {
		t.Fatalf("配置的图片目录应可读: %s", custom)
	}
}

// TestAttachmentDataURLRelativePath 相对路径根修（v4.405.1）：工作区相对形态
// （.gaea/work/<文档>/插图/x.png）曾因 withinReadRoots 只认绝对路径被整类拒绝
// （2026-09-23 18:49 用户真机实录）。解析到工作区根后：根内放行+真实读出
// data URL；相对穿越仍拒；写侧同惯例。
func TestAttachmentDataURLRelativePath(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	a := &App{core: &core{cfg: &config.Config{}}}
	rel := filepath.Join(".gaea", "work", "中秋朗诵", "插图", "04提灯.png")
	abs := filepath.Join(mustCwd(t), rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("PNGDATA"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 相对路径：根内放行
	if !a.withinReadRoots(rel) {
		t.Fatalf("工作区相对路径应可读: %s", rel)
	}
	// 端到端：真实读出 data URL
	got, err := a.GaeaAttachmentDataURL(rel)
	if err != nil {
		t.Fatalf("GaeaAttachmentDataURL 相对路径: %v", err)
	}
	if !strings.HasPrefix(got, "data:image/png;base64,") || !strings.Contains(got, "UE5HREFUQQ==") {
		t.Fatalf("应返回内容 data URL: %.60s", got)
	}
	// 相对穿越：解析后逃出工作区仍拒
	if a.withinReadRoots(filepath.Join("..", "outside.png")) {
		t.Fatal("相对穿越应拒")
	}
	if _, err := a.GaeaAttachmentDataURL(filepath.Join("..", "outside.png")); err == nil || !strings.Contains(err.Error(), "可读数据范围") {
		t.Fatalf("相对穿越应报范围错: %v", err)
	}
	// 写侧同惯例：根内相对路径可写判定
	if !withinWriteRoots(filepath.Join(".gaea", "work", "out.png")) {
		t.Fatal("写侧相对路径应命中工作区根")
	}
}

// mustCwd 返回当前进程工作目录（workspaceTestIsolate 已 chdir 到临时根）。
func mustCwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}
