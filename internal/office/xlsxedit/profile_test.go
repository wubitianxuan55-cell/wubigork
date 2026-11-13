package xlsxedit

// profile_test.go — xlsx_profile 列概况钉子（波2 扩面）：夹具表覆盖数值列
// 含离群值、文本数字列、日期列（含写成文本的日期）、完全空行、整行重复、
// 低基数列、首尾空格——断言概况文本含关键行（模型可读文本契约）。

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// buildProfileFixture 概况夹具表（8 行 4 列，行 6 整行空、行 7 重复行 3）：
//
//	名称      金额        下单日      标签
//	北京      100        2025-01-05 A
//	上海      200        2025-02-01 B
//	北京      8800       2025-03-01 A   ← 8800 是离群值（>Q3+1.5IQR=450）
//	" 广州 "  300        2025-01-06 A   ← 首尾空格
//	（整行空）
//	上海      200        2025-02-01 B   ← 整行重复行 3
//	北京      ¥1,299(文本数字) 20250301(文本日期) A
func buildProfileFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profile.xlsx")
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "订单"); err != nil {
		t.Fatal(err)
	}
	head := []string{"名称", "金额", "下单日", "标签"}
	for c, h := range head {
		axis, _ := excelize.CoordinatesToCellName(c+1, 1)
		if err := f.SetCellValue("订单", axis, h); err != nil {
			t.Fatal(err)
		}
	}
	rows := [][]interface{}{
		{"北京", 100, nil, "A"},
		{"上海", 200, nil, "B"},
		{"北京", 8800, nil, "A"},
		{" 广州 ", 300, nil, "A"},
		nil, // 行 6 整行空
		{"上海", 200, nil, "B"},
		{"北京", "¥1,299", "20250301", "A"},
	}
	dates := map[int]time.Time{
		2: time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
		3: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		4: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		5: time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC),
		7: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	for r, row := range rows {
		abs := r + 2
		for c := 0; c < 4; c++ {
			axis, _ := excelize.CoordinatesToCellName(c+1, abs)
			if d, ok := dates[abs]; ok && c == 2 {
				// 日期列走真实日期格（序列值 + yyyy-mm-dd 显示格式）
				if err := writeDateCell(f, "订单", axis, d); err != nil {
					t.Fatal(err)
				}
				continue
			}
			if c >= len(row) || row[c] == nil {
				continue
			}
			if err := f.SetCellValue("订单", axis, row[c]); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProfileColumnSummary(t *testing.T) {
	path := buildProfileFixture(t)
	out, err := Profile(path, "订单", "")
	if err != nil {
		t.Fatal(err)
	}
	// 整块：空行 + 重复行
	if !strings.Contains(out, "完全空行 1（行 6）") {
		t.Errorf("缺完全空行行: %s", out)
	}
	if !strings.Contains(out, "整行重复 1 行") {
		t.Errorf("缺整行重复行: %s", out)
	}
	// A 名称：首尾空格 + 低基数
	if !strings.Contains(out, "1 格首尾空格") {
		t.Errorf("缺首尾空格行: %s", out)
	}
	if !strings.Contains(out, "3 种取值") {
		t.Errorf("缺低基数行: %s", out)
	}
	// B 金额：min/median/max + 文本数字 + 离群值（行 4）
	if !strings.Contains(out, "min 100，median 200，max 8800") {
		t.Errorf("缺数值三恒行: %s", out)
	}
	if !strings.Contains(out, "1 格数字写成文本") {
		t.Errorf("缺文本数字行: %s", out)
	}
	if !strings.Contains(out, "离群值 1 个") || !strings.Contains(out, "行 4") {
		t.Errorf("缺离群值行: %s", out)
	}
	// C 下单日：日期范围 + 写成文本的日期
	if !strings.Contains(out, "2025-01-05 ~ 2025-03-01") {
		t.Errorf("缺日期范围行: %s", out)
	}
	if !strings.Contains(out, "1 格日期写成文本") {
		t.Errorf("缺文本日期行: %s", out)
	}
	// D 标签：2 种取值
	if !strings.Contains(out, "2 种取值") || !strings.Contains(out, `"A" 4`) {
		t.Errorf("缺标签低基数行: %s", out)
	}
	// 数值文本混杂（B 列数值+文本数字并存）
	if !strings.Contains(out, "数值文本混杂") {
		t.Errorf("缺类型混杂行: %s", out)
	}
}

func TestProfileRangeAndErrors(t *testing.T) {
	path := buildProfileFixture(t)
	// range 限定：B1:B5（首行按表头），行 6 空行在区块外不计
	out, err := Profile(path, "订单", "B1:B5")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "完全空行") {
		t.Errorf("range 区块不应含行 6 空行: %s", out)
	}
	if !strings.Contains(out, "min 100，median 250，max 8800") {
		t.Errorf("range 区块缺数值概况: %s", out)
	}
	if !strings.Contains(out, "数值 4") || !strings.Contains(out, "B \"金额\"") {
		t.Errorf("range 区块数值计数/表头错: %s", out)
	}
	// 工作表不存在
	if _, err := Profile(path, "不存在", ""); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("表不存在应报错，实际 %v", err)
	}
	// 非法 range
	if _, err := Profile(path, "订单", "B2"); err == nil {
		t.Fatal("非法 range 应报错")
	}
	// 缺数据行
	if _, err := Profile(path, "订单", "A1:A1"); err == nil || !strings.Contains(err.Error(), "数据行") {
		t.Fatalf("单行区块应报错，实际 %v", err)
	}
	// 缺省 sheet=第一个表
	if _, err := Profile(path, "", ""); err != nil {
		t.Fatalf("缺省 sheet 应取第一个表: %v", err)
	}
}

func TestProfileRejectsOversizedBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.xlsx")
	f := excelize.NewFile()
	// 铺满 320×320 = 102400 格（> maxProfileCells=100000）
	for r := 1; r <= 320; r++ {
		for c := 1; c <= 320; c++ {
			axis, _ := excelize.CoordinatesToCellName(c, r)
			if err := f.SetCellValue("Sheet1", axis, "x"); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Profile(path, "Sheet1", ""); err == nil || !strings.Contains(err.Error(), "缩小范围") {
		t.Fatalf("超大区块应拒绝，实际 %v", err)
	}
}
