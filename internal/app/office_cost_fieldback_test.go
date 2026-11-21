package app

// office_cost_fieldback_test.go — 2026-11-21 造价门实测复盘（沿江1公里成本测算）
// 修复批回归钉：现算汇总口 / 规格档错配检 / 数据指纹 / xlsx 指纹与版面 /
// genui 双形态 / resolveIn MSYS 盘符。

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

// TestCostProjectSummary 现算汇总口（4.1）：定额行现算+手填价整价归材料+取费链。
func TestCostProjectSummary(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	labor, _ := s.SaveResource(workcost.Resource{Kind: "人工", Title: "技工", Unit: "工日", BasePrice: 150, CurrentPrice: 180})
	if _, err := s.SaveQuota(workcost.Quota{Code: "QK-S", Title: "浇筑定额", Unit: "m³",
		Items: []workcost.QuotaItem{{ResourceCode: labor.Code, Quantity: 0.1}}}, nil); err != nil {
		t.Fatalf("保存定额失败: %v", err)
	}
	pid, err := s.UpsertBillProject("汇总项目", "", "", "", "", "",
		&workcost.RateSet{ManagementRate: 0.05, RegulatoryRate: 0, ProfitRate: 0.03, TaxRate: 0.09}, false, 2000)
	if err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	for _, it := range []workcost.BillItem{
		{ProjectID: pid, Code: "B1", Title: "定额行", Unit: "m³", Quantity: 10, QuotaCode: "QK-S", QuotaCodes: []string{"QK-S"}},
		{ProjectID: pid, Code: "B2", Title: "手填行", Unit: "项", Quantity: 2, PriceOverride: 100, QuotaCodes: []string{}},
	} {
		if _, err := s.SaveBillItem(it); err != nil {
			t.Fatalf("播种行失败: %v", err)
		}
	}
	out, err := costProjectTool{a: a}.Execute(context.Background(), ormJSON(t, map[string]any{"action": "summary", "project_id": pid}))
	if err != nil {
		t.Fatalf("summary 失败: %v", err)
	}
	// 定额行：0.1 工日×180=18/单位 → 量 10 → 合价 180、人=180。
	// 手填行：100×2=200，整价归材料。
	// direct=380；管=19；利=(380+19)×0.03=11.97；规=0；税前=410.97；
	// 税=36.99；含税=447.96（占控制价 2000 的 22.4%）。
	for _, want := range []string{
		"综合单价=18.00 合价=180.00（人=180.00",
		"手填价=100.00 综合单价=100.00 合价=200.00（人=0.00 材=200.00",
		"人材机直接费=380.00",
		"管理费=19.00｜利润=11.97｜规费=0.00｜税前=410.97｜增值税=36.99",
		"含税合计=447.96（占控制价 22.4%）",
		"整价归材料",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary 缺 %q：\n%s", want, out)
		}
	}
}

// TestCostAuditSpecGrade 规格档错配检（4.2/六：C60 充当 C20 的确定性防线，INFO 级）。
func TestCostAuditSpecGrade(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	mat, _ := s.SaveResource(workcost.Resource{Kind: "材料", Title: "商品混凝土 C60", Unit: "m³", BasePrice: 438, CurrentPrice: 438})
	if _, err := s.SaveQuota(workcost.Quota{Code: "QK-C60", Title: "C60 混凝土浇筑", Unit: "m³",
		Items: []workcost.QuotaItem{{ResourceCode: mat.Code, Quantity: 1}}}, nil); err != nil {
		t.Fatalf("保存定额失败: %v", err)
	}
	pid, _ := s.UpsertBillProject("档位项目", "", "", "", "", "",
		&workcost.RateSet{ManagementRate: 0, RegulatoryRate: 0, ProfitRate: 0, TaxRate: 0.09}, false, 0)
	for _, it := range []workcost.BillItem{
		{ProjectID: pid, Code: "G1", Title: "C20 混凝土浇筑", Unit: "m³", Quantity: 5, QuotaCode: "QK-C60", QuotaCodes: []string{"QK-C60"}},
		{ProjectID: pid, Code: "G2", Title: "C60 混凝土浇筑", Unit: "m³", Quantity: 5, QuotaCode: "QK-C60", QuotaCodes: []string{"QK-C60"}},
	} {
		if _, err := s.SaveBillItem(it); err != nil {
			t.Fatalf("播种行失败: %v", err)
		}
	}
	out, err := costAuditTool{a: a}.Execute(context.Background(), ormJSON(t, map[string]any{"project_id": pid}))
	if err != nil {
		t.Fatalf("cost_audit 失败: %v", err)
	}
	if !strings.Contains(out, "spec_grade_mismatch") || !strings.Contains(out, "C20") || !strings.Contains(out, "C60") {
		t.Fatalf("G1 行 C20×C60 错配应报 INFO：\n%s", out)
	}
	// G2 同档不报：spec_grade_mismatch 只出现一次。
	if n := strings.Count(out, "spec_grade_mismatch"); n != 1 {
		t.Fatalf("错配应恰 1 条（G2 同档不报），got %d：\n%s", n, out)
	}
	if strings.Contains(out, "HardCount=1") {
		t.Fatalf("档位错配是 INFO 不进 HardCount：\n%s", out)
	}
	if !strings.Contains(out, "数据指纹：清单 2 行") {
		t.Fatalf("回执应带头数据指纹（5.1）：\n%s", out)
	}
}

// TestXlsxAuditFingerprintAndLayout xlsx 指纹+版面确定性检查（5.1/5.2）。
func TestXlsxAuditFingerprintAndLayout(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "测算.xlsx")
	f := excelize.NewFile()
	// 12 列长表 + 默认纵向（不设 FitToWidth）——实测「整册压成单列」形态。
	for c := 1; c <= 12; c++ {
		cell, _ := excelize.CoordinatesToCellName(c, 1)
		_ = f.SetCellValue("Sheet1", cell, c)
	}
	if err := f.SaveAs(p); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	out, err := xlsxAuditTool{a: &App{}}.Execute(context.Background(), ormJSON(t, map[string]any{"path": p}))
	if err != nil {
		t.Fatalf("xlsx_audit 失败: %v", err)
	}
	for _, want := range []string{"文件指纹：size=", "sha16=", "版面检查", "12 列", "⚠ 长表纵向且未按宽适应"} {
		if !strings.Contains(out, want) {
			t.Errorf("回执缺 %q：\n%s", want, out)
		}
	}
	// 修复面：横向+FitToWidth=1 → 告警消失、指纹仍在。
	f2 := excelize.NewFile()
	for c := 1; c <= 12; c++ {
		cell, _ := excelize.CoordinatesToCellName(c, 1)
		_ = f2.SetCellValue("Sheet1", cell, c)
	}
	_ = f2.SetPageLayout("Sheet1", &excelize.PageLayoutOptions{
		Orientation: strPtr("landscape"), FitToWidth: intPtr(1), FitToHeight: intPtr(0),
	})
	if err := f2.SaveAs(p); err != nil {
		t.Fatal(err)
	}
	_ = f2.Close()
	out2, err := xlsxAuditTool{a: &App{}}.Execute(context.Background(), ormJSON(t, map[string]any{"path": p}))
	if err != nil {
		t.Fatalf("xlsx_audit2 失败: %v", err)
	}
	if strings.Contains(out2, "⚠") {
		t.Fatalf("横向+FitToWidth 不应告警：\n%s", out2)
	}
	if !strings.Contains(out2, "版面无告警") || !strings.Contains(out2, "landscape") {
		t.Fatalf("修复面应报无告警+横向：\n%s", out2)
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
