package app

// v4.6 询价飞轮反向接线（审计 §C ②）：OCR/图片报价单确认导入后，各行自动
// 成为询价库数据点（source=OCR报价）；重复导入同一报价单幂等更新不新增行；
// 不传 inquirySource 时零行为变化（不写询价库）。

import (
	"testing"

	gconfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costinquiry"
	"github.com/gaea/gaea/internal/gaea/db"
)

func TestCostImportApplyWritesInquiryPoints(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	gdb := db.GetDatabase(gconfig.MemoryUserDir())
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(gconfig.MemoryUserDir()) })
	SetCostStoreForTest(cost.Open(gdb))
	t.Cleanup(ResetCostStoreForTest)
	inq := costinquiry.Open(gdb)
	SetCostInquiryStoreForTest(inq)
	t.Cleanup(ResetCostInquiryStoreForTest)

	a := &App{officeState: &officeState{core: &core{}}}
	rows := []CostEntry{
		{CostSummary: CostSummary{Name: "rebar", Title: "热轧光圆钢筋", Category: "材料",
			Unit: "t", Price: 3750, Spec: "HPB300 Φ12", Source: "供应商报价单.pdf"}},
		{CostSummary: CostSummary{Name: "cement", Title: "水泥", Category: "材料", Unit: "t", Price: 480}},
	}

	n, err := a.GaeaCostImportApply(rows, "OCR报价")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if n != 2 {
		t.Fatalf("导入条数 = %d, want 2", n)
	}
	recs := a.GaeaCostInquiryList("", 10)
	if len(recs) != 2 {
		t.Fatalf("询价数据点 = %d, want 2（报价单自动入询价库）", len(recs))
	}
	byTitle := map[string]costinquiry.Record{}
	for _, r := range recs {
		byTitle[r.Title] = r
	}
	if byTitle["热轧光圆钢筋"].Source != "OCR报价" ||
		byTitle["热轧光圆钢筋"].Supplier != "供应商报价单.pdf" {
		t.Fatalf("数据点来源/供应商未标注: %+v", byTitle["热轧光圆钢筋"])
	}

	// 同一报价单再次导入：幂等更新（价格刷新）不新增行
	rows[0].Price = 3780
	if _, err := a.GaeaCostImportApply(rows, "OCR报价"); err != nil {
		t.Fatalf("apply2: %v", err)
	}
	recs = a.GaeaCostInquiryList("", 10)
	if len(recs) != 2 {
		t.Fatalf("重复导入后数据点 = %d, want 2（幂等去重）", len(recs))
	}
	for _, r := range recs {
		if r.Title == "热轧光圆钢筋" && r.Price != 3780 {
			t.Fatalf("重复导入应刷新价格, got %v", r.Price)
		}
	}

	// 不传 inquirySource：只写成本库，不写询价库（旧行为零变化）
	before := len(a.GaeaCostInquiryList("", 10))
	if _, err := a.GaeaCostImportApply(rows); err != nil {
		t.Fatalf("apply(no inquiry): %v", err)
	}
	if got := len(a.GaeaCostInquiryList("", 10)); got != before {
		t.Fatalf("不传 inquirySource 时询价库不应变化: %d → %d", before, got)
	}
}
