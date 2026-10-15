package workcost

// 跨项目对标查漏评测集（11.3 出口判据：误报率有评测集）。
//
// 语料：甲/乙/丙三个同行项目 + 被审项目「丁项目」+ 健康对照「戊项目」。
// 已知发现（必须全中，逐条钉证据）：
//   ① 丁项目缺「B支护降水」分部（3/3 同行都有）
//   ② 定额「污染土壤开挖」丁版（WP-100-4）缺「水」资源（3/4 成员都含）
// 已知不报（宁缺勿误闸，防误报）：
//   ③ 「C外委处置」分部只有 1/3 同行有 → 不报（覆盖不足）
//   ④ 定额「场地消毒」组只有 2 个成员 → 不报（组太小无多数派）
//   ⑤ 健康对照戊项目 → billGaps 空（零误报）
//   ⑥ 含「水」的成员只有 1 个的组 → 不报

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

func auditSeedStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	s := Open(gdb)
	for _, r := range []Resource{
		{Code: "RG0001", Kind: "人工", Title: "普工", Unit: "工日", CurrentPrice: 150},
		{Code: "JX0001", Kind: "机械", Title: "挖掘机 220", Unit: "台班", CurrentPrice: 1200},
		{Code: "CL0001", Kind: "材料", Title: "水", Unit: "m3", CurrentPrice: 4.5},
		{Code: "CL0002", Kind: "材料", Title: "中砂", Unit: "m3", CurrentPrice: 90},
	} {
		if _, err := s.SaveResource(r); err != nil {
			t.Fatal(err)
		}
	}

	mkProject := func(name string) int64 {
		id, err := s.UpsertBillProject(name, name+".xlsx", "项目导入", "四川", "30 天", "清单计价", nil, false, 0)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	mkItem := func(projectID int64, division, title string) {
		if _, err := s.SaveBillItem(BillItem{ProjectID: projectID, Division: division, Title: title, Unit: "m3", Quantity: 100}); err != nil {
			t.Fatal(err)
		}
	}
	mkQuota := func(code, title string, resources map[string]string) {
		items := []QuotaItem{}
		for rc, rt := range resources {
			items = append(items, QuotaItem{ResourceCode: rc, Title: rt, Quantity: 0.5})
		}
		if _, err := s.SaveQuota(Quota{Code: code, OrigCode: code, Title: title, Unit: "m3", Source: "项目导入：语料.xlsx", Items: items}, nil); err != nil {
			t.Fatal(err)
		}
	}

	// 三个同行项目：都有 A临建 + B支护降水；丙另有 C外委处置（1/3，不应拉发现）。
	ja, yi, bing := mkProject("甲项目"), mkProject("乙项目"), mkProject("丙项目")
	for _, id := range []int64{ja, yi, bing} {
		mkItem(id, "A临建", "临时围挡")
		mkItem(id, "B支护降水", "基坑降水")
	}
	mkItem(bing, "C外委处置", "水泥窑协同处置")

	// 被审项目：只有 A临建（缺 B支护降水 → 发现①）。
	ding := mkProject("丁项目")
	mkItem(ding, "A临建", "临时围挡")

	// 健康对照：A + B 都有 → 零 bill_gap（⑤）。
	wu := mkProject("戊项目")
	mkItem(wu, "A临建", "临时围挡")
	mkItem(wu, "B支护降水", "基坑降水")

	// 定额「污染土壤开挖」×4：甲/乙/丙版含 普+水+机械；丁版只缺 水 → 发现②。
	common := map[string]string{"JX0001": "挖掘机 220", "RG0001": "普工"}
	withWater := map[string]string{"CL0001": "水"}
	mkQuota("WP-100", "污染土壤开挖", mergeMaps(common, withWater))
	mkQuota("WP-100-2", "污染土壤开挖", mergeMaps(common, withWater))
	mkQuota("WP-100-3", "污染土壤开挖", mergeMaps(common, withWater))
	mkQuota("WP-100-4", "污染土壤开挖", common)

	// 「场地消毒」组只有 2 个成员（④ 不报）。
	mkQuota("XD-1", "场地消毒", map[string]string{"RG0001": "普工"})
	mkQuota("XD-2", "场地消毒", map[string]string{"RG0001": "普工", "CL0001": "水"})

	_ = wu
	return s
}

func mergeMaps(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func findKind(fs []AuditFinding, substr string) *AuditFinding {
	for i := range fs {
		if strings.Contains(fs[i].Detail, substr) || strings.Contains(fs[i].Target, substr) {
			return &fs[i]
		}
	}
	return nil
}

func TestBillAuditScan_EvalSet(t *testing.T) {
	s := auditSeedStore(t)
	projects, err := s.BillProjects()
	if err != nil {
		t.Fatal(err)
	}
	var dingID int64
	for _, p := range projects {
		if p.Name == "丁项目" {
			dingID = p.ID
		}
	}
	if dingID == 0 {
		t.Fatal("被审项目未落库")
	}

	rep, err := s.BillAuditScan(dingID)
	if err != nil {
		t.Fatal(err)
	}
	if rep.AuditedProj != "丁项目" || rep.ProjectHits != 5 {
		t.Errorf("审计目标/语料 = %s/%d, want 丁项目/5", rep.AuditedProj, rep.ProjectHits)
	}

	// ① 清单缺项：恰好 1 条（B支护降水；C外委处置 1/3 不报，A临建已有）。
	// 语料同行 = 甲/乙/丙/戊 4 个（健康对照也在语料里且都有 B支护降水）。
	if len(rep.BillGaps) != 1 {
		t.Fatalf("billGaps = %d, want 1（评测集已知发现①；实际 %+v）", len(rep.BillGaps), rep.BillGaps)
	}
	gap := rep.BillGaps[0]
	if gap.Target != "B支护降水" {
		t.Errorf("缺项分部 = %q, want B支护降水", gap.Target)
	}
	if !strings.Contains(gap.Detail, "4/4") {
		t.Errorf("覆盖率应 4/4：%q", gap.Detail)
	}
	if len(gap.Evidence) != 4 {
		t.Fatalf("证据应引用 4 个同行项目，got %d", len(gap.Evidence))
	}
	for _, ev := range gap.Evidence {
		if !strings.Contains(ev.Support, "B支护降水") {
			t.Errorf("证据支撑内容错位：%+v", ev)
		}
	}

	// ② 同名定额缺组成：恰好 1 条（WP-100-4 缺 水）。
	if len(rep.QuotaGaps) != 1 {
		t.Fatalf("quotaGaps = %d, want 1（评测集已知发现②；实际 %+v）", len(rep.QuotaGaps), rep.QuotaGaps)
	}
	qg := rep.QuotaGaps[0]
	if !strings.Contains(qg.Target, "污染土壤开挖") || !strings.Contains(qg.Detail, "WP-100-4") || !strings.Contains(qg.Detail, "水") {
		t.Errorf("发现②内容错位：%+v", qg)
	}
	// 溯源：证据 = 3 个含水的成员码。
	if len(qg.Evidence) != 3 {
		t.Fatalf("证据应 3 条，got %d", len(qg.Evidence))
	}
	for _, ev := range qg.Evidence {
		if !strings.Contains(ev.Support, "水") {
			t.Errorf("证据支撑应点名缺失资源：%+v", ev)
		}
	}
}

func TestBillAuditScan_HealthyProjectZeroFalsePositives(t *testing.T) {
	s := auditSeedStore(t)
	projects, _ := s.BillProjects()
	var wuID int64
	for _, p := range projects {
		if p.Name == "戊项目" {
			wuID = p.ID
		}
	}
	rep, err := s.BillAuditScan(wuID)
	if err != nil {
		t.Fatal(err)
	}
	// 健康对照：bill 零误报（⑤）；quota_gap 全语料口径与目标无关（仍是 WP-100-4 那条，
	// 语义正确——它本来就是语料里的真缺口，不是误报）。
	if len(rep.BillGaps) != 0 {
		t.Errorf("健康对照 billGaps 应为 0（误报闸），got %+v", rep.BillGaps)
	}
	if rep.AuditedProj != "戊项目" {
		t.Errorf("审计目标 = %q, want 戊项目", rep.AuditedProj)
	}
}

func TestAuditBillGaps_TooSmallCorpus(t *testing.T) {
	ps := []auditProject{
		{ID: 1, Name: "甲", ItemCount: 5, Divisions: map[string]int{"A临建": 5}},
		{ID: 2, Name: "乙", ItemCount: 5, Divisions: map[string]int{"A临建": 5, "B支护": 2}},
	}
	if got := AuditBillGaps(ps, 1); len(got) != 0 {
		t.Errorf("语料 <3 应零发现（宁缺勿误），got %+v", got)
	}
}

func TestAuditQuotaGaps_MinorityResourceNotReported(t *testing.T) {
	// 4 成员组：R2 只有 D 含（1/4 < 60% → 缺 R2 的 A/B/C 不报，⑥）；
	// R1 有 3/4（75% ≥ 60%）→ 缺 R1 的 D 报（真缺口）。
	qs := []auditQuota{
		{Code: "A", Title: "同工作", Unit: "m3", Resource: map[string]string{"R1": "常见资源"}},
		{Code: "B", Title: "同工作", Unit: "m3", Resource: map[string]string{"R1": "常见资源"}},
		{Code: "C", Title: "同工作", Unit: "m3", Resource: map[string]string{"R1": "常见资源"}},
		{Code: "D", Title: "同工作", Unit: "m3", Resource: map[string]string{"R2": "罕见资源"}},
	}
	got := AuditQuotaGaps(qs)
	if len(got) != 1 {
		t.Fatalf("应恰 1 条（D 缺多数派 R1），got %+v", got)
	}
	if !strings.Contains(got[0].Detail, "D") || !strings.Contains(got[0].Detail, "R1") {
		t.Errorf("发现应点名 D 缺 R1：%+v", got[0])
	}
	if strings.Contains(got[0].Detail, "R2") {
		t.Errorf("少数派资源 R2 不得出现在结论：%+v", got[0])
	}
}
