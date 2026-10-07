// bill_test.go — 清单层（gf_projects + gf_bill_items）单元测试。
//
// 口径（用户定调 2026-10-07）：造价数据库只是数据库——录入费率/清单/模版，
// 不做加总。本文件钉住：项目+费率 CRUD、清单项 CRUD（工程量/引用定额）、
// 手填行自动 M 编号、幂等更新。
package workcost

import "testing"

func TestBillProjectRatesRoundtrip(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	id, err := s.UpsertBillProject("测试项目", "test.xlsx", "/tmp/test.xlsx", "乐山", "90 天", "综合单价只含人材机", nil, false, 0)
	if err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	// 新建且未给费率 → 默认 10/2/7/9。
	projects, err := s.BillProjects()
	if err != nil || len(projects) != 1 {
		t.Fatalf("项目列表应 1 个: %v", err)
	}
	if projects[0].ManagementRate != 0.1 || projects[0].TaxRate != 0.09 {
		t.Errorf("默认费率应 10%%/9%%，得到 %.3f/%.3f", projects[0].ManagementRate, projects[0].TaxRate)
	}

	// 改费率（录入数据随时可调）。
	if err := s.UpdateBillProjectRates(id, RateSet{ManagementRate: 0.12, RegulatoryRate: 0.03, ProfitRate: 0.08, TaxRate: 0.09}, true, 1200000); err != nil {
		t.Fatalf("改费率失败: %v", err)
	}
	projects, _ = s.BillProjects()
	if projects[0].ManagementRate != 0.12 || !projects[0].ProfitIncludesRegulatory || projects[0].ControlPrice != 1200000 {
		t.Errorf("费率回读不符: %+v", projects[0])
	}

	// 同名再 upsert → 更新不新建（rates=nil 不覆盖费率）。
	if _, err := s.UpsertBillProject("测试项目", "test2.xlsx", "/tmp/t2.xlsx", "", "", "", nil, false, 0); err != nil {
		t.Fatalf("重导入失败: %v", err)
	}
	projects, _ = s.BillProjects()
	if len(projects) != 1 {
		t.Fatalf("同名项目应走更新，得到 %d 个", len(projects))
	}
	if projects[0].ManagementRate != 0.12 {
		t.Errorf("rates=nil 不应覆盖费率，得到 %.3f", projects[0].ManagementRate)
	}
}

func TestBillItemSaveAndAutoCode(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	id, _ := s.UpsertBillProject("项目A", "a.xlsx", "", "", "", "", nil, false, 0)

	// 套定额的清单项（编码显式）。
	saved, err := s.SaveBillItem(BillItem{ProjectID: id, Code: "WP01", Title: "施工便道", Unit: "m", Quantity: 850, QuotaCode: "WP01"})
	if err != nil {
		t.Fatalf("保存清单项失败: %v", err)
	}
	if saved.ID == 0 {
		t.Fatal("应分配 id")
	}

	// 手填行×2：code 空 → 自动 M001/M002。
	for _, title := range []string{"外委监测", "成品保护"} {
		if _, err := s.SaveBillItem(BillItem{ProjectID: id, Title: title, Unit: "项", Quantity: 1, PriceOverride: 12000}); err != nil {
			t.Fatalf("手填行 %s 失败: %v", title, err)
		}
	}
	items, err := s.BillItems(id)
	if err != nil || len(items) != 3 {
		t.Fatalf("清单应 3 条，得到 %d（err=%v）", len(items), err)
	}
	if items[1].Code != "M001" || items[2].Code != "M002" {
		t.Errorf("手填行应自动编号 M001/M002，得到 %s/%s", items[1].Code, items[2].Code)
	}
	if items[0].Quantity != 850 {
		t.Errorf("工程量应 850，得到 %v", items[0].Quantity)
	}

	// 同编码再保存 = 更新（改工程量）。
	if _, err := s.SaveBillItem(BillItem{ProjectID: id, Code: "WP01", Title: "施工便道", Unit: "m", Quantity: 900, QuotaCode: "WP01"}); err != nil {
		t.Fatalf("更新清单项失败: %v", err)
	}
	items, _ = s.BillItems(id)
	if len(items) != 3 || items[0].Quantity != 900 {
		t.Errorf("同编码应走更新：条数 %d，工程量 %v", len(items), items[0].Quantity)
	}

	// 删除。
	if err := s.DeleteBillItem(items[2].ID); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	items, _ = s.BillItems(id)
	if len(items) != 2 {
		t.Errorf("删除后应 2 条，得到 %d", len(items))
	}

	// itemCount 聚合。
	projects, _ := s.BillProjects()
	if projects[0].ItemCount != 2 {
		t.Errorf("itemCount 聚合应 2，得到 %d", projects[0].ItemCount)
	}
}

// TestResolveQuotaCodeUnification 定额编码统一规则四态（V29）：
// 空库直用 / 同源重导入复用 / 内容全等跨项目复用 / 冲突消歧。
func TestResolveQuotaCodeUnification(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	// ① 空库：原码直用。
	got, err := s.ResolveQuotaCode("WP02", "施工便道", "m", "3m宽", "项目导入：A.xlsx")
	if err != nil || got != "WP02" {
		t.Fatalf("空库应直用原码，得到 %s（err=%v）", got, err)
	}
	if _, err := s.SaveQuota(Quota{Code: "WP02", OrigCode: "WP02", Title: "施工便道", Unit: "m", Note: "3m宽", Source: "项目导入：A.xlsx"}, nil); err != nil {
		t.Fatalf("建定额失败: %v", err)
	}

	// ② 同源重导入：复用。
	got, err = s.ResolveQuotaCode("WP02", "施工便道", "m", "3m宽", "项目导入：A.xlsx")
	if err != nil || got != "WP02" {
		t.Fatalf("同源重导入应复用，得到 %s（err=%v）", got, err)
	}

	// ③ 他项目内容全等：复用（真统一）。
	got, err = s.ResolveQuotaCode("WP02", "施工便道", "m", "3m宽", "项目导入：B.xlsx")
	if err != nil || got != "WP02" {
		t.Fatalf("内容全等应跨项目复用，得到 %s（err=%v）", got, err)
	}

	// ④ 他项目内容不同：消歧 WP02-2，且 orig_code 保留原码。
	got, err = s.ResolveQuotaCode("WP02", "施工便道", "m", "5m宽双车道", "项目导入：C.xlsx")
	if err != nil || got != "WP02-2" {
		t.Fatalf("内容不同应消歧 WP02-2，得到 %s（err=%v）", got, err)
	}
	if _, err := s.SaveQuota(Quota{Code: got, OrigCode: "WP02", Title: "施工便道", Unit: "m", Note: "5m宽双车道", Source: "项目导入：C.xlsx"}, nil); err != nil {
		t.Fatalf("消歧定额落库失败: %v", err)
	}
	q, err := s.GetQuota("WP02-2")
	if err != nil || q.OrigCode != "WP02" {
		t.Fatalf("消歧定额应保留 orig_code=WP02，得到 %q（err=%v）", q.OrigCode, err)
	}
	// 第三次冲突：WP02-3。
	got, err = s.ResolveQuotaCode("WP02", "施工便道", "m", "6m宽", "项目导入：D.xlsx")
	if err != nil || got != "WP02-3" {
		t.Fatalf("第三个冲突应 WP02-3，得到 %s（err=%v）", got, err)
	}
}

// TestDeleteBillProject 整体删除：项目+清单+独占定额；他项目引用的共享定额
// 与资源保留（编码统一复用的产物不动）。
func TestDeleteBillProject(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	idA, _ := s.UpsertBillProject("项目A", "a.xlsx", "", "", "", "", nil, false, 0)
	idB, _ := s.UpsertBillProject("项目B", "b.xlsx", "", "", "", "", nil, false, 0)

	// A 的独占定额 QA1；A、B 共享的定额 QSH（内容全等复用场景：B 的清单引用它）。
	if _, err := s.SaveQuota(Quota{Code: "QA1", OrigCode: "QA1", Title: "A独有", Unit: "项", Source: "项目导入：a.xlsx"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveQuota(Quota{Code: "QSH", OrigCode: "QSH", Title: "共享便道", Unit: "m", Source: "项目导入：a.xlsx"}, nil); err != nil {
		t.Fatal(err)
	}
	for _, it := range []BillItem{
		{ProjectID: idA, Code: "C1", Title: "A独有项", QuotaCode: "QA1"},
		{ProjectID: idA, Code: "C2", Title: "A共享项", QuotaCode: "QSH"},
		{ProjectID: idB, Code: "C1", Title: "B共享项", QuotaCode: "QSH"},
	} {
		if _, err := s.SaveBillItem(it); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := s.DeleteBillProject(idA)
	if err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	if removed != 1 {
		t.Errorf("应删 1 条独占定额（QA1），得到 %d", removed)
	}
	if _, err := s.GetQuota("QA1"); err == nil {
		t.Error("独占定额 QA1 应被删除")
	}
	if _, err := s.GetQuota("QSH"); err != nil {
		t.Errorf("共享定额 QSH 应保留（B 仍引用）: %v", err)
	}
	projects, _ := s.BillProjects()
	if len(projects) != 1 || projects[0].ID != idB {
		t.Errorf("应只剩项目 B: %+v", projects)
	}
	items, _ := s.BillItems(idB)
	if len(items) != 1 || items[0].QuotaCode != "QSH" {
		t.Errorf("项目 B 清单应完好: %+v", items)
	}
}
