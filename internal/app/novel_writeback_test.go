package app

// 线0 · N3/N4/N5（P1，同根因）写回路径吞错 + v4 blob/scene 分叉。
//
// 旧实现：`_ = sm.Write(sc)` / `_ = pm.WriteChapter(...)` 丢错后仍回报
// done:true（前端提示「已改写 N 句」而磁盘没变）；且 v4 写场景后未调
// syncBlobFromScenes，阅读页/检索/导出读到的整章 blob 仍是旧文。
// 修复后两个 handler 都走 writeBackRewrittenFn：写失败→error（不再报 done），
// 写成功→v4 场景路径同步 blob 投影。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
)

// ── 共用写回 helper ─────────────────────────────────────────────

// TestWriteBackRewritten_WriteFailureReturnsError 真写失败（目录缺失）时
// helper 返回中文可读 error，而不是静默成功。
func TestWriteBackRewritten_WriteFailureReturnsError(t *testing.T) {
	pm := newGateProject(t)
	if err := pm.WriteChapter(1, "旧文。"); err != nil {
		t.Fatalf("预置章节: %v", err)
	}
	// 把 chapters/ 换成同名普通文件：目录位被文件占据时写盘必失败 → 真写错误。
	// （原注入是移除目录靠临时文件创建失败；写盘路径统一走 fileutil.AtomicWrite
	// 后其 MkdirAll 会把缺失目录原样重建，注入失效——占文件对有无 MkdirAll
	// 的实现都成立。）
	v3ChapDir := filepath.Join(pm.Dir, "chapters")
	if err := os.RemoveAll(v3ChapDir); err != nil {
		t.Fatalf("移除 chapters 目录: %v", err)
	}
	if err := os.WriteFile(v3ChapDir, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("占位 chapters: %v", err)
	}
	err := writeBackRewritten(pm, 1, "", false, "改写后的新文。")
	if err == nil {
		t.Fatalf("v3 整章写失败应返回 error")
	}
	if !strings.Contains(err.Error(), "保存章节失败") || !strings.Contains(err.Error(), "第1章") {
		t.Fatalf("错误应可读且指明章节, got: %v", err)
	}

	// v4 场景读失败同样如实返回 error（绝不静默成功）。
	pm2 := newGateProject(t)
	if err := pm2.MigrateV3ToV4(); err != nil {
		t.Fatalf("迁移 v4: %v", err)
	}
	// 把章节目录做成普通文件：场景目录无法建立 → 读/写场景都失败。
	chapDir := filepath.Join(pm2.Dir, "chapters", "001")
	if err := os.WriteFile(chapDir, []byte("not a dir"), 0o644); err != nil {
		// 已存在同名目录则跳过该子断言（Windows 上不能对目录覆盖写）
		t.Logf("跳过 v4 子断言: %v", err)
		return
	}
	if err := writeBackRewritten(pm2, 1, "001-opening", true, "改写后的新文。"); err == nil {
		t.Fatalf("v4 场景读写失败应返回 error")
	}
}

// ── 一键去味（DeSlopChapterAiTaste）────────────────────────────

// mustV4SceneWithAiTaste 造 v4 项目 + 一个承载 AI 味样本的场景（不写 blob）。
func mustV4SceneWithAiTaste(t *testing.T) (*App, *project.Manager, string) {
	t.Helper()
	a := newGateEmptyApp()
	pm := newGateProject(t)
	if err := pm.MigrateV3ToV4(); err != nil {
		t.Fatalf("迁移 v4: %v", err)
	}
	a.setPM(pm)
	sm := pm.SceneManager(1)
	sc, err := sm.Create("opening", "开场")
	if err != nil {
		t.Fatalf("建场景: %v", err)
	}
	sc.Content = "她的眸光流转，带着几分笑意。\n\n夜风从窗缝里钻进来，吹得烛火一阵乱晃。"
	if err := sm.Write(sc); err != nil {
		t.Fatalf("写场景: %v", err)
	}
	return a, pm, sc.Meta.ID
}

// TestDeSlopChapterAiTaste_V4SyncsBlobProjection v4 逐场景去味后必须同步整章
// blob 投影：GetChapter（阅读页/全文检索/导出读的就是它）含新文而非旧文。
func TestDeSlopChapterAiTaste_V4SyncsBlobProjection(t *testing.T) {
	a, pm, sceneID := mustV4SceneWithAiTaste(t)
	before, err := pm.SceneManager(1).Read(sceneID)
	if err != nil {
		t.Fatalf("读场景: %v", err)
	}
	// 前置：blob 尚不存在（此前的分叉点）
	if _, err := os.Stat(pm.ChapterPath(1)); !os.IsNotExist(err) {
		t.Fatalf("前置应无 blob 文件, err=%v", err)
	}

	res, err := a.DeSlopChapterAiTaste(1)
	if err != nil {
		t.Fatalf("去味: %v", err)
	}
	if res["changes"].(int) == 0 {
		t.Fatalf("样本应产生替换: %+v", res)
	}
	after, err := pm.SceneManager(1).Read(sceneID)
	if err != nil {
		t.Fatalf("读场景: %v", err)
	}
	if after.Content == before.Content {
		t.Fatalf("场景正文应被去味改写")
	}

	blob, err := a.GetChapter(1)
	if err != nil {
		t.Fatalf("GetChapter: %v", err)
	}
	got, _ := blob["content"].(string)
	if got == "" {
		t.Fatalf("v4 去味后 blob 应被同步（旧实现此处为旧文/空）")
	}
	if got != after.Content {
		t.Fatalf("blob 应等于场景投影: blob=%q scene=%q", got, after.Content)
	}
	if strings.Contains(got, "眸光流转") {
		t.Fatalf("blob 应含去味后的新文, got: %q", got)
	}
}

// TestDeSlopChapterAiTaste_WriteFailureReportsError 写回失败（注入失败桩）时
// 必须返回 error，而不是吞错后回报 done:true + changes:N。
func TestDeSlopChapterAiTaste_WriteFailureReportsError(t *testing.T) {
	orig := writeBackRewrittenFn
	defer func() { writeBackRewrittenFn = orig }()

	var mu sync.Mutex
	var gotChapter, gotSceneID string
	var gotIsScene bool
	var gotText string
	writeBackRewrittenFn = func(pm *project.Manager, chapterNum int, sceneID string, isScene bool, text string) error {
		mu.Lock()
		gotChapter, gotSceneID, gotIsScene, gotText = strconv.Itoa(chapterNum), sceneID, isScene, text
		mu.Unlock()
		return fmt.Errorf("保存场景失败（第%d章 %s）：模拟磁盘只读", chapterNum, sceneID)
	}

	a, pm, sceneID := mustV4SceneWithAiTaste(t)
	res, err := a.DeSlopChapterAiTaste(1)
	if err == nil {
		t.Fatalf("写回失败应返回 error，实际返回 done 负载: %+v", res)
	}
	if res != nil {
		t.Fatalf("写回失败不得再回报 done 负载: %+v", res)
	}
	if !strings.Contains(err.Error(), "保存场景失败") {
		t.Fatalf("错误应可读, got: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !gotIsScene || gotSceneID != sceneID || gotChapter != "1" {
		t.Fatalf("应经 helper 写回该场景: isScene=%v scene=%q chapter=%q", gotIsScene, gotSceneID, gotChapter)
	}
	if strings.TrimSpace(gotText) == "" {
		t.Fatalf("写回文本不应为空")
	}
	// 磁盘未被改动（失败桩），blob 也不应被写出「已生效」的假象。
	if _, serr := os.Stat(pm.ChapterPath(1)); serr == nil {
		t.Fatalf("写回失败时不应产生 blob")
	}
}

// ── LLM 受限重写（RewriteChapterAiTaste）───────────────────────

// mustRewriteMockApp 构造带 mock LLM 的测试 App：mock 从请求体解析待改写句表，
// 逐句返回中性改写（与原文篇幅接近，不触发 50% 漂移闸），保证安全闸
// （复测分数下降）必过。
func mustRewriteMockApp(t *testing.T) *App {
	t.Helper()
	const neutral = "他看了她一眼，没再说话，转身走进了雨里。"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(raw, &req)
		user := ""
		for _, m := range req.Messages {
			if m.Role == "user" {
				user = m.Content
			}
		}
		var items []string
		for _, line := range strings.Split(user, "\n") {
			if i := strings.Index(line, ". "); i > 0 {
				if _, err := strconv.Atoi(strings.TrimSpace(line[:i])); err == nil {
					items = append(items, strings.TrimSpace(line[i+2:]))
				}
			}
		}
		rewrites := make([]map[string]interface{}, 0, len(items))
		for i := range items {
			rewrites = append(rewrites, map[string]interface{}{"index": i + 1, "text": neutral})
		}
		payload, _ := json.Marshal(map[string]interface{}{"rewrites": rewrites})

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n", strconv.Quote(string(payload)))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := &config.Config{}
	cfg.FuncNovelEngine = "herdsman"
	cfg.FuncNovelModel = "test-model"
	cfg.FuncNovelEnabled = true
	cfg.ActiveEngineID = "herdsman"

	engMgr := modelengine.NewManager("", "")
	if err := engMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "test-model",
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	client := ai.NewClient(cfg)
	client.SetEngineManager(engMgr)

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, mu: sync.RWMutex{}}
	a.ctx = context.Background()
	return a
}

// rewriteFixtureBody 造一段含 AI 味命中、且总分未饱和（<100）的样本正文：
// 安全闸要求复测分数严格下降，饱和文本（100 → 100）永远过不了闸。
func rewriteFixtureBody() string {
	return "她的眸光流转，带着几分笑意，像是早已看穿一切。\n\n" +
		"他攥紧了拳头，指节发白，内心充满了震撼。\n\n" +
		"夜风从窗缝里钻进来，吹得烛火一阵乱晃。"
}

// TestRewriteChapterAiTaste_V4SyncsBlobProjection v4 场景 LLM 改写后写回场景
// 并同步整章 blob：GetChapter 读到新文（含中性改写句），且 disk 场景一致。
func TestRewriteChapterAiTaste_V4SyncsBlobProjection(t *testing.T) {
	a := mustRewriteMockApp(t)
	pm := newGateProject(t)
	if err := pm.MigrateV3ToV4(); err != nil {
		t.Fatalf("迁移 v4: %v", err)
	}
	a.setPM(pm)

	body := rewriteFixtureBody()
	if err := pm.WriteChapter(3, body); err != nil {
		t.Fatalf("写 blob: %v", err)
	}
	scenes, err := a.GetChapterScenes(3) // 物化首场景承载 blob
	if err != nil || len(scenes) != 1 {
		t.Fatalf("物化场景前置失败: %v scenes=%d", err, len(scenes))
	}
	sceneID, _ := scenes[0]["id"].(string)

	res, err := a.RewriteChapterAiTaste(3)
	if err != nil {
		t.Fatalf("受限重写: %v", err)
	}
	if res["done"] != true {
		t.Fatalf("应回报 done=true: %+v", res)
	}
	if n, _ := res["rewritten"].(int); n < 1 {
		t.Fatalf("应至少改写 1 句: %+v", res)
	}

	scene, err := pm.SceneManager(3).Read(sceneID)
	if err != nil {
		t.Fatalf("读场景: %v", err)
	}
	if !strings.Contains(scene.Content, "没再说话") {
		t.Fatalf("场景正文应含改写后的句子: %q", scene.Content)
	}
	blob, err := a.GetChapter(3)
	if err != nil {
		t.Fatalf("GetChapter: %v", err)
	}
	got, _ := blob["content"].(string)
	if got != scene.Content {
		t.Fatalf("blob 应同步为场景投影（旧实现读旧文）: blob=%q scene=%q", got, scene.Content)
	}
	if !strings.Contains(got, "没再说话") {
		t.Fatalf("blob 应含改写后的新文: %q", got)
	}
}

// TestRewriteChapterAiTaste_WriteFailureReportsError 写回失败时返回 error，
// 不再回报 done:true + rewritten:N（旧实现吞错后照旧谎报）。
func TestRewriteChapterAiTaste_WriteFailureReportsError(t *testing.T) {
	orig := writeBackRewrittenFn
	defer func() { writeBackRewrittenFn = orig }()
	writeBackRewrittenFn = func(pm *project.Manager, chapterNum int, sceneID string, isScene bool, text string) error {
		return fmt.Errorf("保存场景失败（第%d章 %s）：模拟磁盘只读", chapterNum, sceneID)
	}

	a := mustRewriteMockApp(t)
	pm := newGateProject(t)
	if err := pm.MigrateV3ToV4(); err != nil {
		t.Fatalf("迁移 v4: %v", err)
	}
	a.setPM(pm)
	if err := pm.WriteChapter(3, rewriteFixtureBody()); err != nil {
		t.Fatalf("写 blob: %v", err)
	}
	if scenes, err := a.GetChapterScenes(3); err != nil || len(scenes) != 1 {
		t.Fatalf("物化场景前置失败: %v scenes=%d", err, len(scenes))
	}

	res, err := a.RewriteChapterAiTaste(3)
	if err == nil {
		t.Fatalf("写回失败应返回 error，实际返回: %+v", res)
	}
	if res != nil {
		t.Fatalf("写回失败不得回报 done 负载: %+v", res)
	}
	if !strings.Contains(err.Error(), "保存场景失败") {
		t.Fatalf("错误应可读, got: %v", err)
	}
}
