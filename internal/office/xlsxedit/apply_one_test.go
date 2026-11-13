package xlsxedit

import (
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// openBase 打开 buildBase 生成的临时工作簿供 applyOne 直调（每例独立文件）。
func openBase(t *testing.T) *excelize.File {
	t.Helper()
	f, err := excelize.OpenFile(buildBase(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// TestApplyOneSummaryContract 锁 applyOne 的第三输出面（回执文案 summary，
// 2026-10-04 表驱动化前置契约：Err/After 由既有 ApplyOps 状态断言覆盖，
// summary 是拆分唯一可能静默漂移的面——照 schedule ops.go goldenCase 先例）。
func TestApplyOneSummaryContract(t *testing.T) {
	tf := true
	cases := []struct {
		name string
		op   Op
		want string
	}{
		{"set_formula", Op{Type: "set_formula", Sheet: "数据", Target: "B4", Formula: "B2*2"}, "写入公式 B4=B2*2"},
		{"set_value", Op{Type: "set_value", Sheet: "数据", Target: "C1", Value: 5}, "写入值 C1=5"},
		{"fill_range", Op{Type: "fill_range", Sheet: "数据", Range: "C1:C3", Value: 0}, "填充 C1:C3 = 0"},
		{"transform", Op{Type: "transform", Sheet: "数据", Range: "B2:B3", Formula: "B2*2"}, "逐行公式 B2:B3（2 行）"},
		{"replace", Op{Type: "replace", Sheet: "数据", Range: "A1:A3", Find: "北京", Replace: "北平"}, `替换 A1:A3："北京" → "北平"（1 格）`},
		{"split_column", Op{Type: "split_column", Sheet: "数据", Col: "A", Sep: "-", NewCols: []string{"C", "D"}}, "拆分 A 列 → C,D（5 格）"},
		{"clean", Op{Type: "clean", Sheet: "数据", Range: "D2:D2", Trim: true}, "清洗 D2:D2（1 格）"},
		{"set_style", Op{Type: "set_style", Sheet: "数据", Target: "B1", Style: &Style{Bold: &tf}}, "设置样式 B1（1 格）"},
		{"merge_cells", Op{Type: "merge_cells", Sheet: "数据", Range: "D1:E1"}, "合并 D1:E1"},
		{"unmerge_cells", Op{Type: "unmerge_cells", Sheet: "数据", Range: "D1:E1"}, "取消合并 D1:E1"},
		{"set_col_width", Op{Type: "set_col_width", Sheet: "数据", Col: "C", Width: 20}, "列宽 C = 20.0"},
		// ── 波2 扩面：数据作业 ops 契约例 ──
		{"fill_blanks_above", Op{Type: "fill_blanks", Sheet: "数据", Range: "C2:C4", With: "above"}, "填充空值 C2:C4：2 格（向上填充）"},
		{"fill_blanks_mean", Op{Type: "fill_blanks", Sheet: "数据", Range: "C2:C4", With: "mean"}, "填充空值 C2:C4：1 格（填均值 3）"},
		{"remove_duplicates", Op{Type: "remove_duplicates", Sheet: "数据"}, "去重 整行：删 1 行（保留首条；行 5）"},
		{"delete_rows_where", Op{Type: "delete_rows", Sheet: "数据", Where: &Where{Col: "A", Is: "contains", Value: "广州"}}, "删行 A 列 contains 广州：1 行（行 4）"},
		{"delete_rows_blank", Op{Type: "delete_rows", Sheet: "数据", Blank: true}, "删行 整行空白：1 行（行 5）"},
		{"delete_rows_row", Op{Type: "delete_rows", Sheet: "数据", Row: 3, Count: 2}, "删行 第 3-4 行：2 行（行 3, 4）"},
		{"insert_rows", Op{Type: "insert_rows", Sheet: "数据", Row: 2, Count: 2}, "插入 2 行（第 2 行前）"},
		{"insert_columns", Op{Type: "insert_columns", Sheet: "数据", Col: "B"}, "插入 1 列（B 列前）"},
		{"delete_columns", Op{Type: "delete_columns", Sheet: "数据", Col: "B"}, "删除 1 列（B 列起）"},
		{"sort", Op{Type: "sort", Sheet: "数据", By: []SortKey{{Column: "B", Order: "desc"}}}, "排序 3 行（B ↓）"},
		{"add_column", Op{Type: "add_column", Sheet: "数据", Header: "税额", Formula: "=B2*0.13"}, "加列 税额（C 列，3 行公式）"},
		{"copy_range", Op{Type: "copy_range", Sheet: "数据", Range: "A1:B3", To: "清洗后!A1"}, "复制 数据!A1:B3 → 清洗后!A1:B3（6 格，复制了值/原始值；不含公式与样式）"},
		{"clean_to_number", Op{Type: "clean", Sheet: "数据", Range: "D2:D2", To: "number"}, "清洗 D2:D2（1 格，转 number）"},
		{"clean_to_date", Op{Type: "clean", Sheet: "数据", Range: "D3:D3", To: "date"}, "清洗 D3:D3（1 格，转 date）"},
		{"clean_to_number_failed", Op{Type: "clean", Sheet: "数据", Range: "D4:D4", To: "number"}, "清洗 D4:D4（0 格，转 number；1 格解析失败，原样保留）"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := openBase(t)
			if tc.name == "clean" {
				if err := f.SetCellValue("数据", "D2", "  x  "); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "clean_to_number" {
				if err := f.SetCellValue("数据", "D2", "¥1,299"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "clean_to_date" {
				if err := f.SetCellValue("数据", "D3", "2026年3月1日"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "clean_to_number_failed" {
				if err := f.SetCellValue("数据", "D4", "abc"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "fill_blanks_above" {
				if err := f.SetCellValue("数据", "C2", 1); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "fill_blanks_mean" {
				if err := f.SetCellValue("数据", "C2", 2); err != nil {
					t.Fatal(err)
				}
				if err := f.SetCellValue("数据", "C3", 4); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "remove_duplicates" {
				if err := f.SetCellValue("数据", "A5", "上海-浦东"); err != nil {
					t.Fatal(err)
				}
				if err := f.SetCellValue("数据", "B5", 200); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "delete_rows_blank" {
				if err := f.SetCellValue("数据", "A5", "   "); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "copy_range" {
				if _, err := f.NewSheet("清洗后"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "unmerge_cells" {
				if _, err := applyOne(f, Op{Type: "merge_cells", Sheet: "数据", Range: "D1:E1"}); err != nil {
					t.Fatal(err)
				}
			}
			got, err := applyOne(f, tc.op)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if got != tc.want {
				t.Fatalf("summary = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestApplyUnmergeCellsState unmerge_cells 此前零测试覆盖（2026-10-04 补）：
// 合并→取消合并 的落盘状态闭环。
func TestApplyUnmergeCellsState(t *testing.T) {
	f := openBase(t)
	if _, err := applyOne(f, Op{Type: "merge_cells", Sheet: "数据", Range: "D1:E1"}); err != nil {
		t.Fatal(err)
	}
	ms, err := f.GetMergeCells("数据")
	if err != nil || len(ms) != 1 {
		t.Fatalf("merge 后应恰 1 个合并区: %v %v", ms, err)
	}
	if _, err := applyOne(f, Op{Type: "unmerge_cells", Sheet: "数据", Range: "D1:E1"}); err != nil {
		t.Fatal(err)
	}
	ms, err = f.GetMergeCells("数据")
	if err != nil || len(ms) != 0 {
		t.Fatalf("unmerge 后应 0 个合并区: %v %v", ms, err)
	}
}

// TestOpHandlersKeySetLocked 注册表键集冻结：新增 op 类型必须显式扩本清单
// 并补 summary 契约例（与 Op.Type 的前端契约面同账）。
func TestOpHandlersKeySetLocked(t *testing.T) {
	want := []string{
		"add_column", "clean", "copy_range", "delete_columns", "delete_rows",
		"fill_blanks", "fill_range", "insert_columns", "insert_rows",
		"merge_cells", "remove_duplicates", "replace", "set_col_width",
		"set_formula", "set_style", "set_value", "sort", "split_column",
		"transform", "unmerge_cells",
	}
	if len(opHandlers) != len(want) {
		t.Fatalf("opHandlers 有 %d 键，want %d", len(opHandlers), len(want))
	}
	for _, k := range want {
		if opHandlers[k] == nil {
			t.Errorf("opHandlers 缺键 %q", k)
		}
	}
}

// TestApplyOneUnknownTypeFailsClosed 未知 op 类型逐字图钉（fail-closed 出口）。
func TestApplyOneUnknownTypeFailsClosed(t *testing.T) {
	f := openBase(t)
	_, err := applyOne(f, Op{Type: "nope"})
	if err == nil || !strings.Contains(err.Error(), "不支持的操作类型") {
		t.Fatalf("未知类型应报「不支持的操作类型」，实际 %v", err)
	}
}

// ── 波2 扩面：数据作业 ops 落盘状态测试 ──────────────────────────────────

func TestApplyOps_FillBlanksAboveMeanMode(t *testing.T) {
	path := buildBase(t)
	ops := []Op{
		// above：C2=1 向下填 C3/C4
		{Type: "fill_blanks", Sheet: "数据", Range: "C2:C4", With: "above"},
		// mean：D2=2 D3=4 → D4 填均值 3
		{Type: "fill_blanks", Sheet: "数据", Col: "D", With: "mean"},
	}
	// 预置均值列数据（写入原文件再跑 ops，Col 形态走全表行界）
	f0, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "C2", 1); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "D2", 2); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "D3", 4); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "D4", ""); err != nil {
		t.Fatal(err)
	}
	if err := f0.Save(); err != nil {
		t.Fatal(err)
	}
	if err := f0.Close(); err != nil {
		t.Fatal(err)
	}
	summary, err := ApplyOps(path, ops)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 2 {
		t.Fatalf("summary = %v", summary)
	}
	if got := readValue(t, path, "数据", "C3"); got != "1" {
		t.Errorf("C3 向上填充 = %q", got)
	}
	if got := readValue(t, path, "数据", "C4"); got != "1" {
		t.Errorf("C4 向上填充 = %q", got)
	}
	if got := readValue(t, path, "数据", "D4"); got != "3" {
		t.Errorf("D4 均值填充 = %q", got)
	}
	// 表头行（C1/D1）不该被填
	if got := readValue(t, path, "数据", "D1"); got != "" {
		t.Errorf("D1 表头被误填 = %q", got)
	}
	// mode：众数按出现最多取值
	f1, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f1.Close() }()
	for _, pair := range [][2]string{{"E2", "A"}, {"E3", "B"}, {"E4", "A"}} {
		if err := f1.SetCellValue("数据", pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := f1.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyOps(path, []Op{{Type: "fill_blanks", Sheet: "数据", Col: "E", With: "mode"}}); err != nil {
		t.Fatal(err)
	}
	// E 列无空格可填：不报错即可；再造一个空格验证众数落值
	f2, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f2.Close() }()
	if err := f2.SetCellValue("数据", "E5", ""); err != nil {
		t.Fatal(err)
	}
	// 让 GetRows 覆盖第 5 行：F5 放一个占位
	if err := f2.SetCellValue("数据", "F5", "x"); err != nil {
		t.Fatal(err)
	}
	if err := f2.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyOps(path, []Op{{Type: "fill_blanks", Sheet: "数据", Col: "E", With: "mode"}}); err != nil {
		t.Fatal(err)
	}
	if got := readValue(t, path, "数据", "E5"); got != "A" {
		t.Errorf("E5 众数填充 = %q, want A", got)
	}
}

func TestApplyOps_RemoveDuplicatesKeepsFirst(t *testing.T) {
	path := buildBase(t)
	f0, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "A5", "上海-浦东"); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "B5", 200); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "A6", "成都-高新"); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "B6", 100); err != nil {
		t.Fatal(err)
	}
	if err := f0.Save(); err != nil {
		t.Fatal(err)
	}
	if err := f0.Close(); err != nil {
		t.Fatal(err)
	}
	// 按城市列（A 列去 " -区县" 后不整行同）：直接按整行判重——A5/B5 与 A3/B3 同
	summary, err := ApplyOps(path, []Op{{Type: "remove_duplicates", Sheet: "数据"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 1 || !strings.Contains(summary[0], "删 1 行") || !strings.Contains(summary[0], "行 5") {
		t.Fatalf("summary = %v", summary)
	}
	// 第 5 行（后出现的重复）应被删掉：行 5 变成成都
	if got := readValue(t, path, "数据", "A5"); got != "成都-高新" {
		t.Errorf("重复行未删：A5 = %q", got)
	}
	if got := readValue(t, path, "数据", "A3"); got != "上海-浦东" {
		t.Errorf("首条应保留：A3 = %q", got)
	}
}

func TestApplyOps_DeleteRowsThreeForms(t *testing.T) {
	// 形态一：where 条件删行（contains）
	path := buildBase(t)
	summary, err := ApplyOps(path, []Op{{Type: "delete_rows", Sheet: "数据",
		Where: &Where{Col: "A", Is: "contains", Value: "广州"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary[0], "1 行（行 4）") {
		t.Fatalf("summary = %v", summary)
	}
	if got := readValue(t, path, "数据", "A4"); got != "" {
		t.Errorf("行 4 应删空：A4 = %q", got)
	}

	// 形态二：blank 整行空删（行 5 只有空白字符）
	path = buildBase(t)
	f0, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "A5", "   "); err != nil {
		t.Fatal(err)
	}
	if err := f0.Save(); err != nil {
		t.Fatal(err)
	}
	if err := f0.Close(); err != nil {
		t.Fatal(err)
	}
	summary, err = ApplyOps(path, []Op{{Type: "delete_rows", Sheet: "数据", Blank: true}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary[0], "行 5") {
		t.Fatalf("summary = %v", summary)
	}

	// 形态三：row 直接删（含 count）
	path = buildBase(t)
	summary, err = ApplyOps(path, []Op{{Type: "delete_rows", Sheet: "数据", Row: 3, Count: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary[0], "2 行（行 3, 4）") {
		t.Fatalf("summary = %v", summary)
	}
	if got := readValue(t, path, "数据", "A3"); got != "" {
		t.Errorf("行 3 应删空：A3 = %q", got)
	}

	// where text：写成文本的数字被判中（SetCellValue 存字符串）
	path = buildBase(t)
	f1, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f1.SetCellValue("数据", "A5", "20250301"); err != nil {
		t.Fatal(err)
	}
	if err := f1.Save(); err != nil {
		t.Fatal(err)
	}
	if err := f1.Close(); err != nil {
		t.Fatal(err)
	}
	summary, err = ApplyOps(path, []Op{{Type: "delete_rows", Sheet: "数据",
		Where: &Where{Col: "A", Is: "text"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary[0], "行 5") {
		t.Fatalf("text 条件应删行 5：summary = %v", summary)
	}
	// 互斥校验：三形态全缺应报错
	path = buildBase(t)
	if _, err := ApplyOps(path, []Op{{Type: "delete_rows", Sheet: "数据"}}); err == nil {
		t.Fatal("delete_rows 三形态全缺应报错")
	}
}

func TestApplyOps_InsertRowsAndColumnsShift(t *testing.T) {
	path := buildBase(t)
	summary, err := ApplyOps(path, []Op{
		{Type: "insert_rows", Sheet: "数据", Row: 2, Count: 2},
		{Type: "insert_columns", Sheet: "数据", Col: "B", Count: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 2 {
		t.Fatalf("summary = %v", summary)
	}
	// B 列插入后：旧 B（金额）右移到 C；行 2 起插入两行后旧数据下移两行
	if got := readValue(t, path, "数据", "C4"); got != "100" {
		t.Errorf("插入后旧 B2 应在 C4 = %q", got)
	}
	if got := readValue(t, path, "数据", "A4"); got != "北京-朝阳" {
		t.Errorf("插入后旧 A2 应在 A4 = %q", got)
	}
	// 删除列：金额列（C）删掉后城市独存
	if _, err := ApplyOps(path, []Op{{Type: "delete_columns", Sheet: "数据", Col: "C"}}); err != nil {
		t.Fatal(err)
	}
	if got := readValue(t, path, "数据", "A4"); got != "北京-朝阳" {
		t.Errorf("删除列不应动城市 = %q", got)
	}
	if got := readValue(t, path, "数据", "B4"); got != "" {
		t.Errorf("金额列应删空 = %q", got)
	}
}

func TestApplyOps_SortMultiKeyValues(t *testing.T) {
	path := buildBase(t)
	// B 降序：300(广州) → 200(上海) → 100(北京)
	summary, err := ApplyOps(path, []Op{{Type: "sort", Sheet: "数据",
		By: []SortKey{{Column: "B", Order: "desc"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 1 || !strings.Contains(summary[0], "排序 3 行（B ↓）") {
		t.Fatalf("summary = %v", summary)
	}
	if got := readValue(t, path, "数据", "B2"); got != "300" {
		t.Errorf("降序首行 B2 = %q, want 300", got)
	}
	if got := readValue(t, path, "数据", "A2"); got != " 广州 " {
		t.Errorf("行随金额整体移动：A2 = %q", got)
	}
	if got := readValue(t, path, "数据", "B4"); got != "100" {
		t.Errorf("降序末行 B4 = %q, want 100", got)
	}
	// 表头不动
	if got := readValue(t, path, "数据", "A1"); got != "城市" {
		t.Errorf("表头被移动：A1 = %q", got)
	}
	// 升序数值优先于文本比：缺省 asc
	if _, err := ApplyOps(path, []Op{{Type: "sort", Sheet: "数据", By: []SortKey{{Column: "B"}}}}); err != nil {
		t.Fatal(err)
	}
	if got := readValue(t, path, "数据", "B2"); got != "100" {
		t.Errorf("升序首行 B2 = %q, want 100", got)
	}
	// 缺 by 应报错
	if _, err := ApplyOps(path, []Op{{Type: "sort", Sheet: "数据"}}); err == nil {
		t.Fatal("sort 缺 by 应报错")
	}
}

func TestApplyOps_AddColumnFormulaAndValue(t *testing.T) {
	path := buildBase(t)
	summary, err := ApplyOps(path, []Op{{Type: "add_column", Sheet: "数据",
		Header: "税额", Formula: "=B2*0.13"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 1 || !strings.Contains(summary[0], "加列 税额（C 列，3 行公式）") {
		t.Fatalf("summary = %v", summary)
	}
	if got := readValue(t, path, "数据", "C1"); got != "税额" {
		t.Errorf("表头 C1 = %q", got)
	}
	if got := readFormula(t, path, "数据", "C2"); got != "B2*0.13" {
		t.Errorf("C2 公式应为第 2 行原式 = %q", got)
	}
	if got := readFormula(t, path, "数据", "C4"); got != "B4*0.13" {
		t.Errorf("C4 公式应按行偏移 = %q", got)
	}
	// after：插在 A 列后（新列 B）
	if _, err := ApplyOps(path, []Op{{Type: "add_column", Sheet: "数据",
		Header: "备注", Value: "待核", After: "A"}}); err != nil {
		t.Fatal(err)
	}
	if got := readValue(t, path, "数据", "B1"); got != "备注" {
		t.Errorf("after A 后新列表头应在 B1 = %q", got)
	}
	if got := readValue(t, path, "数据", "B3"); got != "待核" {
		t.Errorf("固定值列 B3 = %q", got)
	}
	// 旧金额列右移到 C（税额公式列再右移到 D）
	if got := readValue(t, path, "数据", "C2"); got != "100" {
		t.Errorf("旧 B2 应右移到 C2 = %q", got)
	}
	// values:true 直接报错（公式重算走保存后的 Recalc 链）
	path2 := buildBase(t)
	_, err = ApplyOps(path2, []Op{{Type: "add_column", Sheet: "数据",
		Header: "合计", Formula: "=B2*2", Values: true}})
	if err == nil || !strings.Contains(err.Error(), "values:true") {
		t.Fatalf("values:true 应报错，实际 %v", err)
	}
}

func TestApplyOps_CopyRangeValues(t *testing.T) {
	path := buildBase(t)
	f0, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f0.NewSheet("清洗后"); err != nil {
		t.Fatal(err)
	}
	if err := f0.Save(); err != nil {
		t.Fatal(err)
	}
	if err := f0.Close(); err != nil {
		t.Fatal(err)
	}
	summary, err := ApplyOps(path, []Op{{Type: "copy_range", Sheet: "数据",
		Range: "A1:B4", To: "清洗后!A1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 1 || !strings.Contains(summary[0], "复制 数据!A1:B4 → 清洗后!A1:B4") || !strings.Contains(summary[0], "复制了值") {
		t.Fatalf("summary = %v", summary)
	}
	// 值已复制且数值保数值形态（显示 100）
	if got := readValue(t, path, "清洗后", "B2"); got != "100" {
		t.Errorf("清洗后 B2 = %q", got)
	}
	if got := readValue(t, path, "清洗后", "A4"); got != " 广州 " {
		t.Errorf("清洗后 A4 = %q", got)
	}
	// 原表保留
	if got := readValue(t, path, "数据", "B2"); got != "100" {
		t.Errorf("原表应保留：数据 B2 = %q", got)
	}
	// 目标表不存在应报错
	if _, err := ApplyOps(path, []Op{{Type: "copy_range", Sheet: "数据",
		Range: "A1:B2", To: "不存在!A1"}}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("目标表缺失应报错，实际 %v", err)
	}
}

func TestApplyOps_CleanToNumberAndDate(t *testing.T) {
	path := buildBase(t)
	f0, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "C2", "¥1,299"); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "C3", "89元"); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "C4", "abc"); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "D2", "2026年3月1日"); err != nil {
		t.Fatal(err)
	}
	if err := f0.SetCellValue("数据", "D3", "2025/3/1"); err != nil {
		t.Fatal(err)
	}
	if err := f0.Save(); err != nil {
		t.Fatal(err)
	}
	if err := f0.Close(); err != nil {
		t.Fatal(err)
	}
	summary, err := ApplyOps(path, []Op{
		{Type: "clean", Sheet: "数据", Range: "C2:C4", To: "number"},
		{Type: "clean", Sheet: "数据", Range: "D2:D3", To: "date"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 2 || !strings.Contains(summary[0], "2 格，转 number；1 格解析失败") {
		t.Fatalf("summary = %v", summary)
	}
	if got := readValue(t, path, "数据", "C2"); got != "1299" {
		t.Errorf("¥1,299 应转数值 = %q", got)
	}
	if got := readValue(t, path, "数据", "C3"); got != "89" {
		t.Errorf("89元 应转数值 = %q", got)
	}
	if got := readValue(t, path, "数据", "C4"); got != "abc" {
		t.Errorf("解析失败应原样保留 = %q", got)
	}
	// 日期格：写入日期序列值且带 yyyy-mm-dd 显示格式
	f1, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f1.Close() }()
	raw2, _ := f1.GetCellValue("数据", "D2", excelize.Options{RawCellValue: true})
	if raw2 != "46082" { // 2026-03-01 的 Excel 序列值
		t.Errorf("D2 应为日期序列值 = %q, want 46082", raw2)
	}
	styleID, _ := f1.GetCellStyle("数据", "D2")
	st, _ := f1.GetStyle(styleID)
	if st == nil || st.CustomNumFmt == nil || *st.CustomNumFmt != "yyyy-mm-dd" {
		t.Errorf("D2 应带 yyyy-mm-dd 显示格式: %+v", st)
	}
	if got := readValue(t, path, "数据", "D3"); got != "2025-03-01" {
		t.Errorf("D3 日期显示 = %q", got)
	}
	// Map 取值改名 + Remove 删字样 + NoSpaces
	path2 := buildBase(t)
	summary2, err := ApplyOps(path2, []Op{
		{Type: "clean", Sheet: "数据", Range: "A2:A4", Map: map[string]string{"北京-朝阳": "BJ", "上海-浦东": "SH"}},
		{Type: "clean", Sheet: "数据", Range: "A4:A4", NoSpaces: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary2) != 2 {
		t.Fatalf("summary = %v", summary2)
	}
	if got := readValue(t, path2, "数据", "A2"); got != "BJ" {
		t.Errorf("Map 改名 A2 = %q", got)
	}
	if got := readValue(t, path2, "数据", "A4"); got != "广州" {
		t.Errorf("NoSpaces A4 = %q", got)
	}
	// To 非法值 fail-closed
	path3 := buildBase(t)
	if _, err := ApplyOps(path3, []Op{{Type: "clean", Sheet: "数据", Range: "A2:A2", To: "float"}}); err == nil {
		t.Fatal("clean to=float 应报错")
	}
}
