package workcost

// 三期 3.4 资源价批量维护测试：Excel 解析（列头同义/容忍千分位）/匹配宁缺勿
// 误（零命中/歧义跳过）/批量应用（整批事务+历史+现行价）。

import (
	"bytes"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/xuri/excelize/v2"
)

func bulkTestStore(t *testing.T) *Store {
	t.Helper()
	// 真实库（GetDatabase 自带迁移链，临时目录隔离；LIFO 关闭防单例泄漏）
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	s := Open(gdb)
	// 种子资源
	for _, r := range []Resource{
		{Code: "CL0001", Kind: "材料", Title: "P.O 42.5 水泥", Spec: "散装", Unit: "kg", CurrentPrice: 0.42},
		{Code: "JX0001", Kind: "机械", Title: "挖掘机 220", Unit: "台班", CurrentPrice: 1200},
		{Code: "RG0001", Kind: "人工", Title: "普工", Unit: "工日", CurrentPrice: 150},
	} {
		if _, err := s.SaveResource(r); err != nil {
			t.Fatal(err)
		}
	}
	// 同名双行（歧义样本）
	if _, err := s.SaveResource(Resource{Code: "CL0002", Kind: "材料", Title: "P.O 42.5 水泥", Spec: "袋装", Unit: "kg", CurrentPrice: 0.5}); err != nil {
		t.Fatal(err)
	}
	return s
}

func bulkXlsx(t *testing.T, header []string, dataRows [][]any) []byte {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for i, h := range header {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			t.Fatal(err)
		}
	}
	for ri, row := range dataRows {
		for ci, v := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, ri+2)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParseBulkPriceXlsx_HeaderSynonyms(t *testing.T) {
	raw := bulkXlsx(t,
		[]string{"资源名称", "规格型号", "计量单位", "调整价", "信息价期数"},
		[][]any{{"P.O 42.5 水泥", "散装", "kg", "0.48", "2026年3期"}},
	)
	rows, err := ParseBulkPriceXlsx(raw)
	if err != nil {
		t.Fatalf("解析: %v", err)
	}
	if len(rows) != 1 || rows[0].Title != "P.O 42.5 水泥" || rows[0].Unit != "kg" || rows[0].Price != 0.48 || rows[0].Period != "2026年3期" {
		t.Fatalf("解析不符: %+v", rows)
	}
}

func TestParseBulkPriceXlsx_ToleratesCommaAndMissingSheet(t *testing.T) {
	// 千分位/元后缀容忍
	raw := bulkXlsx(t, []string{"名称", "单位", "新价"}, [][]any{{"普工", "工日", "1,250元"}})
	rows, err := ParseBulkPriceXlsx(raw)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Price != 1250 {
		t.Fatalf("千分位解析: %v", rows[0].Price)
	}
	// 无名称列：报错不猜
	bad := bulkXlsx(t, []string{"名字", "单位", "新价"}, [][]any{{"x", "y", 1}})
	if _, err := ParseBulkPriceXlsx(bad); err == nil {
		t.Fatal("缺名称列应报错")
	}
}

func TestMatchBulkPrices_NoFalsePositive(t *testing.T) {
	s := bulkTestStore(t)
	rows := []BulkPriceRow{
		{Row: 2, Title: "P.O 42.5 水泥", Spec: "散装", Unit: "kg", Price: 0.48},   // 唯一命中
		{Row: 3, Title: "P.O 42.5 水泥", Unit: "kg", Price: 0.48},                // 无规格→同名双行歧义
		{Row: 4, Title: "不存在的材料", Unit: "kg", Price: 1},                      // 零命中
		{Row: 5, Title: "挖掘机 220", Unit: "台班", Price: -5},                   // 价格非法
		{Row: 6, Title: "普工", Unit: "工日", Price: 180, Period: "2026年3期"}, // 唯一命中
	}
	matches, skips, err := s.MatchBulkPrices(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("应命中 2 条: %+v", matches)
	}
	if len(skips) != 3 {
		t.Fatalf("应跳过 3 条: %+v", skips)
	}
	for _, sk := range skips {
		if sk.Reason == "" {
			t.Fatalf("跳过原因须人可读: %+v", sk)
		}
	}
}

func TestApplyBulkPrices_TransactionAndHistory(t *testing.T) {
	s := bulkTestStore(t)
	rows := []BulkPriceRow{
		{Row: 2, Title: "P.O 42.5 水泥", Spec: "散装", Unit: "kg", Price: 0.48, Period: "2026年3期"},
		{Row: 3, Title: "普工", Unit: "工日", Price: 180},
		{Row: 4, Title: "不存在的材料", Price: 1}, // 跳过不影响整批
	}
	res, err := s.ApplyBulkPrices(rows)
	if err != nil {
		t.Fatalf("应用: %v", err)
	}
	if res.Updated != 2 || len(res.Skipped) != 1 || res.Total != 3 {
		t.Fatalf("回执不符: %+v", res)
	}
	// 现行价推进
	r, err := s.GetResourceByCode("CL0001")
	if err != nil {
		t.Fatal(err)
	}
	if r.CurrentPrice != 0.48 {
		t.Fatalf("现行价未推进: %v", r.CurrentPrice)
	}
	// 调价历史落档
	ps, err := s.ResourcePrices(r.ID)
	if err != nil || len(ps) != 1 {
		t.Fatalf("历史: %+v err=%v", ps, err)
	}
	if ps[0].Price != 0.48 || ps[0].Period != "2026年3期" || ps[0].PriceType != "bulk_import" {
		t.Fatalf("历史行不符: %+v", ps[0])
	}
}
