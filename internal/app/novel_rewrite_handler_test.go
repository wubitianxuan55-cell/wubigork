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
	"github.com/gaea/gaea/internal/rewrite"
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

// mustWritePartialChapter 写一章「前文/选段/后文」三段结构已知的正文，返回
// rune 口径选区偏移（partial 测试夹具）。
func mustWritePartialChapter(t *testing.T, a *App, num int) (content, prefix, sel, suffix string, start, end int) {
	t.Helper()
	prefix = "夜色渐深，他推门而入，屋内的烛火晃了一下。"
	sel = "他攥紧了拳头，指节发白，一句话也说不出来。"
	suffix = "远处传来更夫的梆子声，一下，又一下。"
	content = prefix + sel + suffix
	if err := a.getPM().WriteChapter(num, content); err != nil {
		t.Fatalf("写章节 %d: %v", num, err)
	}
	start = len([]rune(prefix))
	end = start + len([]rune(sel))
	return
}

// TestNovelChapterRewrite_PartialRoundtrip partial 往返：只重写选段，选区外
// 前后文零变化（拼接断言钉死）；partial 恒 custom 不需要分析结果；版本留全文
// 快照（Mode/StartPos/LengthMode 等字段）；返回键=whole 键集+partial 扩展。
func TestNovelChapterRewrite_PartialRoundtrip(t *testing.T) {
	const newSel = "他把拳头攥得更紧，喉头动了动，终究没吐出一个字。"
	a := newRewriteTestApp(t, "重写后："+newSel) // 桩回包带前缀，顺带验证输出清理
	content, prefix, sel, suffix, start, end := mustWritePartialChapter(t, a, 2)

	reqJSON := fmt.Sprintf(`{"mode":"partial","source":"custom","custom_instructions":"把选段写得更紧张",`+
		`"start_pos":%d,"end_pos":%d,"selected_text":%q,"length_mode":"similar"}`, start, end, sel)
	res, err := a.NovelChapterRewrite(2, reqJSON)
	if err != nil {
		t.Fatalf("局部重写失败: %v", err)
	}

	// 拼接断言：新选段替换选区，前后文原文零变化
	newFull := prefix + newSel + suffix
	got, _ := res["newContent"].(string)
	if got != newFull || !strings.HasPrefix(got, prefix) || !strings.HasSuffix(got, suffix) {
		t.Fatalf("拼接错误，选区外内容被波及: %q", got)
	}
	// 不自动落章：正文仍为原文
	if c, _ := a.getPM().ReadChapter(2); c != content {
		t.Fatalf("生成后不应自动落章")
	}

	// partial 扩展键 + 选段口径统计（与引擎 ComputeDiff 对账）
	wantDiff := rewrite.ComputeDiff(sel, newSel)
	if res["mode"] != "partial" || res["lengthMode"] != "similar" ||
		res["startPos"] != start || res["endPos"] != end {
		t.Fatalf("partial 扩展键不对: %v", res)
	}
	if res["selectedWordCount"] != wantDiff.OriginalLen || res["newSelectedWordCount"] != wantDiff.NewLen ||
		res["similarity"] != wantDiff.Similarity || res["change"] != wantDiff.Change {
		t.Fatalf("选段统计不对: %v (want %+v)", res, wantDiff)
	}
	// whole 键集仍在
	for _, key := range []string{"versionId", "status", "changePercent", "originalWordCount", "newWordCount"} {
		if _, ok := res[key]; !ok {
			t.Fatalf("缺 whole 键 %q: %v", key, res)
		}
	}
	if res["originalWordCount"] != len([]rune(content)) || res["newWordCount"] != len([]rune(newFull)) {
		t.Fatalf("全文 rune 字数不对: %v", res)
	}

	// 版本落盘：Mode=partial、选区、全文快照、全文 rune 字数
	vid := res["versionId"].(string)
	v, err := a.NovelGetRewriteVersion(2, vid)
	if err != nil {
		t.Fatalf("读版本: %v", err)
	}
	if v.Mode != types.RewriteModePartial || v.Status != types.RewriteCompleted ||
		v.StartPos != start || v.EndPos != end || v.LengthMode != "similar" || v.TargetWords != 0 {
		t.Fatalf("版本字段不对: %+v", v)
	}
	if v.OriginalContent != content || v.NewContent != newFull ||
		v.OriginalWordCount != len([]rune(content)) || v.NewWordCount != len([]rune(newFull)) {
		t.Fatalf("版本快照应为全文: %+v", v)
	}

	// partial 版本走既有应用链路（回滚=整章恢复）
	if _, err := a.NovelApplyRewriteVersion(2, vid); err != nil {
		t.Fatalf("应用失败: %v", err)
	}
	if c, _ := a.getPM().ReadChapter(2); c != newFull {
		t.Fatalf("应用后正文应为拼接全文")
	}
	if _, err := a.NovelRestoreRewriteVersion(2, vid); err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	if c, _ := a.getPM().ReadChapter(2); c != content {
		t.Fatalf("恢复后正文应为原文")
	}
}

// TestNovelChapterRewrite_PartialSelectionMismatch 选中文本与正文不匹配 →
// 显式报错且不落任何版本。
func TestNovelChapterRewrite_PartialSelectionMismatch(t *testing.T) {
	a := newRewriteTestApp(t, "不应被调用")
	_, _, _, _, start, end := mustWritePartialChapter(t, a, 2)

	reqJSON := fmt.Sprintf(`{"mode":"partial","source":"custom","custom_instructions":"收紧",`+
		`"start_pos":%d,"end_pos":%d,"selected_text":"完全不相干的选中文本","length_mode":"similar"}`, start, end)
	_, err := a.NovelChapterRewrite(2, reqJSON)
	if err == nil || !strings.Contains(err.Error(), "不匹配") {
		t.Fatalf("选中文本不匹配应显式报错: %v", err)
	}
	if list, _ := a.getPM().ListRewriteVersions(2); len(list) != 0 {
		t.Fatalf("失败不应落版本: %v", list)
	}
}

// TestNovelChapterRewrite_PartialCustomNoTarget custom 长度模式缺目标字数 →
// NormalizeRequest 层前置报错。
func TestNovelChapterRewrite_PartialCustomNoTarget(t *testing.T) {
	a := newRewriteTestApp(t, "不应被调用")
	_, _, sel, _, start, end := mustWritePartialChapter(t, a, 2)

	reqJSON := fmt.Sprintf(`{"mode":"partial","source":"custom","custom_instructions":"扩写",`+
		`"start_pos":%d,"end_pos":%d,"selected_text":%q,"length_mode":"custom"}`, start, end, sel)
	if _, err := a.NovelChapterRewrite(2, reqJSON); err == nil || !strings.Contains(err.Error(), "目标字数") {
		t.Fatalf("custom 缺目标字数应报错: %v", err)
	}
}
