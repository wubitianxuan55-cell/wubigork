package app

// wenxue_tools_test.go — 图文专属工具四族单测（2026-11-15 刀3）。
// GAEA_TUWEN_ROOT 指向临时目录，实测 章节解析/覆盖闸/角色卡自动标题/时间线
// 解析与追加/统计快照 全链（纯 Go 零引擎依赖，hermetic）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

func wenxueToolsTestRoot(t *testing.T) string {
	t.Helper()
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	root := wenxueWorkspace()
	if err := os.MkdirAll(filepath.Join(root, "章节"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func mustRun(t *testing.T, tool *wenxueTool, args string) string {
	t.Helper()
	out, err := tool.Execute(t.Context(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("%s 执行失败: %v", tool.name, err)
	}
	return out
}

func TestWenxueChapterWriteListReadFlow(t *testing.T) {
	root := wenxueToolsTestRoot(t)

	// 新建第一章
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"开局","content":"雨夜。巷口的路灯忽明忽暗。他握紧了刀。"}`)
	// 同号写入沿用原文件名（保命名），overwrite=false 已存在即拒
	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(`{"number":1,"content":"覆盖尝试"}`)); err == nil {
		t.Fatal("overwrite=false 对已存在章节应拒绝")
	}
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"content":"雨夜。巷口的路灯忽明忽暗。他握紧了刀，刀柄上的红绳是他妹妹编的。","overwrite":true}`)
	// 第二章（衔接提示带第一章）
	out := mustRun(t, wenxueChapterWriteTool(), `{"number":2,"title":"追查","content":"第二天清晨，他在废料站找到了第一件证物。"}`)
	if !wenxueContains(out, "001-开局.md") {
		t.Fatalf("第二章回执应带衔接提示（含第一章文件名）: %s", out)
	}

	// 清单：解析+排序+字数
	list := mustRun(t, wenxueChapterListTool(), `{}`)
	if !wenxueContains(list, `"count":2`) || !wenxueContains(list, `001-开局.md`) || !wenxueContains(list, `002-追查.md`) {
		t.Fatalf("章节清单不符: %s", list)
	}

	// 读取：number 定位 + prevTail
	read := mustRun(t, wenxueChapterReadTool(), `{"number":2}`)
	if !wenxueContains(read, "证物") || !wenxueContains(read, "prevTail") {
		t.Fatalf("章节读取缺正文或衔接段: %s", read)
	}

	_ = root
}

func TestWenxueChapterWriteReportsCountRuleAndWorkspaceInvalidation(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	out := mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"口径","content":"雨夜。\n风停了。"}`)
	var receipt struct {
		Words         int    `json:"words"`
		ContentChars  int    `json:"contentChars"`
		WordCountRule string `json:"wordCountRule"`
	}
	if err := json.Unmarshal([]byte(out), &receipt); err != nil {
		t.Fatalf("章节回执不是 JSON: %v\n%s", err, out)
	}
	if receipt.Words != 7 || receipt.ContentChars != 8 {
		t.Fatalf("字数口径不符: words=%d contentChars=%d, want 7/8", receipt.Words, receipt.ContentChars)
	}
	if !wenxueContains(receipt.WordCountRule, "非空白") || !wenxueContains(receipt.WordCountRule, "contentChars") {
		t.Fatalf("回执必须说明计数规则: %q", receipt.WordCountRule)
	}

	paths := wenxueChapterWriteTool().CachePaths(nil)
	if len(paths) != 1 || filepath.Clean(paths[0]) != filepath.Clean(root) {
		t.Fatalf("图文写工具应失效整本工作区缓存: %v (root=%s)", paths, root)
	}
}

func TestWenxueCharacterWriteAutoTitle(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	out := mustRun(t, wenxueCharacterWriteTool(), `{"name":"林晚","content":"设定：法医，32 岁。"}`)
	if !wenxueContains(out, "林晚.md") {
		t.Fatalf("角色卡落点不符: %s", out)
	}
	// 已存在拒
	if _, err := wenxueCharacterWriteTool().Execute(t.Context(), json.RawMessage(`{"name":"林晚","content":"改动"}`)); err == nil {
		t.Fatal("overwrite=false 对已存在角色卡应拒绝")
	}
	// 清单可见 + 名称取自首行标题
	list := mustRun(t, wenxueCharacterListTool(nil), `{}`)
	if !wenxueContains(list, `"name":"林晚"`) {
		t.Fatalf("角色清单缺林晚: %s", list)
	}
	_ = root
}

func TestWenxueTimelineAppendReadParse(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	mustRun(t, wenxueTimelineAppendTool(), `{"chapter":1,"text":"巷口初遇尸体。"}`)
	mustRun(t, wenxueTimelineAppendTool(), `{"chapter":2,"text":"废料站找到证物。"}`)
	mustRun(t, wenxueTimelineAppendTool(), `{"chapter":12,"text":"真相反转。"}`)

	// 全量读（章节序）
	all := mustRun(t, wenxueTimelineReadTool(), `{}`)
	if !wenxueContains(all, "真相反转") || !wenxueContains(all, `"count":3`) {
		t.Fatalf("时间线全量读不符: %s", all)
	}
	// since 过滤：第 12 章起只剩 1 条
	since := mustRun(t, wenxueTimelineReadTool(), `{"sinceChapter":12}`)
	if !wenxueContains(since, "真相反转") || wenxueContains(since, "巷口初遇") {
		t.Fatalf("sinceChapter 过滤不符: %s", since)
	}
	_ = root
}

func TestWenxueStatsSnapshot(t *testing.T) {
	wenxueToolsTestRoot(t)
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"开局","content":"雨夜。"}`)
	mustRun(t, wenxueCharacterWriteTool(), `{"name":"林晚","content":"设定。"}`)
	mustRun(t, wenxueTimelineAppendTool(), `{"chapter":1,"text":"开局事件。"}`)
	stats := mustRun(t, wenxueStatsTool(), `{}`)
	for _, want := range []string{`"chapterCount":1`, `"characterCards":1`, `"timelineEvents":1`, `"settingFiles":0`} {
		if !wenxueContains(stats, want) {
			t.Fatalf("统计快照缺 %s: %s", want, stats)
		}
	}
}

func TestWenxueFinalDomainIsolation(t *testing.T) {
	wenxueToolsTestRoot(t)
	// 定稿写入（final_write 建目录+落盘）
	mustRun(t, wenxueFinalWriteTool(), `{"number":1,"title":"开局","content":"发布稿：雨夜。"}`)
	// 章节工具拒写定稿目录（防草稿入库）
	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(`{"path":"定稿/001-开局.md","content":"草稿误入"}`)); err == nil {
		t.Fatal("章节工具写定稿目录应被拒绝（域隔离）")
	}
	// 定稿工具拒写定稿目录之外
	if _, err := wenxueFinalWriteTool().Execute(t.Context(), json.RawMessage(`{"path":"章节/001-开局.md","content":"越域"}`)); err == nil {
		t.Fatal("定稿工具写章节目录应被拒绝（域隔离）")
	}
	// 清单/读取只在定稿域
	list := mustRun(t, wenxueFinalListTool(), `{}`)
	if !wenxueContains(list, `"count":1`) {
		t.Fatalf("定稿清单应含 1 章: %s", list)
	}
	read := mustRun(t, wenxueFinalReadTool(), `{"number":1}`)
	if !wenxueContains(read, "发布稿：雨夜。") {
		t.Fatalf("定稿读取不符: %s", read)
	}
}

func TestWenxueCreateNovelScaffoldsFinalDir(t *testing.T) {
	wenxueToolsTestRoot(t)
	a := &App{}
	if _, err := a.WenxueCreateNovel("脚手架书", ""); err != nil {
		t.Fatalf("建书: %v", err)
	}
	root := wenxueWorkspace()
	if info, err := os.Stat(filepath.Join(root, "定稿")); err != nil || !info.IsDir() {
		t.Fatalf("新建小说应脚手架定稿目录（err=%v）", err)
	}
}

func TestWenxueToolsRejectPathEscape(t *testing.T) {
	wenxueToolsTestRoot(t)
	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(`{"path":"../逃逸.md","content":"x"}`)); err == nil {
		t.Fatal("穿越路径应被拒绝")
	}
	if _, err := wenxueChapterReadTool().Execute(t.Context(), json.RawMessage(`{"path":"../../外部.md"}`)); err == nil {
		t.Fatal("穿越路径读取应被拒绝")
	}
}

func TestWenxueCandidateWriteAndPromote(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"断点","content":"雨停在凌晨。"}`)

	candidateOut := mustRun(t, wenxueCandidateWriteTool(), `{"number":1,"label":"A","content":"候选甲：他停在门前。"}`)
	var candidate struct {
		Path     string `json:"candidate"`
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal([]byte(candidateOut), &candidate); err != nil {
		t.Fatal(err)
	}
	mustRun(t, wenxueCandidateWriteTool(), `{"number":1,"label":"B","content":"候选乙：他转身离开。"}`)
	if !wenxueContains(candidate.Path, "章节/候选/001-断点/A.md") {
		t.Fatalf("候选落点不符: %s", candidate.Path)
	}
	promoteOut := mustRun(t, wenxueCandidatePromoteTool(), `{"candidatePath":"`+candidate.Path+`","expectedRevision":"`+candidate.Revision+`"}`)
	if !wenxueContains(promoteOut, "archivedPrevious") {
		t.Fatalf("提升回执缺旧稿归档: %s", promoteOut)
	}
	chapterBody, _, err := wenxueReadBody(root, "章节/001-断点.md")
	if err != nil || chapterBody != "候选甲：他停在门前。" {
		t.Fatalf("提升后正文不符: %q err=%v", chapterBody, err)
	}
	archives, err := filepath.Glob(filepath.Join(root, "章节", "候选", "001-断点", "原稿-*.md"))
	if err != nil || len(archives) != 1 {
		t.Fatalf("旧正文应归档一份，got=%v err=%v", archives, err)
	}
	archived, err := os.ReadFile(archives[0])
	if err != nil || !wenxueContains(string(archived), "雨停在凌晨。") {
		t.Fatalf("归档旧稿内容不符: %q err=%v", archived, err)
	}
}

func wenxueContains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// TestWenxueBaseToolsImageGenWorkTagged 图文生图面回归（2026-11-16）：媒体根
// 分档裁决的 image_gen 此前被 work 空间闸静默过滤（play 名表），wxWorkSpaceTool
// 适配后图文装配面在场；办公 gaeaExtraTools 原样 play 语义不变（红线）。
func TestWenxueBaseToolsImageGenWorkTagged(t *testing.T) {
	_ = wenxueToolsTestRoot(t)
	var wxTag, officeTag string
	for _, tl := range wenxueBaseTools(&App{}) {
		if tl.Name() == "image_gen" {
			wxTag = builtin.SpaceTagOf(tl)
		}
	}
	for _, tl := range gaeaExtraTools(&App{}) {
		if tl.Name() == "image_gen" {
			officeTag = builtin.SpaceTagOf(tl)
		}
	}
	if wxTag != "work" {
		t.Fatalf("图文 image_gen 空间标签 = %q, want work（生图面应装配）", wxTag)
	}
	if officeTag != "play" {
		t.Fatalf("办公 image_gen 空间标签 = %q, want play（办公语义不变红线）", officeTag)
	}
}

// TestWenxueWriteFence 写侧会话件围栏（2026-11-17 审查批2 刀2）：写族自由
// path 参数不得落 .gaea/sessions/（读侧 wenxueReadBody 围栏的对称面）。
func TestWenxueWriteFence(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	if err := os.MkdirAll(filepath.Join(root, ".gaea", "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(
		`{"path":".gaea/sessions/evil.md","content":"污染"}`)); err == nil || !strings.Contains(err.Error(), "SESSION_DENIED") {
		t.Fatalf("chapter_write 写会话件应被围栏拒绝，got %v", err)
	}
	if _, err := wenxueTimelineAppendTool().Execute(t.Context(), json.RawMessage(
		`{"text":"测试事件","file":".gaea/sessions/evil.jsonl"}`)); err == nil || !strings.Contains(err.Error(), "SESSION_DENIED") {
		t.Fatalf("timeline_append 追加会话件应被围栏拒绝，got %v", err)
	}
	if _, err := wenxueCharacterWriteTool().Execute(t.Context(), json.RawMessage(
		`{"name":"测试","content":"测试卡","path":".gaea/sessions/evil.md"}`)); err == nil || !strings.Contains(err.Error(), "SESSION_DENIED") {
		t.Fatalf("character_write 写会话件应被围栏拒绝，got %v", err)
	}
	// 正常路径不误伤：写常规目录成功。
	mustRun(t, wenxueTimelineAppendTool(), `{"text":"开局遇袭"}`)
}

// TestWenxueChapterWriteGuards 章节写入双闸（2026-11-17 审查批3）：
// 候选域对称闸（防 overwrite 绕过 candidate overwrite 闸）+000-未命名闸。
func TestWenxueChapterWriteGuards(t *testing.T) {
	_ = wenxueToolsTestRoot(t)
	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(
		`{"path":"章节/候选/001-雾港夜航/A.md","content":"候选"}`)); err == nil || !strings.Contains(err.Error(), "候选稿域") {
		t.Fatalf("chapter_write 写候选域应被拒，got %v", err)
	}
	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(
		`{"content":"正文"}`)); err == nil || !strings.Contains(err.Error(), "无法命名") {
		t.Fatalf("path 缺省且无号无题应显式报错（不再落 000-未命名.md），got %v", err)
	}
}

// TestWenxueQualityCheck 质检族（2026-11-17 审查批8 开工）：五组确定性体检
// 全链——跳号/重号/未编号、角色卡缺节与自由格式、时间线未锚、设定空文档/
// 无标题、孤儿候选；空工作区=groups 空（体检通过）。
func TestWenxueQualityCheck(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	write := func(rel, body string) {
		t.Helper()
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 章节：001 → 003（跳号）、005 两个（重号）、序章（未编号）。
	write("章节/001-开篇.md", "# 第一章 开篇\n\n雨夜。")
	write("章节/003-追查.md", "# 第三章 追查\n\n清晨。")
	write("章节/005-初见.md", "# 第五章 初见\n\n黄昏。")
	write("章节/005-重逢.md", "# 第五章 重逢\n\n入夜。")
	write("章节/序章.md", "# 序章\n\n之前。")
	// 角色卡：缺节卡 + 自由格式卡。
	write("角色/沈青梧.md", "# 沈青梧\n\n## 设定\n\n守夜人。\n\n## 外貌\n\n短发。\n")
	write("角色/自由卡.md", "# 自由卡\n\n随手记的一段人物小传，没有节结构。\n")
	// 时间线：一锚一未锚。
	write("时间线/时间线.md", "# 时间线\n\n- 第1章 · 沈青梧抵达雾港。\n- 遇袭（未锚）。\n")
	// 设定：空文档 + 无标题。
	write("设定/空文档.md", "")
	write("设定/无标题.md", "随手写的设定片段，没有一级标题。")
	// 孤儿候选：stem 不在章节清单。
	write("章节/候选/不存在章/A.md", "候选内容")

	out := mustRun(t, wenxueQualityCheckTool(), `{}`)
	var resp struct {
		Ok     bool `json:"ok"`
		Groups []struct {
			Name     string `json:"name"`
			Severity string `json:"severity"`
			Count    int    `json:"count"`
			Items    []struct {
				Path    string `json:"path"`
				Message string `json:"message"`
			} `json:"items"`
		} `json:"groups"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("回执解析: %v\n%s", err, out)
	}
	if !resp.Ok {
		t.Fatal("ok 应为 true")
	}
	groupMsg := func(gName string) []string {
		for _, g := range resp.Groups {
			if g.Name == gName {
				var msgs []string
				for _, it := range g.Items {
					msgs = append(msgs, it.Message)
				}
				return msgs
			}
		}
		return nil
	}
	if g := groupMsg("章节编号"); len(g) < 3 {
		t.Fatalf("章节编号应报跳号/重号/未编号 ≥3 条，got %v（回执 %s）", g, out)
	}
	if g := groupMsg("角色卡"); len(g) != 2 {
		t.Fatalf("角色卡应报缺节+自由格式 2 条，got %v", g)
	}
	if g := groupMsg("时间线"); len(g) != 1 {
		t.Fatalf("时间线应报未锚 1 条，got %v", g)
	}
	if g := groupMsg("设定"); len(g) != 2 {
		t.Fatalf("设定应报空文档+无标题 2 条，got %v", g)
	}
	if g := groupMsg("孤儿候选"); len(g) != 1 {
		t.Fatalf("孤儿候选应报 1 条，got %v", g)
	}

	// 空工作区：全绿=groups 空。
	emptyRoot := t.TempDir()
	t.Setenv("GAEA_TUWEN_ROOT", emptyRoot)
	out2 := mustRun(t, wenxueQualityCheckTool(), `{}`)
	if strings.Contains(out2, `"name"`) {
		t.Fatalf("空工作区体检应全绿（groups 空），got %s", out2)
	}
}
