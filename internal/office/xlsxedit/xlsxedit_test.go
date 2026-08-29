package xlsxedit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func buildBase(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "edit.xlsx")
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "数据")
	f.SetCellValue("数据", "A1", "城市")
	f.SetCellValue("数据", "B1", "金额")
	f.SetCellValue("数据", "A2", "北京-朝阳")
	f.SetCellValue("数据", "B2", 100)
	f.SetCellValue("数据", "A3", "上海-浦东")
	f.SetCellValue("数据", "B3", 200)
	f.SetCellValue("数据", "A4", " 广州 ")
	f.SetCellValue("数据", "B4", 300)
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

func readFormula(t *testing.T, path, sheet, cell string) string {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	formula, err := f.GetCellFormula(sheet, cell)
	if err != nil {
		t.Fatal(err)
	}
	return formula
}

func readValue(t *testing.T, path, sheet, cell string) string {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	v, err := f.GetCellValue(sheet, cell)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestApplyOps_SetFormulaAndTransform(t *testing.T) {
	path := buildBase(t)
	ops := []Op{
		{Type: "set_formula", Sheet: "数据", Target: "B5", Formula: "SUM(B2:B4)"},
		{Type: "transform", Sheet: "数据", Range: "C2:C4", Formula: "=B2*0.13"},
	}
	summary, err := ApplyOps(path, ops)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 2 {
		t.Fatalf("summary = %v", summary)
	}
	if got := readFormula(t, path, "数据", "B5"); got != "SUM(B2:B4)" {
		t.Errorf("B5 公式 = %q", got)
	}
	if got := readFormula(t, path, "数据", "C2"); got != "B2*0.13" {
		t.Errorf("C2 公式 = %q", got)
	}
	if got := readFormula(t, path, "数据", "C4"); got != "B4*0.13" {
		t.Errorf("C4 公式（行调整） = %q", got)
	}
}

func TestApplyOps_FillReplaceSplitClean(t *testing.T) {
	path := buildBase(t)
	ops := []Op{
		{Type: "fill_range", Sheet: "数据", Range: "D2:D4", Value: 0},
		{Type: "replace", Sheet: "数据", Range: "A2:A4", Find: "-", Replace: " "},
		{Type: "split_column", Sheet: "数据", Col: "A", Sep: "-", NewCols: []string{"B", "C"}, Headers: []string{"市", "区"}},
		{Type: "clean", Sheet: "数据", Range: "A2:A4", Trim: true, Upper: true},
	}
	if _, err := ApplyOps(path, ops); err != nil {
		t.Fatal(err)
	}
	// replace + split 在 A 列上依次执行：先 replace 把 "-" 换成 " "，再 split 按 "-" 已无分隔符。
	// 因此断言 clean 结果：去空格 + 大写
	if got := readValue(t, path, "数据", "A4"); got != "广州" {
		t.Errorf("A4 clean 后 = %q", got)
	}
	if got := readValue(t, path, "数据", "D2"); got != "0" {
		t.Errorf("D2 fill = %q", got)
	}
}

func TestApplyOps_SplitColumnFresh(t *testing.T) {
	// 单独验证拆分列（不叠加 replace）
	path := buildBase(t)
	ops := []Op{
		{Type: "split_column", Sheet: "数据", Col: "A", Sep: "-", NewCols: []string{"D", "E"}, Headers: []string{"市", "区"}},
	}
	if _, err := ApplyOps(path, ops); err != nil {
		t.Fatal(err)
	}
	if got := readValue(t, path, "数据", "D1"); got != "市" {
		t.Errorf("D1 表头 = %q", got)
	}
	if got := readValue(t, path, "数据", "D2"); got != "北京" {
		t.Errorf("D2 拆分 = %q", got)
	}
	if got := readValue(t, path, "数据", "E3"); got != "浦东" {
		t.Errorf("E3 拆分 = %q", got)
	}
}

func TestApplyOps_Errors(t *testing.T) {
	path := buildBase(t)
	if _, err := ApplyOps(path, []Op{{Type: "set_formula", Sheet: "不存在", Target: "A1", Formula: "SUM(A2)"}}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("期望工作表错误，得到 %v", err)
	}
	if _, err := ApplyOps(path, []Op{{Type: "nope", Sheet: "数据"}}); err == nil || !strings.Contains(err.Error(), "不支持") {
		t.Fatalf("期望不支持类型错误，得到 %v", err)
	}
	if _, err := ApplyOps(path, []Op{{Type: "transform", Sheet: "数据", Range: "A2:B4", Formula: "=A2"}}); err == nil || !strings.Contains(err.Error(), "单列") {
		t.Fatalf("期望单列限制错误，得到 %v", err)
	}
	if _, err := ApplyOps(path, nil); err == nil {
		t.Fatal("空操作应报错")
	}
}

func TestApplyOps_TransformKeepsAbsoluteRefs(t *testing.T) {
	path := buildBase(t)
	ops := []Op{
		{Type: "transform", Sheet: "数据", Range: "C2:C3", Formula: "=$B$1*B2"},
	}
	if _, err := ApplyOps(path, ops); err != nil {
		t.Fatal(err)
	}
	// 绝对引用 $B$1 不动，相对 B2 按行下移
	if got := readFormula(t, path, "数据", "C2"); got != "$B$1*B2" {
		t.Errorf("C2 = %q", got)
	}
	if got := readFormula(t, path, "数据", "C3"); got != "$B$1*B3" {
		t.Errorf("C3（应只动相对行） = %q", got)
	}
}

func TestPlanOps_DryRunDiff(t *testing.T) {
	path := buildBase(t)
	ops := []Op{
		{Type: "set_formula", Sheet: "数据", Target: "B5", Formula: "SUM(B2:B4)"},
		{Type: "set_value", Sheet: "数据", Target: "B2", Value: 111},
	}
	summary, changes, total, truncated, err := PlanOps(path, ops)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary) != 2 {
		t.Fatalf("summary = %v", summary)
	}
	if total != 2 || truncated {
		t.Fatalf("total=%d truncated=%v", total, truncated)
	}
	byCell := map[string]Change{}
	for _, c := range changes {
		byCell[c.Cell] = c
	}
	c, ok := byCell["B5"]
	if !ok || c.Formula != "SUM(B2:B4)" {
		t.Fatalf("B5 变更缺失: %+v", changes)
	}
	c, ok = byCell["B2"]
	if !ok || c.Before != "100" || c.After != "111" {
		t.Fatalf("B2 变更异常: %+v", c)
	}
	// 原文件未被触碰
	if got := readValue(t, path, "数据", "B2"); got != "100" {
		t.Fatalf("原文件被修改：B2 = %q", got)
	}
	if got := readFormula(t, path, "数据", "B5"); got != "" {
		t.Fatalf("原文件被写入公式：B5 = %q", got)
	}
}

func TestPlanOps_ReplaceRangeDiff(t *testing.T) {
	path := buildBase(t)
	ops := []Op{{Type: "replace", Sheet: "数据", Range: "A2:A4", Find: "-", Replace: " "}}
	_, changes, total, _, err := PlanOps(path, ops)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(changes) != 2 {
		t.Fatalf("替换应命中 2 格，total=%d changes=%+v", total, changes)
	}
	if changes[0].Before != "北京-朝阳" || changes[0].After != "北京 朝阳" {
		t.Fatalf("变更内容异常: %+v", changes[0])
	}
}

func TestPlanOps_InvalidOpsLeavesOriginal(t *testing.T) {
	path := buildBase(t)
	if _, _, _, _, err := PlanOps(path, []Op{{Type: "set_formula", Sheet: "不存在", Target: "A1", Formula: "SUM(A2)"}}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("期望工作表错误，得到 %v", err)
	}
	if _, _, _, _, err := PlanOps(path, nil); err == nil {
		t.Fatal("空操作应报错")
	}
	// 规划失败后原文件完好
	if got := readValue(t, path, "数据", "B2"); got != "100" {
		t.Fatalf("原文件被修改：B2 = %q", got)
	}
}

func TestApplyOps_RichOps(t *testing.T) {
	path := buildBase(t)
	tf := true
	size := 12.0
	ops := []Op{
		{Type: "set_style", Sheet: "数据", Range: "B1:B1", Style: &Style{
			Bold: &tf, Fill: "fff2cc", FontSize: &size, NumFmt: "0.00", Align: "center",
		}},
		{Type: "set_col_width", Sheet: "数据", Col: "A", Width: 22},
		{Type: "merge_cells", Sheet: "数据", Range: "D1:E1"},
	}
	if _, err := ApplyOps(path, ops); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// 样式叠加落盘：加粗 + 填充 + 字号 + 对齐
	id, err := f.GetCellStyle("数据", "B1")
	if err != nil {
		t.Fatal(err)
	}
	st, err := f.GetStyle(id)
	if err != nil || st == nil {
		t.Fatalf("读回样式失败: %v", err)
	}
	if st.Font == nil || !st.Font.Bold || st.Font.Size != 12 {
		t.Fatalf("字体样式异常: %+v", st.Font)
	}
	if len(st.Fill.Color) == 0 || st.Fill.Color[0] != "FFF2CC" {
		t.Fatalf("填充色异常: %+v", st.Fill)
	}
	if st.Alignment == nil || st.Alignment.Horizontal != "center" {
		t.Fatalf("对齐异常: %+v", st.Alignment)
	}
	// 列宽
	if w, err := f.GetColWidth("数据", "A"); err != nil || w != 22 {
		t.Fatalf("列宽 = %v, %v", w, err)
	}
	// 合并区域
	merges, err := f.GetMergeCells("数据")
	if err != nil || len(merges) == 0 {
		t.Fatalf("合并区域缺失: %v, %v", merges, err)
	}

	// 叠加语义：已有填充色时只加斜体，填充保留
	if _, err := ApplyOps(path, []Op{{Type: "set_style", Sheet: "数据", Target: "B1",
		Style: &Style{Italic: &tf}}}); err != nil {
		t.Fatal(err)
	}
	f2, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	id2, _ := f2.GetCellStyle("数据", "B1")
	st2, _ := f2.GetStyle(id2)
	if st2 == nil || st2.Font == nil || !st2.Font.Italic || !st2.Font.Bold {
		t.Fatalf("叠加后字体异常: %+v", st2.Font)
	}
	if len(st2.Fill.Color) == 0 || st2.Fill.Color[0] != "FFF2CC" {
		t.Fatalf("叠加后填充丢失: %+v", st2.Fill)
	}
}

func TestPlanOps_StyleOpNoValueDiff(t *testing.T) {
	path := buildBase(t)
	tf := true
	ops := []Op{
		{Type: "set_style", Sheet: "数据", Range: "A1:B4", Style: &Style{Bold: &tf}},
	}
	summary, changes, total, truncated, err := PlanOps(path, ops)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(changes) != 0 || truncated {
		t.Fatalf("纯样式不应产生值变更: total=%d changes=%d", total, len(changes))
	}
	if len(summary) != 1 || !strings.Contains(summary[0], "设置样式") {
		t.Fatalf("summary = %v", summary)
	}
}

func TestNormalizeHex(t *testing.T) {
	cases := []struct{ in, want string }{
		{"FFF2CC", "FFF2CC"}, {"#fff2cc", "FFF2CC"}, {" #FF0000 ", "FF0000"},
	}
	for _, c := range cases {
		if got, err := normalizeHex(c.in); err != nil || got != c.want {
			t.Errorf("normalizeHex(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"red", "#12345", "ZZZZZZ"} {
		if _, err := normalizeHex(bad); err == nil {
			t.Errorf("normalizeHex(%q) 应报错", bad)
		}
	}
}

func TestAdjustRowRefs_SkipsFunctionNames(t *testing.T) {
	cases := []struct{ in, want string }{
		{"LOG10(B2)+ROUND(B2,1)", "LOG10(B3)+ROUND(B3,1)"},
		{"$B$2+B2", "$B$2+B3"},
		{"B2", "B3"},
		{"SUM(B2:B4)", "SUM(B3:B5)"},
	}
	for _, c := range cases {
		if got := adjustRowRefs(c.in, 1); got != c.want {
			t.Errorf("adjustRowRefs(%q, 1) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildContext(t *testing.T) {
	path := buildBase(t)
	j, err := BuildContext(path, "数据")
	if err != nil {
		t.Fatal(err)
	}
	var ctx Context
	if err := json.Unmarshal([]byte(j), &ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Active != "数据" || len(ctx.Sheets) != 1 {
		t.Errorf("context 工作表错误: %+v", ctx)
	}
	if len(ctx.Headers) == 0 || ctx.Headers[0] != "城市" {
		t.Errorf("表头错误: %v", ctx.Headers)
	}
	if len(ctx.Sample) == 0 || ctx.Sample[0][0] != "北京-朝阳" {
		t.Errorf("抽样数据错误: %v", ctx.Sample)
	}
}

// TestSmokeRecalc 真实调用 recalc.py（LibreOffice 重算），默认跳过：
//   GAEA_SMOKE_RECALC=1 go test ./internal/office/xlsxedit -run TestSmokeRecalc -v
func TestSmokeRecalc(t *testing.T) {
	if os.Getenv("GAEA_SMOKE_RECALC") == "" {
		t.Skip("未设置 GAEA_SMOKE_RECALC")
	}
	path := buildBase(t)
	if _, err := ApplyOps(path, []Op{{Type: "set_formula", Sheet: "数据", Target: "B5", Formula: "SUM(B2:B4)"}}); err != nil {
		t.Fatal(err)
	}
	root, _ := os.Getwd()
	for i := 0; i < 4; i++ {
		if _, err := os.Stat(filepath.Join(root, ".gaea", "skills", "xlsx", "scripts", "recalc.py")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}
	rep, err := Recalc(path, root)
	if err != nil {
		t.Fatalf("重算失败: %v", err)
	}
	if rep.Status != "success" && rep.Status != "errors_found" {
		t.Fatalf("重算状态异常: %+v", rep)
	}
	if rep.TotalFormulas < 1 {
		t.Errorf("重算后公式数 = %d，期望 ≥1", rep.TotalFormulas)
	}
}
