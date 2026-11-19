package app

// wenxue_funnel_test.go — 回写警告回执 + 事件账本 + 漏斗指标夹具
// （2026-11-19 众神殿 pan_1795084662473 P0 验收合同，action-3 五项中四项新写；
// TestWenxueSurfaceProbe 复用既有装配守卫）。口径全部来自裁决 conditions：
// 首章省略回执 / 缺失目录 none / 旁路 dirty 而非零 / 仅元数据无正文 /
// M5 零候选 n/a / 指标只称频次与待办率。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// funnelEnable 开账（创建书内 .gaea/stats/enable 标记——默认关闭形态的 opt-in）。
func funnelEnable(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".gaea", "stats"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gaea", "stats", "enable"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// setMtime 显式设文件 mtime（规避同毫秒写入的时序歧义，夹具确定性）。
func setMtime(t *testing.T, path string, mt time.Time) {
	t.Helper()
	if err := os.Chtimes(path, mt, mt); err != nil {
		t.Fatal(err)
	}
}

// TestWenxueBodyWriteEvents 事件账本四类事件行：body_write（entry/kind 不虚增）、
// candidate_write、quality_check、final；默认关闭零落盘；行内无正文。
func TestWenxueBodyWriteEvents(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	funnelEnable(t, root)

	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"开局","content":"雨夜。林晚握紧了刀。"}`)
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"content":"雨夜。林晚握紧了刀，刀柄红绳是妹妹编的。","overwrite":true}`)
	cand := mustRun(t, wenxueCandidateWriteTool(), `{"number":1,"label":"A","content":"候选稿：林晚收刀。"}`)
	var candRec struct {
		Path string `json:"candidate"`
	}
	if err := json.Unmarshal([]byte(cand), &candRec); err != nil {
		t.Fatal(err)
	}
	mustRun(t, wenxueCandidatePromoteTool(), `{"candidatePath":"`+candRec.Path+`"}`)
	mustRun(t, wenxueQualityCheckTool(), `{}`)
	mustRun(t, wenxueFinalWriteTool(), `{"number":1,"title":"开局","content":"发布稿。"}`)

	events := wenxueFunnelLoad(root)
	if len(events) != 6 {
		t.Fatalf("应 6 条事件（chapter_write×2/candidate/promote/quality/final），得 %d: %+v", len(events), events)
	}
	want := []struct {
		typ, entry, kind string
	}{
		{"body_write", "chapter_write", "create"},
		{"body_write", "chapter_write", "rewrite"},
		{"candidate_write", "", ""},
		{"body_write", "promote", "rewrite"},
		{"quality_check", "", ""},
		{"final", "", ""},
	}
	for i, w := range want {
		ev := events[i]
		if ev.Type != w.typ || ev.Entry != w.entry || ev.Kind != w.kind {
			t.Fatalf("事件 %d 应 %s/%s/%s，得 %s/%s/%s", i, w.typ, w.entry, w.kind, ev.Type, ev.Entry, ev.Kind)
		}
	}
	if events[0].Revision == "" || events[3].Revision == "" {
		t.Fatalf("body_write 事件行应带 revision 基线: %+v", events)
	}
	if events[0].Path == "" || events[3].Path == "" || events[3].Path == events[2].Path {
		t.Fatalf("body_write 记正文路径、candidate_write 记候选路径，不可混同: %+v", events[:4])
	}
	// 仅元数据红线：行内不得出现正文片段。
	raw, _ := os.ReadFile(wenxueFunnelPath(root))
	if strings.Contains(string(raw), "握紧了刀") || strings.Contains(string(raw), "发布稿") {
		t.Fatalf("事件行泄漏正文: %s", raw)
	}

	// 默认关闭：无 enable 标记的书零落盘（裁决「未授权时只实现回执」）。
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	off := wenxueToolsTestRoot(t)
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"静默","content":"账本关不落事件。"}`)
	if _, err := os.Stat(wenxueFunnelPath(off)); !os.IsNotExist(err) {
		t.Fatalf("账本默认关闭不应落盘: %v", err)
	}
}

// TestWenxueWritebackReceipt 回执四场景（裁决验收表）：
// 首章无块 / 漏回写 stale+hint 指前一章 / 已回写 updated 无 hint / 缺失目录 none。
func TestWenxueWritebackReceipt(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	funnelEnable(t, root)

	// 前置资产：角色卡（林晚）+ 时间线（有别章锚、无第1章锚）。
	mustRun(t, wenxueCharacterWriteTool(), `{"name":"林晚","content":"设定：法医。"}`)
	if err := os.MkdirAll(filepath.Join(root, "时间线"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "时间线", "时间线.md"), []byte("# 时间线\n\n- 第9章 · 远期伏笔。\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// ① 首章：无 writeback 块、无 hint。
	out1 := mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"开局","content":"雨夜。林晚握紧了刀。"}`)
	if strings.Contains(out1, `"writeback"`) || strings.Contains(out1, `"hint"`) {
		t.Fatalf("首章应省略回执块与 hint: %s", out1)
	}
	ch1 := filepath.Join(root, "章节", "001-开局.md")
	// 卡与时间线 mtime 压到第1章之前（漏回写形态确定化）。
	past := time.Now().Add(-2 * time.Hour)
	setMtime(t, ch1, past)
	setMtime(t, filepath.Join(root, "角色", "林晚.md"), past.Add(-time.Minute))
	setMtime(t, filepath.Join(root, "时间线", "时间线.md"), past.Add(-time.Minute))

	// ② 写第2章：前一章（第1章）漏回写 → stale/stale + hint 指第1章。
	out2 := mustRun(t, wenxueChapterWriteTool(), `{"number":2,"title":"追查","content":"清晨。林晚在废料站找到证物。"}`)
	var rec2 struct {
		Writeback struct {
			Characters string `json:"characters"`
			Timeline   string `json:"timeline"`
			Unobserved string `json:"unobserved"`
			Hint       string `json:"hint"`
		} `json:"writeback"`
	}
	if err := json.Unmarshal([]byte(out2), &rec2); err != nil {
		t.Fatal(err)
	}
	wb2 := rec2.Writeback
	if wb2.Characters != "stale" || wb2.Timeline != "stale" {
		t.Fatalf("漏回写应 stale/stale，得 %s/%s: %s", wb2.Characters, wb2.Timeline, out2)
	}
	if wb2.Unobserved != "clean" {
		t.Fatalf("第1章经工具写入未被旁路改动，应 clean，得 %s", wb2.Unobserved)
	}
	if !strings.Contains(wb2.Hint, "第1章") {
		t.Fatalf("hint 应指向第1章: %q", wb2.Hint)
	}

	// ③ 补回写：改卡（mtime 晚于第2章正文）+ 时间线补第2章锚（第3章的前一章
	// 是第2章——回写待办报的就是前一章）→ 写第3章 updated/updated 无 hint。
	future := time.Now().Add(time.Hour)
	setMtime(t, filepath.Join(root, "角色", "林晚.md"), future)
	mustRun(t, wenxueTimelineAppendTool(), `{"chapter":2,"text":"废料站取证。"}`)
	out3 := mustRun(t, wenxueChapterWriteTool(), `{"number":3,"title":"收网","content":"黄昏。林晚收刀入鞘。"}`)
	var rec3 struct {
		Writeback struct {
			Characters string `json:"characters"`
			Timeline   string `json:"timeline"`
			Hint       string `json:"hint"`
		} `json:"writeback"`
	}
	if err := json.Unmarshal([]byte(out3), &rec3); err != nil {
		t.Fatal(err)
	}
	wb3 := rec3.Writeback
	if wb3.Characters != "updated" || wb3.Timeline != "updated" {
		t.Fatalf("已回写应 updated/updated，得 %s/%s: %s", wb3.Characters, wb3.Timeline, out3)
	}
	if wb3.Hint != "" {
		t.Fatalf("已回写不应有 hint: %q", wb3.Hint)
	}

	// ④ 缺失目录：无角色目录/无时间线的书 → none/none 不告警。
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	bare := wenxueToolsTestRoot(t)
	funnelEnable(t, bare)
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"孤本","content":"无配套资产的书。"}`)
	outBare := mustRun(t, wenxueChapterWriteTool(), `{"number":2,"title":"续写","content":"继续。"}`)
	var recB struct {
		Writeback struct {
			Characters string `json:"characters"`
			Timeline   string `json:"timeline"`
			Hint       string `json:"hint"`
		} `json:"writeback"`
	}
	if err := json.Unmarshal([]byte(outBare), &recB); err != nil {
		t.Fatal(err)
	}
	if recB.Writeback.Characters != "none" || recB.Writeback.Timeline != "none" || recB.Writeback.Hint != "" {
		t.Fatalf("缺失目录应 none/none 无 hint，得 %+v: %s", recB.Writeback, outBare)
	}
}

// TestWenxuePromoteReceiptParity promote 与 chapter_write 回执同构：
// 同一 4 字段 writeback 块（unmarshal 同构），不再只有静态 hint。
func TestWenxuePromoteReceiptParity(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"断点","content":"雨停在凌晨。林晚立在天台。"}`)
	mustRun(t, wenxueChapterWriteTool(), `{"number":2,"title":"对峙","content":"正午。对峙。"}`)
	cand := mustRun(t, wenxueCandidateWriteTool(), `{"number":2,"label":"冷雨版","content":"冷雨版：林晚收伞。"}`)
	var candRec struct {
		Path     string `json:"candidate"`
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal([]byte(cand), &candRec); err != nil {
		t.Fatal(err)
	}
	promoteOut := mustRun(t, wenxueCandidatePromoteTool(), `{"candidatePath":"`+candRec.Path+`","expectedRevision":"`+candRec.Revision+`"}`)
	var promote struct {
		Archived  string `json:"archivedPrevious"`
		Writeback *struct {
			Characters string `json:"characters"`
			Timeline   string `json:"timeline"`
			Unobserved string `json:"unobserved"`
			Hint       string `json:"hint"`
		} `json:"writeback"`
	}
	if err := json.Unmarshal([]byte(promoteOut), &promote); err != nil {
		t.Fatal(err)
	}
	if promote.Archived == "" {
		t.Fatalf("提升回执缺归档字段: %s", promoteOut)
	}
	if promote.Writeback == nil {
		t.Fatalf("promote 有前一章（第1章）应带 writeback 块: %s", promoteOut)
	}
	wb := promote.Writeback
	if wb.Characters == "" || wb.Timeline == "" || wb.Unobserved == "" {
		t.Fatalf("writeback 三态字段不完整（应与 chapter_write 同构）: %+v", wb)
	}
	_ = root
}

// TestWenxueFunnelMetrics 指标纯函数 + 端到端旁路检出（M6 dirty 而非零漏算）。
func TestWenxueFunnelMetrics(t *testing.T) {
	now := time.Now()
	mk := func(typ, entry, path string, wb *wenxueWriteback) wenxueFunnelEvent {
		return wenxueFunnelEvent{Type: typ, Entry: entry, Path: path, Ts: now.Add(-time.Minute).UnixMilli(), Writeback: wb}
	}
	stale := &wenxueWriteback{Characters: "stale", Timeline: "stale", Unobserved: "clean"}
	fresh := &wenxueWriteback{Characters: "updated", Timeline: "updated", Unobserved: "clean"}
	events := []wenxueFunnelEvent{
		mk("body_write", "chapter_write", "章节/001-a.md", nil),
		mk("body_write", "chapter_write", "章节/001-a.md", nil), // rewrite 不虚增 M1
		mk("body_write", "chapter_write", "章节/002-b.md", stale),
		mk("body_write", "chapter_write", "章节/003-c.md", fresh),
		{Type: "candidate_write", Path: "章节/候选/002-b/A.md", Ts: now.UnixMilli()},
		mk("body_write", "promote", "章节/002-b.md", nil),
		{Type: "quality_check", Scope: "book", Chapters: 3, Ts: now.UnixMilli()},
	}
	rep := wenxueFunnelCompute(events, now)
	if rep.BodyChaptersTotal != 3 {
		t.Fatalf("M1 首写章数应 3（rewrite 不虚增），得 %d", rep.BodyChaptersTotal)
	}
	if len(rep.BodyChaptersPerWeek) != 1 {
		t.Fatalf("同期应归一周桶: %+v", rep.BodyChaptersPerWeek)
	}
	if rep.WritebackPending == nil || rep.WritebackPending.Num != 1 || rep.WritebackPending.Den != 2 {
		t.Fatalf("M3 待办率应 1/2，得 %+v", rep.WritebackPending)
	}
	if rep.QualityFrequency == nil || rep.QualityFrequency.Num != 1 || rep.QualityFrequency.Den != 3 {
		t.Fatalf("M4 质检频次应 1/3，得 %+v", rep.QualityFrequency)
	}
	if rep.PromoteUsage == nil || rep.PromoteUsage.Num != 1 || rep.PromoteUsage.Den != 1 {
		t.Fatalf("M5 提升使用率应 1/1，得 %+v", rep.PromoteUsage)
	}
	if rep.BypassDirty == nil || rep.BypassDirty.Num != 0 || rep.BypassDirty.Den != 2 {
		t.Fatalf("M6 应 0/2，得 %+v", rep.BypassDirty)
	}
	// 加一个 dirty 观测 → 1/3。
	events = append(events, mk("body_write", "chapter_write", "章节/004-d.md",
		&wenxueWriteback{Characters: "updated", Timeline: "updated", Unobserved: "dirty"}))
	if rep := wenxueFunnelCompute(events, now); rep.BypassDirty.Num != 1 || rep.BypassDirty.Den != 3 {
		t.Fatalf("M6 加 dirty 例应 1/3，得 %+v", rep.BypassDirty)
	}
	// 零候选 n/a：无 candidate_write → PromoteUsage=nil。
	noCand := []wenxueFunnelEvent{mk("body_write", "chapter_write", "章节/001-a.md", nil)}
	if rep := wenxueFunnelCompute(noCand, now); rep.PromoteUsage != nil {
		t.Fatalf("零候选应 n/a（nil），得 %+v", rep.PromoteUsage)
	}

	// 端到端旁路：账本开；工具写第1章 → 直写改第1章（模拟通用 write_file）→
	// 写第2章：回执 unobserved=dirty、事件 M6=1/1（不当成零）。
	root := wenxueToolsTestRoot(t)
	funnelEnable(t, root)
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"开局","content":"初稿。"}`)
	if err := os.WriteFile(filepath.Join(root, "章节", "001-开局.md"), []byte("旁路改过的正文。"), 0o644); err != nil {
		t.Fatal(err)
	}
	out2 := mustRun(t, wenxueChapterWriteTool(), `{"number":2,"title":"次章","content":"次章。"}`)
	var rec struct {
		Writeback struct {
			Unobserved string `json:"unobserved"`
		} `json:"writeback"`
	}
	if err := json.Unmarshal([]byte(out2), &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Writeback.Unobserved != "dirty" {
		t.Fatalf("旁路改第1章后写第2章应 dirty，得 %s: %s", rec.Writeback.Unobserved, out2)
	}
	if rep := wenxueFunnelCompute(wenxueFunnelLoad(root), now); rep.BypassDirty == nil || rep.BypassDirty.Num != 1 || rep.BypassDirty.Den != 1 {
		t.Fatalf("账本 M6 应 1/1，得 %+v", rep.BypassDirty)
	}
}
