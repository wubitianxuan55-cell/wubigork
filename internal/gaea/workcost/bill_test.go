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
