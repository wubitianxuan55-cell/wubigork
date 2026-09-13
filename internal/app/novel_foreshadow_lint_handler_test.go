package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// findLintCode 取指定条目+检查码的发现（无则 nil）。
func findLintCode(findings []ForeshadowLintFinding, id, code string) *ForeshadowLintFinding {
	for i := range findings {
		if findings[i].ForeshadowID == id && findings[i].Code == code {
			return &findings[i]
		}
	}
	return nil
}

func TestChapterNumOf(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"001.md", 1}, {"012.md", 12}, {"001a.md", 1}, {"", 0}, {"abc.md", 0},
	}
	for _, c := range cases {
		if got := chapterNumOf(c.in); got != c.want {
			t.Fatalf("chapterNumOf(%q)=%d want %d", c.in, got, c.want)
		}
	}
}

func TestLintForeshadowItems_AllChecks(t *testing.T) {
	items := []types.Foreshadow{
		// 正常长线条目：不触发悬置/序/状态问题。
		{ID: "ok_1", Description: "铜匣里藏着的信", PlantedIn: "001.md", Status: types.ForeshadowPlanted, IsLongTerm: true},
		// 回收先于埋设：high ordering。
		{ID: "ord_1", Description: "消失的怀表", PlantedIn: "003.md", RevealedIn: "001.md", Status: types.ForeshadowRevealed},
		// 已回收但没填回收章：medium status-mismatch。
		{ID: "st_1", Description: "断弦的琴", PlantedIn: "001.md", Status: types.ForeshadowRevealed},
		// 填了回收章但状态没跟上：medium status-mismatch。
		{ID: "st_2", Description: "墙上的划痕", PlantedIn: "001.md", RevealedIn: "002.md", Status: types.ForeshadowHinted},
		// 指向未写章节：high dangling（埋设+回收两处）。
		{ID: "dan_1", Description: "夜半的梆子声", PlantedIn: "099.md", RevealedIn: "100.md", Status: types.ForeshadowRevealed},
		// 悬置：12 章全书，第 1 章埋设已 11 章 → medium stale。
		{ID: "stal_1", Description: "巷口忽明忽暗的灯", PlantedIn: "001.md", Status: types.ForeshadowPlanted},
		// 重复登记：与 stal_1 描述相同（含首尾空白）→ low duplicate。
		{ID: "dup_1", Description: "  巷口忽明忽暗的灯  ", PlantedIn: "002.md", Status: types.ForeshadowPlanted},
	}
	findings := lintForeshadowItems(items, 12)

	if f := findLintCode(findings, "ord_1", "ordering"); f == nil || f.Severity != "high" || !strings.Contains(f.Message, "序颠倒") {
		t.Fatalf("ordering 检查失效: %+v", f)
	}
	if f := findLintCode(findings, "st_1", "status-mismatch"); f == nil || !strings.Contains(f.Message, "未填回收章节") {
		t.Fatalf("已回收缺回收章检查失效: %+v", f)
	}
	if f := findLintCode(findings, "st_2", "status-mismatch"); f == nil || !strings.Contains(f.Message, "应改为「已回收」") {
		t.Fatalf("回收章未跟状态检查失效: %+v", f)
	}
	if f := findLintCode(findings, "dan_1", "dangling"); f == nil || f.Severity != "high" {
		t.Fatalf("dangling 检查失效: %+v", f)
	}
	// dan_1 埋设/回收两处都指向空缺 → 2 条 dangling。
	n := 0
	for _, f := range findings {
		if f.ForeshadowID == "dan_1" && f.Code == "dangling" {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("dangling 应 2 条（埋设+回收），got %d", n)
	}
	if f := findLintCode(findings, "stal_1", "stale"); f == nil || !strings.Contains(f.Message, "11 章") {
		t.Fatalf("stale 检查失效: %+v", f)
	}
	// 长线豁免：ok_1 埋设 11 章也不得触发 stale。
	if f := findLintCode(findings, "ok_1", "stale"); f != nil {
		t.Fatalf("长线应豁免悬置: %+v", f)
	}
	if f := findLintCode(findings, "dup_1", "duplicate"); f == nil || !strings.Contains(f.Message, "stal_1") {
		t.Fatalf("duplicate 检查失效: %+v", f)
	}
	// 首个登记者不算重复。
	if f := findLintCode(findings, "stal_1", "duplicate"); f != nil {
		t.Fatalf("首个登记不应报重复: %+v", f)
	}
	// 正常条目零发现。
	for _, f := range findings {
		if f.ForeshadowID == "ok_1" {
			t.Fatalf("正常长线条目不应有发现: %+v", f)
		}
	}
}

func TestLintForeshadows_EmptyRegistryIsNormal(t *testing.T) {
	a := newFingerprintTestApp(t)
	rep, err := a.LintForeshadows()
	if err != nil {
		t.Fatalf("空登记表应正常态: %v", err)
	}
	if rep.Items != 0 || rep.TotalChapters != 0 || len(rep.Findings) != 0 {
		t.Fatalf("空报告形状不对: %+v", rep)
	}
}

func TestLintForeshadows_WithFixture(t *testing.T) {
	a := newFingerprintTestApp(t)
	for i := 1; i <= 3; i++ {
		mustWriteChapter(t, a, i, "样文。")
	}
	mustSaveForeshadows(t, a, []types.Foreshadow{
		{ID: "manual_a", Category: "plot", Description: "铜匣", PlantedIn: "001.md", Status: types.ForeshadowPlanted},
		{ID: "manual_b", Category: "plot", Description: "怀表", PlantedIn: "003.md", RevealedIn: "001.md", Status: types.ForeshadowRevealed},
		{ID: "manual_c", Category: "world", Description: "怀表", PlantedIn: "005.md", Status: types.ForeshadowPlanted},
	})
	rep, err := a.LintForeshadows()
	if err != nil {
		t.Fatalf("体检失败: %v", err)
	}
	if rep.TotalChapters != 3 || rep.Items != 3 || rep.Planted != 2 || rep.Revealed != 1 {
		t.Fatalf("统计口径不对: %+v", rep)
	}
	if findLintCode(rep.Findings, "manual_b", "ordering") == nil {
		t.Fatalf("序颠倒应命中: %+v", rep.Findings)
	}
	if findLintCode(rep.Findings, "manual_c", "dangling") == nil {
		t.Fatalf("指向未写章应命中: %+v", rep.Findings)
	}
	if findLintCode(rep.Findings, "manual_c", "duplicate") == nil {
		t.Fatalf("重复描述应命中: %+v", rep.Findings)
	}
	// manual_a 埋设 2 章 < 10 门槛，不应悬置。
	if findLintCode(rep.Findings, "manual_a", "stale") != nil {
		t.Fatalf("未到门槛不应报悬置: %+v", rep.Findings)
	}
}

// TestForeshadowLintReportWireShape 形状锁：报告负载 marshal 键为 camelCase。
func TestForeshadowLintReportWireShape(t *testing.T) {
	b, err := json.Marshal(ForeshadowLintReport{TotalChapters: 3, Items: 1, Findings: []ForeshadowLintFinding{{
		Code: "ordering", Severity: "high", ForeshadowID: "x", ItemDesc: "d", Message: "m", Chapter: "001.md",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(b, &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"totalChapters", "items", "planted", "hinted", "revealed", "longTerm", "findings"} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("缺键 %q: %s", key, b)
		}
	}
	var findings []map[string]json.RawMessage
	if err := json.Unmarshal(keys["findings"], &findings); err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("应 1 条发现: %s", keys["findings"])
	}
	for _, key := range []string{"code", "severity", "foreshadowId", "itemDesc", "message"} {
		if _, ok := findings[0][key]; !ok {
			t.Fatalf("发现缺键 %q: %s", key, keys["findings"])
		}
	}
	for _, bad := range []string{"TotalChapters", "Findings", "ForeshadowID"} {
		if strings.Contains(string(b), `"`+bad+`"`) {
			t.Fatalf("出现 PascalCase 键 %q: %s", bad, b)
		}
	}
}
