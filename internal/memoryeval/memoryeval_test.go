package memoryeval

// 7.1-1 记忆质量受控测评测试：
//   结构（硬失败）：题集可解析/每条有预期/expected ID 全部命中种子语料/
//                   经验习得 ≥20 题/语料规模下限。
//   基线（软告警）：四域 recall@10 低于 0.8 只打 [WARN] 不阻断（阶段七口径）。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMemoryEvalSet(t *testing.T) {
	doc := "# 题集\n\n```json\n" +
		`{"story":[{"query":"q1","expected":["第1章·甲"]}]}` +
		"\n```\n"
	set, err := ParseSet([]byte(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(set.Story) != 1 || set.Story[0].Query != "q1" || set.Story[0].Expected[0] != "第1章·甲" {
		t.Errorf("set = %+v", set)
	}
	if _, err := ParseSet([]byte("# 无代码块\n")); err == nil {
		t.Error("缺 json 代码块应报错")
	}
	if _, err := ParseSet([]byte("```json\nnot-json\n```\n")); err == nil {
		t.Error("非法 JSON 应报错")
	}
}

func TestEvaluateReportMath(t *testing.T) {
	// 命中 1/2 预期、命中列表 4 条 → recall 0.5、precision 0.25。
	retrieve := func(q string) []string {
		return []string{"a", "x", "y", "z"}
	}
	report := Evaluate("t", retrieve, []Item{{Query: "q", Expected: []string{"a", "b"}}})
	if report.RecallAt10 != 0.5 {
		t.Errorf("recall = %v, want 0.5", report.RecallAt10)
	}
	if report.PrecisionAvg != 0.25 {
		t.Errorf("precision = %v, want 0.25", report.PrecisionAvg)
	}
	if report.Passed {
		t.Error("0.5 < 0.8 不应过门槛")
	}
	// 空预期不计 recall 分母；无命中不计 precision 分母。
	report = Evaluate("t", retrieve, []Item{{Query: "q"}})
	if report.RecallAt10 != 0 || report.PrecisionAvg != 0 {
		t.Errorf("空预期 report = %+v", report)
	}
	// TopK 截断：第 11 名之后的命中不参与计分。
	retrieve11 := func(q string) []string {
		hits := []string{"a"}
		for i := 0; i < 12; i++ {
			hits = append(hits, "x")
		}
		return hits
	}
	report = Evaluate("t", retrieve11, []Item{{Query: "q", Expected: []string{"a"}}})
	if report.PerQuery[0].Recall != 1 {
		t.Errorf("topk 截断后 recall = %v, want 1", report.PerQuery[0].Recall)
	}
}

// evalPersonaRoot 独立临时数据根（whisper db 按 dataRoot 单例，互踩会串库）。
func evalPersonaRoot(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "gaea-memoryeval", t.Name())
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, 0o755)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// TestMemoryEvalSetStructure 结构硬断言：题集落档且可解析、每条查询非空且
// 至少 1 个预期、expected ID 与种子语料零漂移、经验习得 ≥20、语料规模下限。
func TestMemoryEvalSetStructure(t *testing.T) {
	path, err := ResolveSetPath()
	if err != nil {
		t.Fatalf("定位题集: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读题集: %v", err)
	}
	set, err := ParseSet(data)
	if err != nil {
		t.Fatalf("解析题集: %v", err)
	}

	if len(set.Experience) < MinExperienceItems {
		t.Errorf("经验习得题集 = %d 题, want ≥ %d", len(set.Experience), MinExperienceItems)
	}

	storySeeds := StorySeeds()
	workDocs, workLabels := WorkSeeds()
	expDocs, expLabels := ExperienceSeeds()
	personaSeeds := PersonaSeeds()
	if len(storySeeds) < 50 || len(workDocs) < 40 || len(personaSeeds) < 30 || len(expDocs) < MinExperienceItems {
		t.Errorf("语料规模不足: story=%d work=%d persona=%d experience=%d",
			len(storySeeds), len(workDocs), len(personaSeeds), len(expDocs))
	}

	collect := func(items []Item) (queries, ids []string) {
		for _, it := range items {
			if strings.TrimSpace(it.Query) == "" {
				t.Errorf("空查询: %+v", it)
			}
			if len(it.Expected) == 0 {
				t.Errorf("查询 %q 无预期命中", it.Query)
			}
			ids = append(ids, it.Expected...)
		}
		return
	}

	assertKnown := func(domain string, ids []string, known func(id string) bool) {
		for _, id := range ids {
			if !known(id) {
				t.Errorf("%s 域 expected ID %q 在种子语料中不存在（题集与 seeds.go 漂移）", domain, id)
			}
		}
	}

	_, ids := collect(set.Story)
	storyIDs := map[string]bool{}
	for _, m := range storySeeds {
		storyIDs[m.ID] = true
	}
	assertKnown("story", ids, func(id string) bool { return storyIDs[id] })

	_, ids = collect(set.Work)
	assertKnown("work", ids, func(id string) bool { return contains(workLabels, id) })

	_, ids = collect(set.Persona)
	assertKnown("persona", ids, func(id string) bool {
		for _, f := range personaSeeds {
			if f.ID == id {
				return true
			}
		}
		return false
	})

	_, ids = collect(set.Experience)
	assertKnown("experience", ids, func(id string) bool { return contains(expLabels, id) })
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// TestMemoryEvalBaseline 四域基线：真实检索路径跑批，逐条与汇总数字进测试日志
// （数字即阶段七 7.1-1 的落档基线）。低于门槛 0.8 只 [WARN] 不阻断（口径：
// 质量退化告警、结构错误硬失败——见 TestMemoryEvalSetStructure）。
func TestMemoryEvalBaseline(t *testing.T) {
	path, err := ResolveSetPath()
	if err != nil {
		t.Fatalf("定位题集: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读题集: %v", err)
	}
	set, err := ParseSet(data)
	if err != nil {
		t.Fatalf("解析题集: %v", err)
	}

	var reports []DomainReport

	storySeeds := StorySeeds()
	reports = append(reports, RunStory(storySeeds, set.Story))

	workDocs, workLabels := WorkSeeds()
	reports = append(reports, RunWork(workDocs, workLabels, set.Work))

	expDocs, expLabels := ExperienceSeeds()
	reports = append(reports, RunExperience(expDocs, expLabels, set.Experience))

	personaReport, err := RunPersona(evalPersonaRoot(t), PersonaSeeds(), set.Persona)
	if err != nil {
		t.Fatalf("persona 域跑批失败: %v", err)
	}
	reports = append(reports, personaReport)

	for _, r := range reports {
		status := "PASS"
		if !r.Passed {
			status = fmt.Sprintf("[WARN] 低于门槛 %.1f", Threshold)
		}
		t.Logf("[%s] %-10s 题数=%d recall@10=%.3f precision=%.3f",
			status, r.Domain, r.Total, r.RecallAt10, r.PrecisionAvg)
		for _, q := range r.PerQuery {
			t.Logf("  %-10s recall=%.2f prec=%.2f | %s → %s",
				r.Domain, q.Recall, q.Precision, q.Query, strings.Join(q.TopHits, " | "))
		}
	}
}
