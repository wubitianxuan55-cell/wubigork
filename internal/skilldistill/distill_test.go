package skilldistill

import (
	"strings"
	"testing"
	"time"
)

// pat 构造 PatternStep 的简写。
func pat(tool, shape string) PatternStep { return PatternStep{Tool: tool, Shape: shape} }

// step 构造 FlowStep 的简写（At 由调用方给）。
func step(tool, target string, at int64) FlowStep {
	return FlowStep{Tool: tool, Target: target, At: at}
}

// 同款模式的三个会话流：edit_file .md → write_file .xlsx（每次文件名不同——
// 同形不同名必须折叠为同一模式）。
func threeFlows() []Flow {
	base := int64(time.Date(2026, 9, 15, 10, 0, 0, 0, time.Local).UnixMilli())
	return []Flow{
		{SessionID: "s1", Steps: []FlowStep{
			step("edit_file", "周报/0915.md", base),
			step("write_file", "周报/0915.xlsx", base+1000),
		}},
		{SessionID: "s2", Steps: []FlowStep{
			step("edit_file", "周报/0922.md", base+weekMs),
			step("write_file", "汇总0922.xlsx", base+weekMs+1000),
		}},
		{SessionID: "s3", Steps: []FlowStep{
			step("edit_file", "reports/v2.md", base+2*weekMs),
			step("write_file", "reports/v2.xlsx", base+2*weekMs+1000),
		}},
	}
}

const weekMs = 7 * 24 * 3600 * 1000

func TestMineFlowsAggregatesAndSorts(t *testing.T) {
	base := int64(1000)
	recs := []Record{
		{SessionID: "s2", Tool: "write_file", Target: "b.xlsx", At: base + 5},
		{SessionID: "s1", Tool: "edit_file", Target: "a.md", At: base + 3},
		{SessionID: "s1", Tool: "write_file", Target: "a.xlsx", At: base + 9},
		{SessionID: "s1", Tool: "edit_file", Target: "a2.md", At: base + 1}, // 组内最早
		{SessionID: "", Tool: "edit_file", Target: "x.md", At: base},        // 空 SessionID 跳过
		{SessionID: "s3", Tool: "", Target: "y.md", At: base},               // 空 Tool 跳过
	}
	flows := MineFlows(recs)
	if len(flows) != 2 {
		t.Fatalf("期望 2 条流（空值跳过），得到 %d", len(flows))
	}
	if flows[0].SessionID != "s1" || flows[1].SessionID != "s2" {
		t.Fatalf("会话序应为 s1<s2：%v", flows)
	}
	if got := flows[0].Steps[0].Target; got != "a2.md" {
		t.Fatalf("组内应按 At 升序，首步 %q", got)
	}
	if len(flows[0].Steps) != 3 {
		t.Fatalf("s1 应 3 步，得到 %d", len(flows[0].Steps))
	}
}

func TestMineFlowsEmpty(t *testing.T) {
	if got := MineFlows(nil); len(got) != 0 {
		t.Fatalf("nil 输入应空，得到 %d", len(got))
	}
}

func TestCollapseFoldsConsecutiveSameShape(t *testing.T) {
	flow := Flow{SessionID: "s1", Steps: []FlowStep{
		step("edit_file", "a.md", 1),
		step("edit_file", "b.md", 2), // 连续同形折叠
		step("write_file", "c.xlsx", 3),
		step("edit_file", "d.md", 4), // 非连续不折叠
	}}
	got := collapse(flow.Steps)
	want := []PatternStep{pat("edit_file", ".md"), pat("write_file", ".xlsx"), pat("edit_file", ".md")}
	if len(got) != len(want) {
		t.Fatalf("折叠后期望 %d 步，得到 %d：%v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("第 %d 步不符：%+v want %+v", i, got[i], want[i])
		}
	}
}

func TestShapeOf(t *testing.T) {
	cases := []struct{ target, want string }{
		{"docs/周报.MD", ".md"},   // 大写扩展名归一
		{"a.b.xlsx", ".xlsx"},     // 最后一个扩展名
		{"noext", "*"},            // 无扩展名
		{"path/", "*"},            // 目录形
		{".hidden", ".hidden"},    // 点开头即扩展名（filepath.Ext 口径）
	}
	for _, c := range cases {
		if got := shapeOf(c.target); got != c.want {
			t.Fatalf("shapeOf(%q)=%q want %q", c.target, got, c.want)
		}
	}
}

func TestDistillExactMatchAcrossSessions(t *testing.T) {
	got := Distill(threeFlows(), nil)
	if len(got) != 1 {
		t.Fatalf("期望恰好 1 条候选，得到 %d", len(got))
	}
	c := got[0]
	if c.Repeat != 3 {
		t.Fatalf("Repeat 应 3（跨会话），得到 %d", c.Repeat)
	}
	if len(c.Pattern) != 2 || c.Pattern[0] != pat("edit_file", ".md") || c.Pattern[1] != pat("write_file", ".xlsx") {
		t.Fatalf("模式不符：%v", c.Pattern)
	}
	wantID := BuildPatternID([]PatternStep{pat("edit_file", ".md"), pat("write_file", ".xlsx")})
	if c.ID != wantID {
		t.Fatalf("ID 应确定性：%s want %s", c.ID, wantID)
	}
	if !ParsePatternID(c.ID) {
		t.Fatalf("ID 形状不合法：%s", c.ID)
	}
	// 证据会话最近优先：s3（At 最大）在首位
	if len(c.Sessions) != 3 || c.Sessions[0] != "s3" || c.Sessions[2] != "s1" {
		t.Fatalf("Sessions 应最近优先 [s3 s2 s1]：%v", c.Sessions)
	}
	if c.FirstAt >= c.LastAt {
		t.Fatalf("FirstAt/LastAt 区间错误：%d ~ %d", c.FirstAt, c.LastAt)
	}
	// 证据行含工具与文件名
	if !strings.Contains(c.Evidence[0], "edit_file") || !strings.Contains(c.Evidence[0], "v2.md") {
		t.Fatalf("证据行应含工具与文件名：%q", c.Evidence[0])
	}
}

func TestDistillBelowRepeatSilent(t *testing.T) {
	flows := threeFlows()[:2] // 仅 2 会话
	if got := Distill(flows, nil); len(got) != 0 {
		t.Fatalf("2 会话不足门槛不应出候选，得到 %d", len(got))
	}
}

func TestDistillPatternMismatchSeparates(t *testing.T) {
	// 第 4 会话工具序不同 → 独立模式，不影响原模式计数
	flows := append(threeFlows(), Flow{SessionID: "s4", Steps: []FlowStep{
		step("write_file", "a.xlsx", 1),
		step("edit_file", "a.md", 2),
	}})
	got := Distill(flows, nil)
	if len(got) != 1 || got[0].Repeat != 3 {
		t.Fatalf("乱序会话应独立模式不出声；得到 %+v", got)
	}
}

func TestDistillStepBounds(t *testing.T) {
	single := []Flow{} // 折叠后 1 步：三会话同型
	for i := 0; i < 3; i++ {
		single = append(single, Flow{SessionID: sid(i), Steps: []FlowStep{
			step("edit_file", "a.md", int64(i+1)),
		}})
	}
	if got := Distill(single, nil); len(got) != 0 {
		t.Fatalf("单步流不成流程，得到 %d", len(got))
	}
	// 折叠后 13 步（超上限）：构造 13 个交替形状
	var long []Flow
	for i := 0; i < 3; i++ {
		var steps []FlowStep
		for j := 0; j < 13; j++ {
			ext := ".md"
			if j%2 == 1 {
				ext = ".xlsx"
			}
			steps = append(steps, step("edit_file", "f"+ext, int64(j)))
		}
		long = append(long, Flow{SessionID: sid(i), Steps: steps})
	}
	if got := Distill(long, nil); len(got) != 0 {
		t.Fatalf("13 步流超上限不应参评，得到 %d", len(got))
	}
	// 折叠后恰 12 步：12 个交替形状步（md/xlsx 交替防折叠）
	var twelve []Flow
	for i := 0; i < 3; i++ {
		var steps []FlowStep
		for j := 0; j < 12; j++ {
			ext := ".md"
			if j%2 == 1 {
				ext = ".xlsx"
			}
			steps = append(steps, step("edit_file", "f"+ext, int64(j)))
		}
		twelve = append(twelve, Flow{SessionID: sid(i), Steps: steps})
	}
	if got := Distill(twelve, nil); len(got) != 1 || got[0].Repeat != 3 {
		t.Fatalf("折叠后 12 步应参评，得到 %+v", got)
	}
}

func TestDistillIgnoredSilent(t *testing.T) {
	id := BuildPatternID([]PatternStep{pat("edit_file", ".md"), pat("write_file", ".xlsx")})
	got := Distill(threeFlows(), map[string]struct{}{id: {}})
	if len(got) != 0 {
		t.Fatalf("被忽略模式应静默，得到 %d", len(got))
	}
}

func TestDistillDeterministicID(t *testing.T) {
	a := Distill(threeFlows(), nil)
	b := Distill(threeFlows(), nil)
	if len(a) != 1 || len(b) != 1 || a[0].ID != b[0].ID {
		t.Fatalf("同输入两次挖掘 ID 必须一致：%v vs %v", a, b)
	}
	// 不同模式不同 ID
	other := BuildPatternID([]PatternStep{pat("write_file", ".xlsx"), pat("edit_file", ".md")})
	if a[0].ID == other {
		t.Fatal("不同模式不得同 ID")
	}
}

func TestDistillSortAndCap(t *testing.T) {
	// 两个模式：A 命中 5 会话、B 命中 3 会话 → A 先；再补两个 3 会话模式
	// 凑 4 条候选验证截断 MaxCandidates=3。
	var flows []Flow
	patA := []FlowStep{step("edit_file", "a.md", 1), step("write_file", "a.xlsx", 2)}
	patB := []FlowStep{step("docx_apply", "b.docx", 1), step("edit_file", "b.md", 2)}
	patC := []FlowStep{step("xlsx_apply", "c.xlsx", 1), step("edit_file", "c.md", 2)}
	patD := []FlowStep{step("move_file", "d.md", 1), step("edit_file", "d2.md", 2)}
	for i := 0; i < 5; i++ {
		flows = append(flows, Flow{SessionID: "a" + sid(i), Steps: patA})
	}
	for i, ps := range [][]FlowStep{patB, patC, patD} {
		_ = i
		for j := 0; j < 3; j++ {
			flows = append(flows, Flow{SessionID: "x" + string(rune('b'+i)) + sid(j), Steps: ps})
		}
	}
	got := Distill(flows, nil)
	if len(got) != MaxCandidates {
		t.Fatalf("应截 %d 条，得到 %d", MaxCandidates, len(got))
	}
	if got[0].Repeat != 5 {
		t.Fatalf("首条应为 5 会话模式，得到 Repeat=%d", got[0].Repeat)
	}
	for i := 1; i+1 < len(got); i++ {
		if got[i].Repeat < got[i+1].Repeat {
			t.Fatalf("Repeat 应降序：%v", got)
		}
		if got[i].Repeat == got[i+1].Repeat && got[i].ID > got[i+1].ID {
			t.Fatalf("同 Repeat 应按 ID 升序：%v", got)
		}
	}
}

func TestEvidenceLineCap(t *testing.T) {
	var steps []FlowStep
	for j := 0; j < 12; j++ { // 12 步长证据行 → 必超 120 rune
		steps = append(steps, step("edit_file", "很长的文件名"+strings.Repeat("甲", 12)+".md", int64(j)))
	}
	// 交替形状防折叠：奇数步换 xlsx
	for j := 1; j < len(steps); j += 2 {
		steps[j] = step("edit_file", strings.Repeat("乙", 20)+".xlsx", int64(j))
	}
	line := evidenceLine(Flow{SessionID: "s9", Steps: steps})
	if r := len([]rune(line)); r > evidenceLineMax+1 { // 截断加 "…" 允许 +1
		t.Fatalf("证据行应截 %d rune，得到 %d", evidenceLineMax, r)
	}
	if !strings.HasSuffix(line, "…") {
		t.Fatalf("超长证据行应以省略号收尾：%q 尾部", line[maxInt(0, len(line)-10):])
	}
}

func TestBuildCandidateEvidenceCap(t *testing.T) {
	// 6 个会话命中同一模式 → Sessions/Evidence 截 5
	var flows []Flow
	for i := 0; i < 6; i++ {
		flows = append(flows, Flow{SessionID: "cap" + sid(i), Steps: []FlowStep{
			step("edit_file", "a.md", int64(i+1)),
			step("write_file", "a.xlsx", int64(i+2)),
		}})
	}
	c := buildCandidate("jd-deadbeef", []PatternStep{pat("edit_file", ".md"), pat("write_file", ".xlsx")}, flows)
	if c.Repeat != 6 {
		t.Fatalf("Repeat 应全量 6，得到 %d", c.Repeat)
	}
	if len(c.Sessions) != maxSessions || len(c.Evidence) != maxEvidenceLines {
		t.Fatalf("证据应截 5：Sessions=%d Evidence=%d", len(c.Sessions), len(c.Evidence))
	}
	if c.Sessions[0] != "cap"+sid(5) {
		t.Fatalf("最近会话应在首位：%v", c.Sessions[0])
	}
}

func TestParsePatternID(t *testing.T) {
	if ParsePatternID("jd-1a2b3c4d") != true {
		t.Fatal("合法 ID 应通过")
	}
	for _, bad := range []string{"", "jd-1A2B3C4D", "jd-1a2b3c4", "xd-1a2b3c4d", "jd-1a2b3c4dz", "jd-1a2b3c4g"} {
		if ParsePatternID(bad) {
			t.Fatalf("非法 ID %q 不应通过", bad)
		}
	}
}

func TestPatternLine(t *testing.T) {
	if got := PatternLine(pat("edit_file", ".md")); got != "edit_file .md" {
		t.Fatalf("PatternLine 不符：%q", got)
	}
}

func sid(i int) string {
	if i < 0 || i > 9 {
		return "s00"
	}
	return "s0" + string(rune('0'+i))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
