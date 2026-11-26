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

// 行类空但有资源编码/消耗量的明细行显形为跳过行（太原 #2：此前被当成汇总行、
// 撞「已见编码」静默丢弃——工作簿里它仍计入定额单价，库内却缺这条消耗）。
// 清单自己的汇总行行类留空的合法形态不受影响。
func TestParseUnitPriceSheetBlankKindDetailRowIsReported(t *testing.T) {
	rows := [][]any{
		{"序号", "清单编码", "项目名称", "计量单位", "行类", "定额编码", "定额名称", "系数", "资源编码", "资源名称", "消耗量单位", "消耗量", "单价", "合价", "", "", "", "", "综合单价"},
		{"1", "Q1", "污染土清挖", "m³", "汇总", "", "", "", "", "", "", "", "", "", "", "", "", "", 70},
		{"", "", "", "", "定额", "D1", "挖机挖装", 1, "", "", "", "", "", "", "", "", "", "", ""},
		{"", "", "", "", "人工", "", "", "", "L01", "普工", "工日", 0.1, 300, 30, "", "", "", "", ""},
		{"", "", "", "", "", "", "", "", "EXC", "挖机", "台班", 0.02, 2000, 40, "", "", "", "", ""}, // 行类漏填
		{"2", "Q2", "污染土外运", "m³", "", "", "", "", "", "", "", "", "", "", "", "", "", "", 20},  // 汇总行行类留空：合法
	}
	b := &ProjectBundle{}
	parseUnitPriceSheet(sheetWithRows(t, "综合单价", rows), "综合单价", b)

	if len(b.Items) != 2 || b.Items[0].Code != "Q1" || b.Items[1].Code != "Q2" {
		t.Fatalf("两条清单都应识别（含行类留空的汇总行），实得 %+v", b.Items)
	}
	if len(b.Lines) != 1 || b.Lines[0].ResourceCode != "L01" {
		t.Fatalf("只应收行类齐全的那条明细，实得 %+v", b.Lines)
	}
	if len(b.Skipped) != 1 || b.Skipped[0].Row != 5 || !strings.Contains(b.Skipped[0].Reason, "行类为空") || !strings.Contains(b.Skipped[0].Reason, "EXC") {
		t.Fatalf("漏填行类的明细行应显形为跳过行并点名，实得 %+v", b.Skipped)
	}
}
