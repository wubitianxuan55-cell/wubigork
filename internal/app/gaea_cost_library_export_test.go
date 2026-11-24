package app

// gaea_cost_library_export_test.go — 造价库列表导出绑定（2026-11-25 造价数据库
// 推倒重建）与删除护栏：四类导出的表结构、筛选口径、拒绝覆盖；定额被引用拒删。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// exportRows 打开导出的工作簿，返回各表的数据行（不含表头）。
func exportRows(t *testing.T, path string) map[string][][]string {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("打开导出文件失败: %v", err)
	}
	defer func() { _ = f.Close() }()
	out := map[string][][]string{}
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			t.Fatalf("读表 %s 失败: %v", sheet, err)
		}
		if len(rows) > 0 {
			out[sheet] = rows[1:]
		} else {
			out[sheet] = nil
		}
	}
	return out
}

func TestCostLibraryExportBillAndGuard(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	res, pid, _ := clSeed(t, s)
	_ = res
	// 再建一个项目、一条别的分部的清单行，验证筛选。
	pid2, err := s.UpsertBillProject("测算Q", "", "", "", "", "", nil, false, 0)
	if err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	if _, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid2, Code: "WP77", Title: "围挡", Unit: "m", Quantity: 50, Division: "临建", PriceOverride: 30}); err != nil {
		t.Fatalf("建清单行失败: %v", err)
	}

	out := filepath.Join(t.TempDir(), "清单.xlsx")
	written, err := a.GaeaCostLibraryExport("bill", 0, "", "", out)
	if err != nil {
		t.Fatalf("导出清单失败: %v", err)
	}
	if written != filepath.ToSlash(out) {
		t.Fatalf("应返回写出路径，得到 %q", written)
	}
	sheets := exportRows(t, out)
	if len(sheets["分部分项清单"]) != 2 {
		t.Fatalf("全部项目应导出 2 行，得到 %d", len(sheets["分部分项清单"]))
	}
	// 关键词筛选（忽略大小写）+ 分部筛选。
	out2 := filepath.Join(t.TempDir(), "清单-筛选")
	if _, err := a.GaeaCostLibraryExport("bill", pid2, "WP77", "临建", out2); err != nil {
		t.Fatalf("筛选导出失败: %v", err)
	}
	if rows := exportRows(t, out2+".xlsx")["分部分项清单"]; len(rows) != 1 || rows[0][1] != "WP77" || rows[0][9] != "30" {
		t.Fatalf("筛选后应只剩 WP77 且手填单价 30，得到 %v", rows)
	}
	if _, err := a.GaeaCostLibraryExport("bill", pid, "", "", out2); err == nil || !strings.Contains(err.Error(), "已存在") {
		t.Fatalf("目标文件已存在应拒绝覆盖，得到 err=%v", err)
	}
	if _, err := a.GaeaCostLibraryExport("nope", 0, "", "", out); err == nil {
		t.Fatal("未知导出类型应被拒绝")
	}

	// 删除护栏：定额 A 被清单行引用 → 绑定拒绝；未被引用的零含量定额可删。
	if err := a.GaeaWorkcostQuotaDelete("A"); err == nil || !strings.Contains(err.Error(), "清单项引用") {
		t.Fatalf("被引用的定额应拒删，得到 err=%v", err)
	}
	if _, err := s.GetQuota("A"); err != nil {
		t.Fatal("被拒绝的删除不应生效")
	}
	// 工料机引用面（回链）。
	refs := a.GaeaWorkcostResourceUsage(res.Code)
	if len(refs) != 1 || refs[0].Code != "A" {
		t.Fatalf("工料机应被定额 A 引用，得到 %+v", refs)
	}
	if refs := a.GaeaWorkcostResourceUsage("MISSING"); refs == nil || len(refs) != 0 {
		t.Fatalf("无引用应返回空切片（非 nil），得到 %#v", refs)
	}
}

func TestCostLibraryExportQuotaResourceEntry(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	res, _, _ := clSeed(t, s)

	// 定额：主表 + 含量明细附表。
	quotaOut := filepath.Join(t.TempDir(), "定额")
	if _, err := a.GaeaCostLibraryExport("quota", 0, "", "", quotaOut); err != nil {
		t.Fatalf("导出定额失败: %v", err)
	}
	sheets := exportRows(t, quotaOut+".xlsx")
	main := sheets["消耗定额"]
	if len(main) != 2 {
		t.Fatalf("应导出 2 条定额，得到 %d", len(main))
	}
	// 行序：专业/章节/编码——A 在前。
	if main[0][0] != "A" || main[0][7] != "1" || main[0][8] != "1" {
		t.Fatalf("定额 A 应有 1 行含量、被 1 条清单引用，得到 %v", main[0])
	}
	if !strings.HasPrefix(main[0][9], "469") { // 1.02 × 460 = 469.2
		t.Fatalf("定额 A 现算综合单价应为 469.2，得到 %q", main[0][9])
	}
	detail := sheets["含量明细"]
	if len(detail) != 1 || detail[0][3] != res.Code || detail[0][7] != "1.02" {
		t.Fatalf("含量明细应有 1 行指向工料机 %s、含量 1.02，得到 %v", res.Code, detail)
	}

	// 工料机：含被定额引用次数。
	resOut := filepath.Join(t.TempDir(), "工料机")
	if _, err := a.GaeaCostLibraryExport("resource", 0, "", "", resOut); err != nil {
		t.Fatalf("导出工料机失败: %v", err)
	}
	resRows := exportRows(t, resOut+".xlsx")["工料机"]
	if len(resRows) != 1 || resRows[0][0] != res.Code || resRows[0][18] != "1" || resRows[0][5] != "460" {
		t.Fatalf("工料机导出应为 1 行、现行价 460、被引用 1 次，得到 %v", resRows)
	}
	// 按类别筛选：库里只有一条材料，筛「人工」应为空表（只有表头）。
	laborOut := filepath.Join(t.TempDir(), "工料机-人工")
	if _, err := a.GaeaCostLibraryExport("resource", 0, "", "人工", laborOut); err != nil {
		t.Fatalf("按类别导出失败: %v", err)
	}
	if rows := exportRows(t, laborOut+".xlsx")["工料机"]; len(rows) != 0 {
		t.Fatalf("按「人工」筛选应无数据行，得到 %v", rows)
	}

	// 资料条目。
	if err := a.hubCostStore().Save(cost.Entry{Name: "sand-1", Title: "中砂", Category: "材料", Unit: "m³", Price: 110, Source: "市场询价"}); err != nil {
		t.Fatalf("写资料条目失败: %v", err)
	}
	entryOut := filepath.Join(t.TempDir(), "资料条目")
	if _, err := a.GaeaCostLibraryExport("entry", 0, "中砂", "", entryOut); err != nil {
		t.Fatalf("导出资料条目失败: %v", err)
	}
	entryRows := exportRows(t, entryOut+".xlsx")["资料条目"]
	if len(entryRows) != 1 || entryRows[0][1] != "中砂" || entryRows[0][5] != "110" {
		t.Fatalf("资料条目应导出 1 行「中砂」单价 110，得到 %v", entryRows)
	}
	if _, err := os.Stat(entryOut + ".xlsx"); err != nil {
		t.Fatalf("导出文件应存在: %v", err)
	}
}
