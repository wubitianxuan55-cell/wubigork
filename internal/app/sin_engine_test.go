package app

// 原罪独立引擎刀1测试：书绑定存储/校验/存量迁移 + 工具注册表集合 + 回合输入富化。
// 引擎控制器装配（sinBuildController）依赖 provider/网络面，走 mock 走查验收，
// 不在单测内构建；这里锁纯函数面。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cache"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// newSinEngineTestApp 与 sin 系测试同口径的临时 home（sinRoot 不落真实 %AppData%）。
func newSinEngineTestApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	return newSinTestApp(t)
}

// newShelfBook 造一本合法书架项目（shelf 临时目录 + 最小 project.json），
// 并把 a.cfg.NovelsDir 指向该 shelf（containment 校验的前提）。
func newShelfBook(t *testing.T, a *App) string {
	t.Helper()
	shelf := t.TempDir()
	a.cfg.NovelsDir = shelf
	book := filepath.Join(shelf, "试读书")
	if err := os.MkdirAll(book, 0o755); err != nil {
		t.Fatalf("MkdirAll(book): %v", err)
	}
	if err := os.WriteFile(filepath.Join(book, "project.json"), []byte(`{"title":"试读书"}`), 0o644); err != nil {
		t.Fatalf("WriteFile(project.json): %v", err)
	}
	return book
}

// TestSinBookSlug slug 归一：分隔符/盘符 → '-'，不含路径语义字符。
func TestSinBookSlug(t *testing.T) {
	slug := sinBookSlug(`C:\AI\xiaoshuo\试读书`)
	if strings.ContainsAny(slug, `\/:`) {
		t.Fatalf("slug 含路径字符: %q", slug)
	}
	if slug != "C--AI-xiaoshuo-试读书" {
		t.Fatalf("slug = %q", slug)
	}
	if sinBookSlug("/home/u/书") != "-home-u-书" {
		t.Fatalf("posix slug = %q", sinBookSlug("/home/u/书"))
	}
}

// TestSinWorkspaceBindGuard 绑定校验 fail-closed：缺 project.json / 越出书架 /
// 非 sin 话题 全部拒绝；合法路径往返 + 全量清单。
func TestSinWorkspaceBindGuard(t *testing.T) {
	a := newSinEngineTestApp(t)
	book := newShelfBook(t, a)
	topic, err := a.SinTopicCreate("绑定测试")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}

	// 1) 缺 project.json 拒绝
	if _, err := a.SinWorkspaceSet(topic.ID, t.TempDir()); err == nil {
		t.Fatal("缺 project.json 的目录应被拒绝")
	}
	// 2) 越出书架拒绝
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, "野书"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "野书", "project.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SinWorkspaceSet(topic.ID, filepath.Join(outside, "野书")); err == nil {
		t.Fatal("书架外的项目应被拒绝")
	}
	// 3) 非 sin 话题拒绝（fail-closed 守卫）
	chatTopic, err := a.ChatTopicCreate("闲聊", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	if _, err := a.SinWorkspaceSet(chatTopic.ID, book); err == nil {
		t.Fatal("非 sin 话题应被拒绝绑定")
	}
	// 4) 合法绑定往返
	info, err := a.SinWorkspaceSet(topic.ID, book)
	if err != nil {
		t.Fatalf("SinWorkspaceSet: %v", err)
	}
	if !info.Bound || info.Title != "试读书" {
		t.Fatalf("info = %+v", info)
	}
	got, err := a.SinWorkspaceGet(topic.ID)
	if err != nil || !got.Bound || got.Path != book {
		t.Fatalf("Get = %+v, err=%v", got, err)
	}
	bindings := a.SinWorkspaceBindings()
	if len(bindings) != 1 || bindings[0].TopicID != topic.ID {
		t.Fatalf("bindings = %+v", bindings)
	}
	// 5) 解绑
	if err := a.SinWorkspaceUnbind(topic.ID); err != nil {
		t.Fatalf("Unbind: %v", err)
	}
	if got, _ = a.SinWorkspaceGet(topic.ID); got.Bound {
		t.Fatal("解绑后仍 Bound")
	}
}

// TestSinWorkspaceMigration 绑定时存量键迁移：notes 文件与 cast/styles 键
// storyID → 书 slug（复制不移动；目标已存在不覆盖；原键保留）。
func TestSinWorkspaceMigration(t *testing.T) {
	a := newSinEngineTestApp(t)
	book := newShelfBook(t, a)
	topic, err := a.SinTopicCreate("迁移测试")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	slug := sinBookSlug(book)

	// 1) 存量 notes 文件（storyID 键）
	srcPath, err := sinNotesPath(topic.ID)
	if err != nil {
		t.Fatalf("sinNotesPath: %v", err)
	}
	if err := os.MkdirAll(sinNotesDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	srcDoc := `{"version":1,"notes":["夜半钟声"],"outline":"三幕：起/承/合"}`
	if err := os.WriteFile(srcPath, []byte(srcDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	// 2) 存量 cast/styles 键
	sinCastMu.Lock()
	doc := loadSinCast()
	doc.Casts[topic.ID] = []string{"char_1"}
	doc.Styles[topic.ID] = "水墨风"
	if err := saveSinCast(doc); err != nil {
		t.Fatal(err)
	}
	sinCastMu.Unlock()

	if _, err := a.SinWorkspaceSet(topic.ID, book); err != nil {
		t.Fatalf("SinWorkspaceSet: %v", err)
	}

	// notes 迁移：slug 键文件存在且内容一致；原文件保留
	dstPath, err := sinNotesPath(slug)
	if err != nil {
		t.Fatalf("sinNotesPath(slug): %v", err)
	}
	raw, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("迁移后 notes 缺失: %v", err)
	}
	if string(raw) != srcDoc {
		t.Fatalf("迁移内容不一致: %q", string(raw))
	}
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatal("原 notes 文件应保留")
	}
	// cast/styles 迁移：slug 键生效（引擎读路径 sinCastCharacters 按任意键直查，
	// SinCastGet 守话题属面板绑定——slug 非话题，这里断言存储真源），原键保留
	sinCastMu.Lock()
	slugIDs := append([]string(nil), loadSinCast().Casts[slug]...)
	sinCastMu.Unlock()
	if len(slugIDs) != 1 || slugIDs[0] != "char_1" {
		t.Fatalf("cast slug 键 = %v", slugIDs)
	}
	if style := sinStoryStyle(slug); style != "水墨风" {
		t.Fatalf("style slug 键 = %q", style)
	}
	sinCastMu.Lock()
	doc = loadSinCast()
	_, oldCast := doc.Casts[topic.ID]
	sinCastMu.Unlock()
	if !oldCast {
		t.Fatal("原 cast 键应保留")
	}
}

// TestSinEngineRegistryTools 注册表集合断言：只读 builtin+元工具+sin 适配器，
// 办公写具/bash/子代理/技能/sin_export 不在面内。
func TestSinEngineRegistryTools(t *testing.T) {
	a := newSinEngineTestApp(t)
	book := newShelfBook(t, a)
	cfg := gaeaConfig.Default()
	userDir := gaeaConfig.MemoryUserDir()
	// Hephaestus.db 进程内全局句柄：测试结束显式关闭，否则 Windows 下
	// t.TempDir 清理 unlinkat 失败（句柄被 sqlite 持有）。
	t.Cleanup(func() { _ = db.CloseDatabase(userDir) })
	mem := memory.Load(memory.Options{CWD: book, UserDir: userDir, DB: db.GetDatabase(userDir), Space: "play"})
	reg := sinEngineRegistry(a, cfg, book, sinBookSlug(book), mem)

	have := map[string]bool{}
	for _, n := range reg.Names() {
		have[n] = true
	}
	want := []string{
		"read_file", "ls",
		"ask", "remember", "forget", "memory_get", "promote_session_facts",
		sinToolWebSearch, sinToolWebFetch, sinToolBookSearch, sinToolBookDownload,
		sinToolCast, sinToolNotes, sinToolOutline, sinToolIllustrate,
	}
	for _, n := range want {
		if !have[n] {
			t.Fatalf("缺少工具 %q（现有: %v）", n, reg.Names())
		}
	}
	notWant := []string{"bash", "write_file", "edit", "task", "run_skill", "install_skill", sinToolExport, "summarize_file"}
	for _, n := range notWant {
		if have[n] {
			t.Fatalf("不该装配的工具 %q 出现了", n)
		}
	}
}

// TestSinEngineAssembleInput 回合输入富化：角色/底稿块注入与空态逐字直通。
func TestSinEngineAssembleInput(t *testing.T) {
	a := newSinEngineTestApp(t)
	book := newShelfBook(t, a)
	slug := sinBookSlug(book)

	// 空态：零注入，输入逐字节直通
	in := "写一段雨夜追逐"
	if got := a.sinEngineAssembleInput(slug, in); got != in {
		t.Fatalf("空态应直通: %q", got)
	}

	// 注入态：底稿块（notes/outline 非空）出现在输入前
	dstPath, err := sinNotesPath(slug)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sinNotesDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstPath, []byte(`{"version":1,"notes":["主角怕狗"],"outline":"雨夜开场"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := a.sinEngineAssembleInput(slug, in)
	if !strings.Contains(got, "主角怕狗") || !strings.Contains(got, "雨夜开场") {
		t.Fatalf("底稿块未注入: %q", got)
	}
	if !strings.HasSuffix(got, in) {
		t.Fatalf("用户指令应在末尾: %q", got)
	}
	if strings.Contains(got, "【前情回顾】") {
		t.Fatal("引擎路径不注入历史窗口段（会话历史由引擎自带）")
	}
}

// TestSinPromptOutCompiler 提示词装配：sin 基座在、记忆块可空、L1/L2 就位。
func TestSinPromptOutCompiler(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	cfg := gaeaConfig.Default()
	book := t.TempDir()
	t.Cleanup(func() { _ = db.CloseDatabase(gaeaConfig.MemoryUserDir()) })
	sp, err := sinBuildEnginePrompt(cfg, book)
	if err != nil {
		t.Fatalf("sinBuildEnginePrompt: %v", err)
	}
	if !strings.Contains(sp.prompt, "原罪") {
		t.Fatal("系统提示词缺 sin 基座")
	}
	if strings.Contains(sp.prompt, "晨报") {
		t.Fatal("sin 不注入晨报块（work 专属）")
	}
	if sp.compiler == nil || sp.runtimeCtx == nil || sp.mem == nil {
		t.Fatal("compiler/runtimeCtx/mem 未就位")
	}
	var _ *cache.Compiler = sp.compiler
}
