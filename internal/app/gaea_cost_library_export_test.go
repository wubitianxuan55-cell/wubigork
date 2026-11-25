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
	written, err := a.GaeaCostLibraryExport("bill", workcost.LibraryFilter{}, out)
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
	// 关键词筛选（忽略大小写）+ 分部筛选。列序：项目, 专业, 编码, 自有码, …, 手填单价（第 11 列）。
	out2 := filepath.Join(t.TempDir(), "清单-筛选")
	if _, err := a.GaeaCostLibraryExport("bill", workcost.LibraryFilter{ProjectID: pid2, Keyword: "WP77", Division: "临建"}, out2); err != nil {
		t.Fatalf("筛选导出失败: %v", err)
	}
	if rows := exportRows(t, out2+".xlsx")["分部分项清单"]; len(rows) != 1 || rows[0][2] != "WP77" || rows[0][11] != "30" {
		t.Fatalf("筛选后应只剩 WP77 且手填单价 30，得到 %v", rows)
	}
	// 「未标注专业」哨兵：清单行都没标专业（跟随项目默认专业，项目也没标）→ 两行都在。
	noneOut := filepath.Join(t.TempDir(), "清单-未标注")
	if _, err := a.GaeaCostLibraryExport("bill", workcost.LibraryFilter{Specialty: workcost.FilterNone}, noneOut); err != nil {
		t.Fatalf("未标注筛选导出失败: %v", err)
	}
	if rows := exportRows(t, noneOut+".xlsx")["分部分项清单"]; len(rows) != 2 {
		t.Fatalf("筛「未标注专业」应仍是 2 行，得到 %d", len(rows))
	}
	specOut := filepath.Join(t.TempDir(), "清单-市政")
	if _, err := a.GaeaCostLibraryExport("bill", workcost.LibraryFilter{Specialty: "市政"}, specOut); err != nil {
		t.Fatalf("导出空结果不应报错: %v", err)
	}
	if rows := exportRows(t, specOut+".xlsx")["分部分项清单"]; len(rows) != 0 {
		t.Fatalf("筛「市政」应无数据行，得到 %v", rows)
	}
	if _, err := a.GaeaCostLibraryExport("bill", workcost.LibraryFilter{ProjectID: pid}, out2); err == nil || !strings.Contains(err.Error(), "已存在") {
		t.Fatalf("目标文件已存在应拒绝覆盖，得到 err=%v", err)
	}
	if _, err := a.GaeaCostLibraryExport("nope", workcost.LibraryFilter{}, out); err == nil {
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

	// 定额：主表 + 含量明细附表。主表列序：编码, 自有码, 原码, 名称, 单位, 专业, 章节,
	// 借鉴来源, 来源, 含量行数, 被清单引用, 现算综合单价。
	quotaOut := filepath.Join(t.TempDir(), "定额")
	if _, err := a.GaeaCostLibraryExport("quota", workcost.LibraryFilter{}, quotaOut); err != nil {
		t.Fatalf("导出定额失败: %v", err)
	}
	sheets := exportRows(t, quotaOut+".xlsx")
	main := sheets["消耗定额"]
	if len(main) != 2 {
		t.Fatalf("应导出 2 条定额，得到 %d", len(main))
	}
	// 行序：专业/章节/编码——A 在前。
	if main[0][0] != "A" || main[0][9] != "1" || main[0][10] != "1" {
		t.Fatalf("定额 A 应有 1 行含量、被 1 条清单引用，得到 %v", main[0])
	}
	if !strings.HasPrefix(main[0][11], "469") { // 1.02 × 460 = 469.2
		t.Fatalf("定额 A 现算综合单价应为 469.2，得到 %q", main[0][10])
	}
	detail := sheets["含量明细"]
	if len(detail) != 1 || detail[0][4] != res.Code || detail[0][8] != "1.02" {
		t.Fatalf("含量明细应有 1 行指向工料机 %s、含量 1.02，得到 %v", res.Code, detail)
	}

	// 工料机：含被定额引用次数。
	resOut := filepath.Join(t.TempDir(), "工料机")
	if _, err := a.GaeaCostLibraryExport("resource", workcost.LibraryFilter{}, resOut); err != nil {
		t.Fatalf("导出工料机失败: %v", err)
	}
	resRows := exportRows(t, resOut+".xlsx")["工料机"]
	if len(resRows) != 1 || resRows[0][0] != res.Code || resRows[0][19] != "1" || resRows[0][6] != "460" {
		t.Fatalf("工料机导出应为 1 行、现行价 460、被引用 1 次，得到 %v", resRows)
	}
	// 按类别筛选：库里只有一条材料，筛「人工」应为空表（只有表头）。
	laborOut := filepath.Join(t.TempDir(), "工料机-人工")
	if _, err := a.GaeaCostLibraryExport("resource", workcost.LibraryFilter{Kind: "人工"}, laborOut); err != nil {
		t.Fatalf("按类别导出失败: %v", err)
	}
	if rows := exportRows(t, laborOut+".xlsx")["工料机"]; len(rows) != 0 {
		t.Fatalf("按「人工」筛选应无数据行，得到 %v", rows)
	}
	// 地区筛选：按地区分行后，选一个地区只导那个地区的行（未标注地区是另一桶）。
	if err := s.DictAdd(workcost.DictItem{Kind: workcost.DictRegion, Name: "成都市区"}); err != nil {
		t.Fatal(err)
	}
	cq, err := s.SaveResource(workcost.Resource{Kind: res.Kind, Title: res.Title, Spec: res.Spec, Unit: res.Unit,
		Region: "成都市区", BasePrice: 480, CurrentPrice: 480})
	if err != nil {
		t.Fatalf("建成都地区的同身份工料机失败: %v", err)
	}
	cdOut := filepath.Join(t.TempDir(), "工料机-成都")
	if _, err := a.GaeaCostLibraryExport("resource", workcost.LibraryFilter{Region: "成都市区"}, cdOut); err != nil {
		t.Fatalf("按地区导出失败: %v", err)
	}
	if rows := exportRows(t, cdOut+".xlsx")["工料机"]; len(rows) != 1 || rows[0][0] != cq.Code {
		t.Fatalf("按「成都市区」应只导出 1 行，得到 %v", rows)
	}
	// 「类别待整理」：存量里类别不在四类之内的行（写入面已拒收，只能直插模拟），导出按同一哨兵取。
	if _, err := s.DB().Exec(`INSERT INTO gf_resources(code, kind, title, unit, tags) VALUES('NOTE-1', '一段说明', '错记成工料机的说明行', '项', '[]')`); err != nil {
		t.Fatalf("直插类别不合规矩的工料机失败: %v", err)
	}
	outsideOut := filepath.Join(t.TempDir(), "工料机-类别待整理")
	if _, err := a.GaeaCostLibraryExport("resource", workcost.LibraryFilter{Kind: workcost.FilterOutside}, outsideOut); err != nil {
		t.Fatalf("按「类别待整理」导出失败: %v", err)
	}
	if rows := exportRows(t, outsideOut+".xlsx")["工料机"]; len(rows) != 1 || rows[0][0] != "NOTE-1" || rows[0][2] != "一段说明" {
		t.Fatalf("按「类别待整理」应只导出 NOTE-1，得到 %v", rows)
	}

	// 资料条目。
	if err := a.hubCostStore().Save(cost.Entry{Name: "sand-1", Title: "中砂", Category: "材料", Unit: "m³", Price: 110, Source: "市场询价"}); err != nil {
		t.Fatalf("写资料条目失败: %v", err)
	}
	entryOut := filepath.Join(t.TempDir(), "资料条目")
	if _, err := a.GaeaCostLibraryExport("entry", workcost.LibraryFilter{Keyword: "中砂"}, entryOut); err != nil {
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

// TestCostLibraryExportTemplate 测算模版出口（kind=template）：六表结构；工料机价格表
// 按口径从库预填（带自有码），类别不在四类之内的存量行不预填；「未标注」哨兵对
// 模版当没给；费率预填库内最近项目的取费口径并在封面注明出处。
func TestCostLibraryExportTemplate(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	res, _, _ := clSeed(t, s)
	for _, d := range []struct{ kind, name string }{{workcost.DictRegion, "重庆中心城区"}, {workcost.DictCategory, "材料/钢材"}} {
		if err := s.DictAdd(workcost.DictItem{Kind: d.kind, Name: d.name}); err != nil {
			t.Fatalf("加字典失败: %v", err)
		}
	}
	labor, err := s.SaveResource(workcost.Resource{Kind: workcost.KindLabor, Title: "普工", Unit: "工日", CurrentPrice: 300, Region: "重庆中心城区"})
	if err != nil {
		t.Fatalf("建重庆人工失败: %v", err)
	}
	steel, err := s.SaveResource(workcost.Resource{Kind: workcost.KindMaterial, Title: "螺纹钢", Spec: "HRB400", Unit: "t", CurrentPrice: 3800,
		Region: "重庆中心城区", CategoryPath: "材料/钢材"})
	if err != nil {
		t.Fatalf("建重庆钢材失败: %v", err)
	}
	// 类别不在四类之内的存量行（写入面已拒收，直插模拟）：不是工料机，不进模版。
	if _, err := s.DB().Exec(`INSERT INTO gf_resources(code, kind, title, unit, tags) VALUES('NOTE-1', '一段说明', '错记成工料机的说明行', '项', '[]')`); err != nil {
		t.Fatalf("直插类别不合规矩的工料机失败: %v", err)
	}
	// 最新项目带费率：模版的费率口径取自它。
	rates := workcost.RateSet{ManagementRate: 0.05, RegulatoryRate: 0.02, ProfitRate: 0.04, TaxRate: 0.03}
	if _, err := s.UpsertBillProject("新项目Q", "", "", "", "", "", &rates, false, 0); err != nil {
		t.Fatalf("建带费率的项目失败: %v", err)
	}

	export := func(f workcost.LibraryFilter) map[string][][]string {
		t.Helper()
		out := filepath.Join(t.TempDir(), "模版")
		if _, err := a.GaeaCostLibraryExport("template", f, out); err != nil {
			t.Fatalf("导出测算模版失败: %v", err)
		}
		return exportRows(t, out+".xlsx")
	}
	resourceRows := func(sheets map[string][][]string) [][]string {
		rows := sheets["工料机价格"]
		for i, r := range rows { // 说明两行 + 表头，之后是预填行
			if len(r) > 0 && r[0] == "编码" {
				return rows[i+1:]
			}
		}
		t.Fatalf("工料机价格表未找到表头：%v", rows)
		return nil
	}

	// 不给口径：库里三条合规工料机全部预填，脏行不进；每行带库内自有码。
	all := resourceRows(export(workcost.LibraryFilter{}))
	if len(all) != 3 {
		t.Fatalf("应预填 3 条工料机，得到 %d 行：%v", len(all), all)
	}
	codes := map[string]string{}
	for _, r := range all {
		if len(r) < 7 {
			t.Fatalf("预填行列数不足：%v", r)
		}
		codes[r[0]] = r[1]
		if r[1] == "" || !strings.HasPrefix(r[1], "R") {
			t.Errorf("预填行 %s 应带 R 族自有码，得到 %q", r[0], r[1])
		}
	}
	if codes[res.Code] == "" || codes[labor.Code] == "" || codes[steel.Code] == "" {
		t.Errorf("三条合规工料机都应预填，得到 %v", codes)
	}
	for _, r := range all {
		if r[0] == "NOTE-1" {
			t.Error("类别不在四类之内的行不应预填进模版")
		}
	}

	// 按地区：只预填该地区的行；按分类：选上级分类连带下级。
	if got := resourceRows(export(workcost.LibraryFilter{Region: "重庆中心城区"})); len(got) != 2 {
		t.Errorf("按地区应预填 2 条，得到 %d 行：%v", len(got), got)
	}
	if got := resourceRows(export(workcost.LibraryFilter{Category: "材料/钢材"})); len(got) != 1 || got[0][0] != steel.Code {
		t.Errorf("按分类应只预填钢材，得到 %v", got)
	}
	// 「未标注」哨兵对模版=没给口径（模版没有「只要未标注」这种用法）。
	if got := resourceRows(export(workcost.LibraryFilter{Region: workcost.FilterNone, Kind: workcost.FilterOutside})); len(got) != 3 {
		t.Errorf("哨兵应 当没给，预填 3 条，得到 %d 行", len(got))
	}

	// 封面写明口径与费率出处；费用汇总费率=最近项目的口径。
	sheets := export(workcost.LibraryFilter{Specialty: "市政", Region: "重庆中心城区", Period: "2026-06"})
	cover := ""
	for _, r := range sheets["封面"] {
		cover += strings.Join(r, " ") + "\n"
	}
	for _, want := range []string{"专业：市政", "地区：重庆中心城区", "计价期：2026-06", "取自库内最近项目「新项目Q」"} {
		if !strings.Contains(cover, want) {
			t.Errorf("封面应写明 %q，得到：\n%s", want, cover)
		}
	}
	found := false
	for _, r := range sheets["费用汇总"] {
		if len(r) > 3 && r[1] == "企业管理费" {
			found = true
			if r[3] != "0.05" {
				t.Errorf("企业管理费率应预填最近项目的 5%%，得到 %q", r[3])
			}
		}
	}
	if !found {
		t.Error("费用汇总应有「企业管理费」取费行")
	}
	if len(sheets["清单"]) == 0 || len(sheets["综合单价"]) == 0 {
		t.Error("模版应有「清单」与「综合单价」两张表")
	}
}
