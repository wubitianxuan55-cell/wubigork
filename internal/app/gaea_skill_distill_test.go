package app

// 阶段七 7.2-2 journal 历史蒸馏 App 层测试（线 B）：状态往返与回放构建
// 先独立编译验证；依赖 internal/skilldistill（线 A 并行实现）的用例在文件
// 末尾独立 Test 函数，线 A 落盘前编译不过属预期。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/evidence"
)

// ── 状态文件往返与容错（route_suggestions 同款手法）───────────────────────

// TestSkillDistillStateRoundTrip ignore 与 crystallized（含 sessions 与技能名）
// 整写 → 再读断言；文件形状 camelCase 键 + version 1。
func TestSkillDistillStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if st := loadSkillDistillState(dir); st.Ignored == nil || len(st.Ignored) != 0 || st.Crystallized == nil || len(st.Crystallized) != 0 {
		t.Fatalf("文件缺失应回空: %+v", st)
	}
	id, id2 := "jd-01234567", "jd-89abcdef"
	st := loadSkillDistillState(dir)
	st.Ignored[id] = skillDistillDecision{DecidedAt: "2026-09-15T09:00:00Z"}
	st.Crystallized = append(st.Crystallized, skillDistillCrystallized{
		ID: id2, Skill: "weekly-report", Sessions: []string{"s1", "s2", "s3"}, DecidedAt: "2026-09-15T10:00:00Z",
	})
	if err := saveSkillDistillState(dir, st); err != nil {
		t.Fatalf("saveSkillDistillState: %v", err)
	}
	got := loadSkillDistillState(dir)
	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}
	if d, ok := got.Ignored[id]; !ok || d.DecidedAt != "2026-09-15T09:00:00Z" {
		t.Errorf("ignored 往返不符: %+v", got.Ignored)
	}
	if len(got.Crystallized) != 1 {
		t.Fatalf("crystallized = %d 条, want 1", len(got.Crystallized))
	}
	c := got.Crystallized[0]
	if c.ID != id2 || c.Skill != "weekly-report" || c.DecidedAt != "2026-09-15T10:00:00Z" {
		t.Errorf("crystallized 往返不符: %+v", c)
	}
	if len(c.Sessions) != 3 || c.Sessions[0] != "s1" || c.Sessions[2] != "s3" {
		t.Errorf("sessions 往返不符: %v", c.Sessions)
	}
	// 文件形状：version 1 + camelCase 键（与 route_suggestions 的 snake 键不同，契约钉死）
	b, err := os.ReadFile(skillDistillStatePath(dir))
	if err != nil {
		t.Fatalf("读取状态文件失败: %v", err)
	}
	for _, want := range []string{`"version": 1`, `"ignored"`, `"decidedAt"`, `"crystallized"`, `"skill"`, `"sessions"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("状态文件缺少 %q:\n%s", want, b)
		}
	}
	// 二次决定：ignored 追加、crystallized 保留（decide 追加后整写的口径）
	got.Ignored[id2] = skillDistillDecision{DecidedAt: "2026-09-15T11:00:00Z"}
	if err := saveSkillDistillState(dir, got); err != nil {
		t.Fatalf("二次整写失败: %v", err)
	}
	again := loadSkillDistillState(dir)
	if len(again.Ignored) != 2 || len(again.Crystallized) != 1 {
		t.Errorf("二次整写后应 ignored=2 crystallized=1: %+v", again)
	}
}

// TestLoadSkillDistillStateTolerant 损坏/缺字段文件均容错回空（候选可重算，状态丢得起）。
func TestLoadSkillDistillStateTolerant(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(skillDistillStatePath(dir), []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if st := loadSkillDistillState(dir); len(st.Ignored) != 0 || len(st.Crystallized) != 0 {
		t.Fatalf("损坏文件应回空: %+v", st)
	}
	if err := os.WriteFile(skillDistillStatePath(dir), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if st := loadSkillDistillState(dir); st.Ignored == nil || len(st.Ignored) != 0 || len(st.Crystallized) != 0 {
		t.Fatalf("ignored 缺省应回空（且非 nil）: %+v", st)
	}
}

// ── 回放构建（纯函数）─────────────────────────────────────────────────────

func distillDayMilli(day string) int64 {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		panic(err)
	}
	return t.UnixMilli()
}

// distillReplayFixture 4 个会话同模式（各 2 步），lastAt 分四天递增；s2 摘要
// 超 200 rune、s3 摘要带换行、s4 无摘要；s3 的步骤故意倒序入参。
func distillReplayFixture() []journalEvidence {
	mk := func(sid, day, after string, reverse bool) journalEvidence {
		at := distillDayMilli(day)
		edit := journalStep{tool: "edit_file", target: "docs/" + sid + ".md", at: at}
		write := journalStep{tool: "write_file", target: "out/" + sid + ".xlsx", after: after, at: at + 1}
		if reverse {
			return journalEvidence{sessionID: sid, lastAt: at + 1, steps: []journalStep{write, edit}}
		}
		return journalEvidence{sessionID: sid, lastAt: at + 1, steps: []journalStep{edit, write}}
	}
	return []journalEvidence{
		mk("s4", "2026-09-13", "", false),
		mk("s2", "2026-09-11", strings.Repeat("摘", journalReplaySummaryMax+50), false),
		mk("s3", "2026-09-12", "汇总当月数据\n带换行的摘要", true),
		mk("s1", "2026-09-10", "最早会话应被丢弃", false),
	}
}

// TestBuildJournalReplay 多会话分段 / ≤3 会话截断（丢最早）/ 日期段头 /
// 段内步骤按 At 正序 / 段间空行。
func TestBuildJournalReplay(t *testing.T) {
	got := buildJournalReplay(distillReplayFixture())
	for _, want := range []string{
		"── 第 1 次执行（会话 s2，2026-09-11）──",
		"── 第 2 次执行（会话 s3，2026-09-12）──",
		"── 第 3 次执行（会话 s4，2026-09-13）──",
		"【工具】edit_file docs/s3.md",
		"【工具】write_file out/s3.xlsx",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("回放缺少 %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "s1") {
		t.Error("超出 3 个会话应丢弃最早的 s1")
	}
	if n := strings.Count(got, "次执行"); n != 3 {
		t.Errorf("段数 = %d, want 3", n)
	}
	if n := strings.Count(got, "\n\n"); n < 2 {
		t.Errorf("段间应空行分隔（\\n\\n × ≥2）: %q", got)
	}
	// s3 步骤倒序入参 → 段内仍按 At 正序（edit 在 write 前）
	if strings.Index(got, "【工具】edit_file docs/s3.md") > strings.Index(got, "【工具】write_file out/s3.xlsx") {
		t.Errorf("段内步骤应按 At 正序:\n%s", got)
	}
}

// TestBuildJournalReplay_SummaryTruncate AfterSummary 前 200 rune 截断、
// 换行压平单行、空摘要不输出行。
func TestBuildJournalReplay_SummaryTruncate(t *testing.T) {
	got := buildJournalReplay(distillReplayFixture())
	var s2line string
	for _, ln := range strings.Split(got, "\n") {
		if strings.Contains(ln, strings.Repeat("摘", 20)) {
			s2line = ln
			break
		}
	}
	if s2line == "" {
		t.Fatalf("s2 摘要行缺失:\n%s", got)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(s2line, "　　→ "), "…")
	if r := len([]rune(body)); r != journalReplaySummaryMax {
		t.Errorf("摘要截断 = %d rune, want %d", r, journalReplaySummaryMax)
	}
	if !strings.HasSuffix(s2line, "…") {
		t.Error("超长摘要应带省略号")
	}
	// s3 摘要换行压平为一行（同一行同时含两段文本）
	flat := false
	for _, ln := range strings.Split(got, "\n") {
		if strings.Contains(ln, "汇总当月数据") && strings.Contains(ln, "带换行的摘要") {
			flat = true
		}
	}
	if !flat {
		t.Errorf("摘要内换行应压平单行:\n%s", got)
	}
	// s4 无摘要 → s4 段内不应出现摘要行（该段只有两行工具）
	if strings.Contains(got, "【工具】write_file out/s4.xlsx\n　　→ ") {
		t.Error("空 AfterSummary 不应输出摘要行")
	}
	if buildJournalReplay(nil) != "" {
		t.Error("空证据应返回空串")
	}
}

// TestBuildJournalReplay_SegCap 单段整体 600 rune 截断防护。
func TestBuildJournalReplay_SegCap(t *testing.T) {
	ev := journalEvidence{
		sessionID: "big", lastAt: distillDayMilli("2026-09-10") + 1,
		steps: []journalStep{
			{tool: "edit_file", target: "a.md", after: strings.Repeat("长", journalReplaySummaryMax), at: 1},
			{tool: "write_file", target: "b.xlsx", after: strings.Repeat("文", journalReplaySummaryMax), at: 2},
			{tool: "edit_file", target: "c.md", after: strings.Repeat("再", journalReplaySummaryMax), at: 3},
		},
	}
	got := buildJournalReplay([]journalEvidence{ev})
	if r := len([]rune(got)); r > journalReplaySegMax+1 { // 截断后恰 600 rune（truncRunes 不额外加长）
		t.Errorf("整段应 ≤ %d rune（含省略号），got %d:\n%s", journalReplaySegMax+1, r, got)
	}
	if !strings.Contains(got, "── 第 1 次执行（会话 big，2026-09-10）──") {
		t.Errorf("段头应保留:\n%s", got)
	}
}

// ── 依赖 internal/skilldistill（线 A）的用例：Candidates/Decide/守卫 ────────

// distillJournalFixture 临时工作区 + 隔离数据根 + 3 个会话 × 同模式
// （edit_file .md → write_file .xlsx），At 逐天递增（顺序确定）；返回
// 工作区、数据根与全库最晚一步 At（即候选 LastAt 期望值）。
func distillJournalFixture(t *testing.T) (ws, dataRoot string, wantLastAt int64) {
	t.Helper()
	ws = t.TempDir()
	isolateWorkspaceTo(t, ws)
	dataRoot = t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", dataRoot)
	st, err := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	if err != nil {
		t.Fatal(err)
	}
	day := distillDayMilli("2026-09-10")
	for k, sid := range []string{"s1", "s2", "s3"} {
		base := day + int64(k)*86400000 + int64(k)*3600000
		for _, rec := range []evidence.ChangeRecord{
			{SessionID: sid, Space: "work", Tool: "edit_file", Target: "docs/周报-" + sid + ".md", AfterSummary: "整理第 " + sid + " 份纪要", At: base, Status: evidence.StatusPendingVerify},
			{SessionID: sid, Space: "work", Tool: "write_file", Target: "out/周报-" + sid + ".xlsx", AfterSummary: "导出第 " + sid + " 份表格", At: base + 1, Status: evidence.StatusPendingVerify},
		} {
			if err := st.Append(rec); err != nil {
				t.Fatal(err)
			}
		}
		wantLastAt = base + 1
	}
	return ws, dataRoot, wantLastAt
}

// TestSkillDistillCandidatesEndToEnd 3 会话同模式 → 候选 1 条 Repeat=3；
// ignore 落状态后复算静默（判据①：拒绝即不再出现）且 Available 不变。
func TestSkillDistillCandidatesEndToEnd(t *testing.T) {
	_, _, wantLastAt := distillJournalFixture(t)
	a := &App{}
	view := a.GaeaSkillDistillCandidates()
	if !view.Available {
		t.Fatal("journal 可读且有卡 → Available=true")
	}
	if len(view.Candidates) != 1 {
		t.Fatalf("候选 = %d 条, want 1: %+v", len(view.Candidates), view.Candidates)
	}
	c := view.Candidates[0]
	if c.Repeat != 3 {
		t.Errorf("Repeat = %d, want 3", c.Repeat)
	}
	if len(c.Pattern) != 2 || c.Pattern[0] != "edit_file .md" || c.Pattern[1] != "write_file .xlsx" {
		t.Errorf("Pattern 行不符: %v", c.Pattern)
	}
	if !validSkillDistillID(c.ID) {
		t.Errorf("ID 形状不符: %q", c.ID)
	}
	if len(c.Sessions) != 3 || c.Sessions[0] != "s3" {
		t.Errorf("Sessions 应 3 个且最近优先（s3 打头）: %v", c.Sessions)
	}
	if len(c.Evidence) != 3 {
		t.Errorf("Evidence 应每会话一行共 3 行: %v", c.Evidence)
	}
	if c.LastAt != wantLastAt {
		t.Errorf("LastAt = %d, want %d", c.LastAt, wantLastAt)
	}
	if view.GeneratedAt == "" {
		t.Error("GeneratedAt 不应为空")
	}
	// ignore → 状态落盘 → 复算静默（Available 不变：journal 仍有卡）
	if err := a.GaeaSkillDistillDecide(c.ID, "ignore", ""); err != nil {
		t.Fatalf("Decide(ignore): %v", err)
	}
	view2 := a.GaeaSkillDistillCandidates()
	if len(view2.Candidates) != 0 {
		t.Fatalf("ignore 后复算应静默: %+v", view2.Candidates)
	}
	if !view2.Available {
		t.Error("journal 仍有证据卡，Available 应保持 true（空态两分：无候选 ≠ 不可用）")
	}
}

// TestSkillDistillCandidatesAvailability 目录缺失 → Available=false 空表不报错；
// 有卡但无 ≥3 重复模式 → Available=true 零候选（无候选与不可用两态分开）。
func TestSkillDistillCandidatesAvailability(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	t.Setenv("GAEA_DATA_ROOT", t.TempDir())
	a := &App{}
	if view := a.GaeaSkillDistillCandidates(); view.Available || len(view.Candidates) != 0 {
		t.Fatalf("目录缺失应 Available=false 空表: %+v", view)
	}
	// 单会话两步：重复 1 < MinRepeat，不成候选但卡已读到
	st, err := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	if err != nil {
		t.Fatal(err)
	}
	base := distillDayMilli("2026-09-10")
	for _, rec := range []evidence.ChangeRecord{
		{SessionID: "only", Space: "work", Tool: "edit_file", Target: "a.md", At: base},
		{SessionID: "only", Space: "work", Tool: "write_file", Target: "b.xlsx", At: base + 1},
	} {
		if err := st.Append(rec); err != nil {
			t.Fatal(err)
		}
	}
	if view := a.GaeaSkillDistillCandidates(); !view.Available || len(view.Candidates) != 0 {
		t.Fatalf("有卡无重复模式应 Available=true 零候选: %+v", view)
	}
}

// TestSkillDistillGuards 空 App（无 ctrl）：Candidates 不 panic 回
// Available=false；Draft 一律先挡在「办公引擎未初始化」（journal 有数据也一样）。
func TestSkillDistillGuards(t *testing.T) {
	isolateWorkspaceTo(t, t.TempDir())
	t.Setenv("GAEA_DATA_ROOT", t.TempDir())
	a := &App{}
	view := a.GaeaSkillDistillCandidates()
	if view.Available || len(view.Candidates) != 0 {
		t.Fatalf("空 App 无 journal：应 Available=false 空表且不 panic: %+v", view)
	}
	if _, err := a.GaeaSkillDistillDraft("jd-01234567"); err == nil || !strings.Contains(err.Error(), "办公引擎未初始化") {
		t.Fatalf("Draft 无 ctrl 应报办公引擎未初始化: %v", err)
	}
	// journal 有数据也无济于事——引擎守卫先于候选查找
	distillJournalFixture(t)
	if _, err := a.GaeaSkillDistillDraft("jd-01234567"); err == nil || !strings.Contains(err.Error(), "办公引擎未初始化") {
		t.Fatalf("有 journal 时 Draft 仍应先报办公引擎未初始化: %v", err)
	}
}

// TestSkillDistillDecideValidation 坏 ID / 非法 decision / crystallized 缺
// 技能名：一律触盘前拦截（不产生状态文件）。
func TestSkillDistillDecideValidation(t *testing.T) {
	isolateWorkspaceTo(t, t.TempDir())
	dataRoot := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", dataRoot)
	a := &App{}
	for _, bad := range []string{"", "jd", "jd-0123456", "jd-012345678", "JD-01234567", "jd-0123456z"} {
		if err := a.GaeaSkillDistillDecide(bad, "ignore", ""); err == nil {
			t.Errorf("坏 ID 应报错: %q", bad)
		}
	}
	if err := a.GaeaSkillDistillDecide("jd-01234567", "maybe", ""); err == nil {
		t.Error("非法 decision 应报错")
	}
	if err := a.GaeaSkillDistillDecide("jd-01234567", "crystallized", "  "); err == nil {
		t.Error("crystallized 缺技能名应报错")
	}
	if _, err := os.Stat(skillDistillStatePath(dataRoot)); !os.IsNotExist(err) {
		t.Error("校验失败不应产生状态文件")
	}
}

// TestSkillDistillDecidePersist 合法路径状态落盘：crystallized 审计条目带
// sessions 与技能名（判据③）；ignore 落 ignored；两者并存整写；未知模式报错。
func TestSkillDistillDecidePersist(t *testing.T) {
	_, dataRoot, _ := distillJournalFixture(t)
	a := &App{}
	view := a.GaeaSkillDistillCandidates()
	if len(view.Candidates) != 1 {
		t.Fatalf("候选 = %d 条, want 1", len(view.Candidates))
	}
	id := view.Candidates[0].ID
	if err := a.GaeaSkillDistillDecide("jd-ffffffff", "ignore", ""); err == nil || !strings.Contains(err.Error(), "未找到该模式") {
		t.Fatalf("未知模式应报错: %v", err)
	}
	if err := a.GaeaSkillDistillDecide(id, "crystallized", " weekly-report "); err != nil {
		t.Fatalf("Decide(crystallized): %v", err)
	}
	st := loadSkillDistillState(dataRoot)
	if len(st.Crystallized) != 1 {
		t.Fatalf("crystallized 应 1 条: %+v", st.Crystallized)
	}
	c := st.Crystallized[0]
	if c.ID != id || c.Skill != "weekly-report" || c.DecidedAt == "" {
		t.Errorf("crystallized 审计条目不符: %+v", c)
	}
	if len(c.Sessions) != 3 || c.Sessions[0] != "s3" {
		t.Errorf("审计条目应带全部证据会话（最近优先）: %v", c.Sessions)
	}
	// 结晶后候选静默（ignored ∪ crystallized 双过滤）
	if view2 := a.GaeaSkillDistillCandidates(); len(view2.Candidates) != 0 {
		t.Fatalf("crystallized 后复算应静默: %+v", view2.Candidates)
	}
	// 同一模式再 ignore 仍可追溯（原始候选不过滤已决定）
	if err := a.GaeaSkillDistillDecide(id, "ignore", ""); err != nil {
		t.Fatalf("Decide(ignore): %v", err)
	}
	st = loadSkillDistillState(dataRoot)
	if _, ok := st.Ignored[id]; !ok {
		t.Errorf("ignored 应含 %q: %+v", id, st.Ignored)
	}
	if len(st.Crystallized) != 1 {
		t.Errorf("整写应保留既有 crystallized: %+v", st.Crystallized)
	}
}
