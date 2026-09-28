package novelcontext

// 小说板块优化批 2 · 线5：场景圣经预算配账（G4）与「上一章摘要」章号过滤（G5）。

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// TestRender_KeepsStyleAndThreadWhenSectionsOverflow G4：HiddenFacts 很多 +
// 伏笔几十条时，产物必须**仍含**「文风」「故事主线」两段，且总长 ≤ maxRunes。
// 旧实现把无上限的 HiddenFacts/伏笔/角色块直接拼接后从尾部截断，被砍掉的正是
// 排在末尾的这两段脊椎。
func TestRender_KeepsStyleAndThreadWhenSectionsOverflow(t *testing.T) {
	const maxRunes = 2000
	b := &SceneBible{
		Setting: strings.Repeat("世界观", 100),
		POVView: strings.Repeat("已知", 300),
		Style:   "短句收尾，少用形容词，对话推进。",
		Thread:  "主角寻找失落的剑心，重铸剑宗荣光",
	}
	for i := 0; i < 12; i++ {
		b.HiddenFacts = append(b.HiddenFacts, "秘密"+strings.Repeat("隐", 120))
	}
	for i := 0; i < 40; i++ {
		b.Foreshadows = append(b.Foreshadows, "[待规划回收章] 伏笔"+strings.Repeat("线", 80))
	}
	b.Characters = []SceneChar{
		{Name: "甲", RoleType: "主角", Status: "Alive", Location: "青阳城"},
		{Name: "乙", RoleType: "配角", Status: "Alive", Location: "北境"},
	}

	rendered := b.Render(maxRunes)

	if n := len([]rune(rendered)); n > maxRunes {
		t.Fatalf("总长超预算: %d > %d", n, maxRunes)
	}
	if !strings.Contains(rendered, "## 文风") {
		t.Fatalf("文风段被整段丢弃（G4）:\n%s", rendered)
	}
	if !strings.Contains(rendered, "## 故事主线") {
		t.Fatalf("故事主线段被整段丢弃（G4）:\n%s", rendered)
	}
	if !strings.Contains(rendered, "短句收尾") || !strings.Contains(rendered, "剑心") {
		t.Fatalf("脊椎段内容不应被截空:\n%s", rendered)
	}
	// 各区段受各自常量约束：HiddenFacts / 伏笔合计不应再是无界拼接。
	hiddenSeg := sectionBody(rendered, "## 生成约束 · 当前 POV 不知情（不得泄露）")
	if hiddenSeg != "" && len([]rune(hiddenSeg)) > hiddenBudget {
		t.Fatalf("HiddenFacts 区段应 ≤%d rune，实际 %d", hiddenBudget, len([]rune(hiddenSeg)))
	}
	foreshadowSeg := sectionBody(rendered, "## 未回收伏笔（分层约束）")
	if foreshadowSeg != "" && len([]rune(foreshadowSeg)) > foreshadowBudget {
		t.Fatalf("伏笔区段应 ≤%d rune，实际 %d", foreshadowBudget, len([]rune(foreshadowSeg)))
	}
	charSeg := sectionBody(rendered, "## 出场角色")
	if charSeg != "" && len([]rune(charSeg)) > charBudget {
		t.Fatalf("出场角色区段应 ≤%d rune，实际 %d", charBudget, len([]rune(charSeg)))
	}
}

// TestRender_TailSectionsSurviveTightBudget 预算被压到很小（调用方传的
// maxRunes 远小于常量之和）时，尾部脊椎段仍按保底占比压缩而不是被丢掉。
func TestRender_TailSectionsSurviveTightBudget(t *testing.T) {
	b := &SceneBible{
		Setting:     strings.Repeat("世界观", 200),
		Style:       strings.Repeat("风", 300),
		Thread:      strings.Repeat("线", 300),
		HiddenFacts: []string{strings.Repeat("隐", 500)},
		Foreshadows: []string{strings.Repeat("伏", 500)},
	}
	const maxRunes = 400
	rendered := b.Render(maxRunes)
	if n := len([]rune(rendered)); n > maxRunes {
		t.Fatalf("总长超预算: %d > %d", n, maxRunes)
	}
	if !strings.Contains(rendered, "## 文风") || !strings.Contains(rendered, "## 故事主线") {
		t.Fatalf("紧预算下仍应保底输出文风与故事主线:\n%s", rendered)
	}
}

// sectionBody 取某区段标题下的正文（到下一个空行/标题为止）；不存在返回 ""。
func sectionBody(rendered, title string) string {
	i := strings.Index(rendered, title)
	if i < 0 {
		return ""
	}
	rest := rendered[i+len(title):]
	rest = strings.TrimPrefix(rest, "\n\n")
	if j := strings.Index(rest, "\n## "); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
}

// TestReadPrevSummary_FiltersByChapterNum G5：兜底必须按章号过滤——
// 重生成第 1 章时「上一章」为空（不拿全书最后一章充数）；第 N 章取 N-1。
func TestReadPrevSummary_FiltersByChapterNum(t *testing.T) {
	pm, _ := newTestProject(t, "char-b") // 已含第 1 章摘要
	if err := pm.WriteChapterSummary(5, &types.ChapterSummary{
		Title: "第五章 结局", Summary: "全书最后一章的摘要（绝不能被当成上一章）",
	}); err != nil {
		t.Fatalf("写第5章摘要: %v", err)
	}
	if err := pm.WriteChapterSummary(2, &types.ChapterSummary{
		Title: "第二章 启程", Summary: "第二章摘要",
	}); err != nil {
		t.Fatalf("写第2章摘要: %v", err)
	}

	// 重生成第 1 章：没有更早的章，兜底返回 nil（旧实现会返回第 5 章）。
	if got := readPrevSummary(pm, 1); got != nil {
		t.Fatalf("第 1 章不应有「上一章」摘要，得到 %+v", got)
	}
	// 第 2 章：取第 1 章。
	got := readPrevSummary(pm, 2)
	if got == nil || !strings.Contains(got.Title, "第一章") {
		t.Fatalf("第 2 章的上一章应为第 1 章，得到 %+v", got)
	}
	// 第 3 章：磁盘上只有 1/2/5，严格小于 3 的最近者是第 2 章（不得取第 5 章）。
	got = readPrevSummary(pm, 3)
	if got == nil || !strings.Contains(got.Title, "第二章") {
		t.Fatalf("第 3 章的上一章应为第 2 章，得到 %+v", got)
	}
	// 第 6 章：最近者是第 5 章。
	got = readPrevSummary(pm, 6)
	if got == nil || !strings.Contains(got.Title, "第五章") {
		t.Fatalf("第 6 章的上一章应为第 5 章，得到 %+v", got)
	}
	// 分支章摘要（002a）不得冒充第 2 章之后的「上一章」：第 3 章仍取第 2 章。
	if err := pm.WriteChapterBranchSummary(2, "a", &types.ChapterSummary{
		Title: "分支二a", Summary: "分支摘要",
	}); err != nil {
		t.Fatalf("写分支摘要: %v", err)
	}
	got = readPrevSummary(pm, 3)
	if got == nil || got.Title != "第二章 启程" {
		t.Fatalf("分支摘要不应改变主线上一章口径，得到 %+v", got)
	}
}

// TestReadLatestChapterSummaryBefore_MissingChaptersDir 目录缺失 = 无摘要（新书
// 正常态），不报错。
func TestReadLatestChapterSummaryBefore_MissingChaptersDir(t *testing.T) {
	dir := t.TempDir()
	pm := &project.Manager{Dir: dir, Meta: &types.ProjectMeta{Title: "空项目"}}
	got, err := pm.ReadLatestChapterSummaryBefore(3)
	if err != nil || got != nil {
		t.Fatalf("chapters/ 缺失应返回 (nil,nil)，得到 %+v %v", got, err)
	}
}
