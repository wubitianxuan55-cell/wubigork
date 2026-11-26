package workcost

import (
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// ── 2026-11-26 造价反馈三批：导入解析的两处静默错误 ───────────────────────

func sheetWithRows(t *testing.T, name string, rows [][]any) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", name); err != nil {
		t.Fatalf("改名工作表失败: %v", err)
	}
	for i, row := range rows {
		for j, v := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
			if err := f.SetCellValue(name, cell, v); err != nil {
				t.Fatalf("写 %s!%s 失败: %v", name, cell, err)
			}
		}
	}
	return f
}

// 控制价只认纯控制价行、先到先得（百锦路 #7：此前任何含「控制价」的行都命中、
// 后行覆盖前行——「控制价利用率 62.64%」排在真控制价之后，8,913,200 被读成 0.63）。
func TestParseFeeSectionControlPrice(t *testing.T) {
	header := []any{"序号", "费用名称", "计算基础", "费率", "金额", "", "备注"}
	cases := []struct {
		name      string
		rows      [][]any
		wantPrice float64
		wantAlert bool
	}{
		{"利用率行在控制价之后：不得覆盖", [][]any{
			header,
			{"六", "招标控制价（含税）", "招标文件口径", "—", "—", 8913200},
			{"七", "控制价利用率", "", "", 0.6264},
		}, 8913200, false},
		{"下浮率行在控制价之前：不得抢占", [][]any{
			header,
			{"五", "控制价下浮率", "", "", 5},
			{"六", "招标控制价（含税）", "招标文件口径", "—", "—", 8913200},
		}, 8913200, false},
		{"只有派生行：控制价保持未读", [][]any{
			header,
			{"七", "占控制价比例", "", "", 105},
		}, 0, false},
		{"控制价小得不像工程价：提示疑似误读", [][]any{
			header,
			{"六", "招标控制价", "", "", 0.63},
		}, 0.63, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &ProjectBundle{}
			parseFeeSection(sheetWithRows(t, "费用汇总", tc.rows), "费用汇总", b)
			if b.Fee.ControlPrice != tc.wantPrice {
				t.Fatalf("控制价 = %v, want %v", b.Fee.ControlPrice, tc.wantPrice)
			}
			alerted := false
			for _, w := range b.Warnings {
				if strings.HasPrefix(w, ControlPriceSuspectPrefix) {
					alerted = true
				}
			}
			if alerted != tc.wantAlert {
				t.Fatalf("疑似误读告警 = %v, want %v（告警: %v）", alerted, tc.wantAlert, b.Warnings)
			}
		})
	}
}

// 漏填层级 / 缺编码 / 重复清单等残缺行的显形用例，随三级树表版式挪到
// projectimport_tree_test.go（TestParseUnitPriceTreeSkipsAreNamed）。旧「行类」
// 版式的同名用例随版式退役删除（2026-11-27）。
