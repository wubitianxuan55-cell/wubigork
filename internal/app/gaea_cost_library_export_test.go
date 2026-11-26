package app

// gaea_cost_library_export_test.go — 造价库列表导出绑定（2026-11-25 造价数据库
// 推倒重建）与删除护栏：四类导出的表结构、筛选口径、拒绝覆盖；定额被引用拒删。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/skill"
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

func TestCostLibraryExportQuotaResource(t *testing.T) {
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
}

// TestCostLibraryExportTemplate 测算样表出口（kind=template；2026-11-25 删渲染器
// 拍板）：把样表复制到目标路径——六表结构完整、可直接被解析链吃下；不按库渲染、
// 不预填（工料机价格与费率由模型查库填）。filter 参数保留签名兼容，不再使用。
func TestCostLibraryExportTemplate(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()

	out := filepath.Join(t.TempDir(), "样表")
	if _, err := a.GaeaCostLibraryExport("template", workcost.LibraryFilter{}, out); err != nil {
		t.Fatalf("导出测算样表失败: %v", err)
	}
	raw, err := os.ReadFile(out + ".xlsx")
	if err != nil {
		t.Fatalf("样表应已落盘: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("样表为空")
	}
	// 与内嵌种子逐字节一致（复制语义，不是渲染）。
	if string(raw) != string(skill.CostEstimateSampleXLSX()) {
		t.Error("样表出口应与内嵌种子逐字节一致")
	}
	// 解析链吃得下：六表、60 条样例预填、费率样例口径。
	path := out + ".xlsx"
	b, perr := workcost.ParseProjectWorkbook(path)
	if perr != nil {
		t.Fatalf("样表应可被导入解析链解析: %v", perr)
	}
	if len(b.Sheets) != 6 || len(b.Resources) != 60 {
		t.Fatalf("样表应 6 张表 / 60 条样例工料机，得到 %v / %d", b.Sheets, len(b.Resources))
	}
	if b.Fee.ManagementRate != 0.05 || b.Fee.ProfitRate != 0.04 || b.Fee.TaxRate != 0.09 {
		t.Errorf("样表费率样例应 5%%/4%%/0%%/9%%，得到 %+v", b.Fee)
	}
	// 工作区覆盖：.gaea/skills/cost-estimate/样表 存在时优先于种子。
	ws := t.TempDir()
	origCfg := ga.cfg
	ga.cfg = &config.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = origCfg })
	overrideDir := filepath.Join(ws, ".gaea", "skills", "cost-estimate")
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	override := filepath.Join(overrideDir, "样表-成本测算模版.xlsx")
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SaveAs(override); err != nil {
		t.Fatalf("写工作区覆盖样表失败: %v", err)
	}
	out2 := filepath.Join(t.TempDir(), "样表2")
	if _, err := a.GaeaCostLibraryExport("template", workcost.LibraryFilter{}, out2); err != nil {
		t.Fatalf("导出（覆盖态）失败: %v", err)
	}
	got, err := os.ReadFile(out2 + ".xlsx")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(override)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Error("工作区覆盖样表应优先于内嵌种子")
	}
}
