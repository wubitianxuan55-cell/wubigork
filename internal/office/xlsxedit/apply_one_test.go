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
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := openBase(t)
			if tc.name == "clean" {
				if err := f.SetCellValue("数据", "D2", "  x  "); err != nil {
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
		"clean", "fill_range", "merge_cells", "replace", "set_col_width",
		"set_formula", "set_style", "set_value", "split_column", "transform",
		"unmerge_cells",
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
