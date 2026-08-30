package standard

import "testing"

// v4.6.1 规范包机制化：注册表 + 可插拔检查器 + 聚合报告。
func TestRegistryAndLintDocument(t *testing.T) {
	// 注册表幂等（同名不重复）
	r := NewRegistry()
	r.Add(RedheadChecker{})
	r.Add(RedheadChecker{})
	r.Add(CostTableChecker{})
	if len(r.Names()) != 2 {
		t.Fatalf("注册表应去重为 2 个规范包, got %v", r.Names())
	}

	// 造价表式合规文档：全部要素齐备
	doc := `××小区 1#楼 预算书（工程名称：××小区 1#楼）
编制单位：××造价咨询；编制人：张三；编制依据：2026 定额
项目名称：土建工程
单价表：混凝土 C30 元/m³ 450
人工费 100 材料费 300 机械费 50
合计：450
编制说明：未含税金。
备注：暂估。`
	report := LintDocument("doc.md", doc, "")
	// 聚合报告 = 红头 + 造价表式：造价表式全符合，但红头要素（发文字号/版记等）
	// 缺失 → 整体 passed=false（符合预期：表式合规 ≠ 公文合规）。
	if report.Passed {
		t.Fatal("含红头缺失的聚合报告 passed 应为 false")
	}
	// 「工程名称：」应命中项目名称要素（(工程|项目)\s*(名称|概况)）。
	projectOK := false
	for _, it := range report.Issues {
		if it.Spec == "造价工程表式" && it.Element == "工程/项目名称（如：××工程预算书 / 项目概况）" && it.Found {
			projectOK = true
		}
	}
	if !projectOK {
		t.Fatalf("「工程名称」应命中项目名称要素: %+v", report.Issues)
	}

	// 聚合报告带规范包归属与分组统计
	if len(report.Issues) < 7 {
		t.Fatalf("聚合报告应含红头 7 项 + 造价表式 6 项, got %d", len(report.Issues))
	}
	specs := map[string]bool{}
	for _, it := range report.Issues {
		specs[it.Spec] = true
	}
	if !specs["GB/T 9704 红头要素"] || !specs["造价工程表式"] {
		t.Fatalf("报告应含两个规范包, got %v", specs)
	}
}

// v4.6.1 造价工程表式：六要素检出与修复建议。
func TestCostTableChecker(t *testing.T) {
	c := CostTableChecker{}
	issues := c.Check("t.md", "", "这里只有一行文字，没有表格。")
	if len(issues) != 6 {
		t.Fatalf("要素数 = %d, want 6", len(issues))
	}
	for _, it := range issues {
		if it.Spec != "造价工程表式" {
			t.Fatalf("Issue 应带规范包归属, got %q", it.Spec)
		}
		if it.Found {
			t.Errorf("空文档不应命中任何要素: %+v", it)
		}
		if it.Note == "" || it.Note == "符合" {
			t.Errorf("缺失项应有修复建议: %+v", it)
		}
	}
}
