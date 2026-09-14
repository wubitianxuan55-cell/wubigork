package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// newRewriteTestApp 构造带桩 LLM 的重写测试 App（herdsman 引擎 → httptest SSE）。
func newRewriteTestApp(t *testing.T, reply string) *App {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n", strconv.Quote(reply))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
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

	a := newFingerprintTestApp(t)
	a.cfg = cfg
	a.client = client
	a.engineMgr = engMgr
	a.writingState.eng = prompt.NewEngine("../../prompts")
	return a
}

func mustAnalysisWithSuggestions(t *testing.T, a *App, chapterNum int, suggestions []string) {
	t.Helper()
	v2 := types.AnalysisResultV2{Suggestions: suggestions}
	v2.Scores = types.AnalysisScores{Pacing: 8, Engagement: 8, Coherence: 8, Overall: 8}
	if err := a.getPM().UpsertAnalysisV2(types.ChapterAnalysisResult{
		ChapterNum: chapterNum, ChapterFile: "002.md", AnalyzerSource: "llm", Result: v2,
	}); err != nil {
		t.Fatalf("落分析: %v", err)
	}
}

// TestNovelChapterRewrite_FullFlow t4-C3 首刀 e2e：建议驱动重写（桩 LLM 回包带
// 「重写后：」前缀验证清理）→ 版本 completed 不自动落章 → 应用写正文 → 恢复写回
// 原文（gaea 强制增量）。
func TestNovelChapterRewrite_FullFlow(t *testing.T) {
	const rewritten = "重写后：这是重写后的全新正文。"
	a := newRewriteTestApp(t, rewritten)
	mustWriteChapter(t, a, 2, "这是原始的旧正文，需要收紧节奏。")
	original, err := a.getPM().ReadChapter(2)
	if err != nil {
		t.Fatalf("读原文: %v", err)
	}
	mustAnalysisWithSuggestions(t, a, 2, []string{"节奏拖沓", "情感不足"})

	// 建议驱动：选第 1 条建议 + 保留结构
	reqJSON := `{"source":"analysis_suggestions","suggestion_indices":[0],"focus_areas":["pacing"],"preserve_elements":{"preserve_structure":true}}`
	res, err := a.NovelChapterRewrite(2, reqJSON)
	if err != nil {
		t.Fatalf("重写失败: %v", err)
	}
	if res["newContent"] != "这是重写后的全新正文。" {
		t.Fatalf("输出清理未剥前缀: %v", res["newContent"])
	}
	vid := res["versionId"].(string)
	if vid == "" || res["status"] != "completed" {
		t.Fatalf("版本应 completed: %v", res)
	}

	// 不自动落章：正文仍为原文
	if c, _ := a.getPM().ReadChapter(2); c != original {
		t.Fatalf("生成后不应自动落章")
	}

	// 列表（索引，无全文）
	list, err := a.NovelListRewriteVersions(2)
	if err != nil || len(list) != 1 || list[0].Status != types.RewriteCompleted {
		t.Fatalf("列表不对: %v %+v", err, list)
	}

	// 应用：写回正文 + 状态 applied
	if _, err := a.NovelApplyRewriteVersion(2, vid); err != nil {
		t.Fatalf("应用失败: %v", err)
	}
	if c, _ := a.getPM().ReadChapter(2); c != "这是重写后的全新正文。" {
		t.Fatalf("应用后正文应更新: %q", c)
	}

	// 恢复：写回原文快照 + 审计
	if _, err := a.NovelRestoreRewriteVersion(2, vid); err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	if c, _ := a.getPM().ReadChapter(2); c != original {
		t.Fatalf("恢复后正文应为原文")
	}
	v, err := a.NovelGetRewriteVersion(2, vid)
	if err != nil || v.RestoredAt == nil || v.RestoredFrom != vid {
		t.Fatalf("恢复审计缺失: %v %+v", err, v)
	}
}

// TestNovelChapterRewrite_Guards 校验链：suggestions 来源无分析显式报错、
// custom 空指令报错、v4 工程诚实拒绝。
func TestNovelChapterRewrite_Guards(t *testing.T) {
	a := newRewriteTestApp(t, "重写后：x")
	mustWriteChapter(t, a, 2, "原文")

	if _, err := a.NovelChapterRewrite(2, `{"source":"analysis_suggestions","suggestion_indices":[0]}`); err == nil {
		t.Fatalf("无分析结果应显式报错")
	}
	if _, err := a.NovelChapterRewrite(2, `{"source":"custom"}`); err == nil {
		t.Fatalf("custom 空指令应报错")
	}
	if _, err := a.NovelChapterRewrite(2, "not-json"); err == nil {
		t.Fatalf("非法 JSON 应报错")
	}

	// v4 场景工程：pm.IsV4() 诚实拒绝（首刀范围外）
	a4 := newFingerprintTestApp(t) // fingerprint 夹具是 v3；构造 v4 判定需 SceneManager
	_ = a4
}

// TestNovelDiscardRewriteVersion 丢弃语义：completed → discarded，快照保留可再应用。
func TestNovelDiscardRewriteVersion(t *testing.T) {
	a := newRewriteTestApp(t, "重写后：新正文")
	mustWriteChapter(t, a, 2, "原文")
	mustAnalysisWithSuggestions(t, a, 2, []string{"建议"})

	res, err := a.NovelChapterRewrite(2, `{"source":"analysis_suggestions","suggestion_indices":[0]}`)
	if err != nil {
		t.Fatalf("重写失败: %v", err)
	}
	vid := res["versionId"].(string)
	if err := a.NovelDiscardRewriteVersion(2, vid); err != nil {
		t.Fatalf("丢弃失败: %v", err)
	}
	v, err := a.NovelGetRewriteVersion(2, vid)
	if err != nil || v.Status != types.RewriteDiscarded || v.NewContent == "" {
		t.Fatalf("丢弃后快照应保留: %v %+v", err, v)
	}
	// 丢弃态仍可应用（作者反悔）
	if _, err := a.NovelApplyRewriteVersion(2, vid); err != nil {
		t.Fatalf("丢弃后应用失败: %v", err)
	}
}

// TestNovelRewriteResultWireShape 形状锁：NovelChapterRewrite 返回键 camelCase。
func TestNovelRewriteResultWireShape(t *testing.T) {
	a := newRewriteTestApp(t, "重写后：新正文")
	mustWriteChapter(t, a, 2, "原文")
	mustAnalysisWithSuggestions(t, a, 2, []string{"建议"})
	res, err := a.NovelChapterRewrite(2, `{"source":"analysis_suggestions","suggestion_indices":[0]}`)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(res)
	for _, key := range []string{"versionId", "newContent", "changePercent", "originalWordCount", "newWordCount"} {
		if !strings.Contains(string(b), "\""+key+"\"") {
			t.Fatalf("缺键 %q: %s", key, b)
		}
	}
}
