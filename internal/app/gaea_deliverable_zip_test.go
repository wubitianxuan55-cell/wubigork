package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func makeXlsxChartFixture(t *testing.T) string {
	t.Helper()
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "chart.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "月份")
	f.SetCellValue("Sheet1", "B1", "销售额")
	f.SetCellValue("Sheet1", "A2", "一月")
	f.SetCellValue("Sheet1", "B2", 120)
	f.SetCellValue("Sheet1", "A3", "二月")
	f.SetCellValue("Sheet1", "B3", 200)
	f.SetCellValue("Sheet1", "A4", "三月")
	f.SetCellValue("Sheet1", "B4", 150)
	f.SetCellValue("Sheet1", "C1", "备注列")
	f.SetCellValue("Sheet1", "C2", "仅用于验证区域裁剪")
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return filepath.ToSlash(rel)
}

func TestExtractRangeChartData_Auto(t *testing.T) {
	rel := makeXlsxChartFixture(t)
	labels, values, err := extractRangeChartData(rel, "Sheet1", "")
	if err != nil {
		t.Fatalf("自动模式失败：%v", err)
	}
	if len(labels) != 3 || len(values) != 3 {
		t.Fatalf("自动模式应取 3 行数据（跳过表头），得到 labels=%d values=%d", len(labels), len(values))
	}
	if labels[0] != "一月" || values[1] != 200 {
		t.Fatalf("数据装配异常：labels=%v values=%v", labels, values)
	}
}

func TestExtractRangeChartData_ExplicitRange(t *testing.T) {
	rel := makeXlsxChartFixture(t)
	// 显式区域 A2:B4：全部视为数据行（无表头跳过）
	labels, values, err := extractRangeChartData(rel, "Sheet1", "A2:B4")
	if err != nil {
		t.Fatalf("显式区域失败：%v", err)
	}
	if len(labels) != 3 || values[0] != 120 || values[2] != 150 {
		t.Fatalf("显式区域装配异常：labels=%v values=%v", labels, values)
	}
}

func TestExtractRangeChartData_SingleCell(t *testing.T) {
	rel := makeXlsxChartFixture(t)
	// 单单元格 B3 → A1:B3（表头到选中行），跳过表头取 2 行
	labels, values, err := extractRangeChartData(rel, "Sheet1", "B3")
	if err != nil {
		t.Fatalf("单单元格模式失败：%v", err)
	}
	if len(labels) != 2 || values[0] != 120 || values[1] != 200 {
		t.Fatalf("单单元格装配异常：labels=%v values=%v", labels, values)
	}
}

func TestExtractRangeChartData_Validation(t *testing.T) {
	rel := makeXlsxChartFixture(t)
	// 越界区域（clamp 后无有效数据）应返回明确错误而非 panic
	if _, _, err := extractRangeChartData(rel, "Sheet1", "Z9:AA10"); err == nil {
		t.Fatal("越界区域应报错（clamp 后无数据）")
	}
	if _, _, err := extractRangeChartData(rel, "Sheet1", "not-a-range"); err == nil {
		t.Fatal("非法区域应报错")
	}
}

// TestGaeaXlsxChart_Smoke 真实调用 matplotlib 生成 PNG（默认跳过）：
//   GAEA_SMOKE_CHART=1 go test ./internal/app -run TestGaeaXlsxChart_Smoke -v
func TestGaeaXlsxChart_Smoke(t *testing.T) {
	if os.Getenv("GAEA_SMOKE_CHART") != "1" {
		t.Skip("设置 GAEA_SMOKE_CHART=1 运行真实图表 smoke")
	}
	rel := makeXlsxChartFixture(t)
	a := &App{core: &core{}}
	r, err := a.GaeaXlsxChart(XlsxChartInput{Rel: rel, Sheet: "Sheet1", ChartType: "bar"})
	if err != nil {
		t.Fatalf("GaeaXlsxChart 失败：%v", err)
	}
	if r.DataURL == "" || !strings.HasPrefix(r.DataURL, "data:image/png;base64,") {
		t.Fatalf("应返回 PNG dataURL，得到前缀 %q", r.DataURL[:min(len(r.DataURL), 40)])
	}
	if r.Labels != 3 || r.ChartType != "bar" {
		t.Fatalf("结果异常：labels=%d type=%s", r.Labels, r.ChartType)
	}
	if _, err := os.Stat(filepath.Join(gaeaCwd(), r.Path)); err != nil {
		t.Fatalf("产物未落盘：%v", err)
	}
}

func TestGaeaXlsxChart_Validation(t *testing.T) {
	rel := makeXlsxChartFixture(t)
	a := &App{core: &core{}}
	if _, err := a.GaeaXlsxChart(XlsxChartInput{Rel: "", ChartType: "bar"}); err == nil || !strings.Contains(err.Error(), "路径") {
		t.Fatalf("缺路径应报错，得到 %v", err)
	}
	if _, err := a.GaeaXlsxChart(XlsxChartInput{Rel: filepath.ToSlash(filepath.Join(".gaea", "uploads", "missing.xlsx"))}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("文件不存在应报错，得到 %v", err)
	}
	if _, err := a.GaeaXlsxChart(XlsxChartInput{Rel: rel, Sheet: "Sheet1", ChartType: "radar"}); err == nil || !strings.Contains(err.Error(), "图表类型") {
		t.Fatalf("非法图表类型应报错，得到 %v", err)
	}
}

func TestGaeaZipDeliverables(t *testing.T) {
	t.Chdir(t.TempDir())
	exportsDir := filepath.Join(".gaea", "exports")
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 造两个产物文件（含子目录同名文件验证相对路径保留）
	files := map[string]string{
		filepath.Join(exportsDir, "报告.docx"):            "docx content",
		filepath.Join(exportsDir, "成本测算.xlsx"):        "xlsx content",
		filepath.Join("sub", "报告.docx"):                 "sub docx",
		filepath.Join(exportsDir, "not-a-real-file.pdf"): "",
	}
	for p, content := range files {
		if content == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := &App{core: &core{}}

	// 空列表
	if _, err := a.GaeaZipDeliverables(nil); err == nil || !strings.Contains(err.Error(), "没有可打包") {
		t.Fatalf("空列表应报错，得到 %v", err)
	}

	// 正常打包：3 个存在的文件 + 1 个缺失（跳过）+ 目录（跳过）
	res, err := a.GaeaZipDeliverables([]string{
		filepath.ToSlash(filepath.Join(exportsDir, "报告.docx")),
		filepath.ToSlash(filepath.Join(exportsDir, "成本测算.xlsx")),
		filepath.ToSlash(filepath.Join("sub", "报告.docx")),
		filepath.ToSlash(filepath.Join(exportsDir, "not-a-real-file.pdf")),
		"sub", // 目录应跳过
	})
	if err != nil {
		t.Fatalf("打包失败：%v", err)
	}
	if res.Entries != 3 {
		t.Fatalf("应打包 3 个文件，得到 %d", res.Entries)
	}
	if res.Bytes == 0 || res.Name == "" || !strings.HasSuffix(res.Name, ".zip") {
		t.Fatalf("结果异常：bytes=%d name=%s", res.Bytes, res.Name)
	}
	abs := filepath.Join(gaeaCwd(), res.Path)
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("zip 未落盘：%v", err)
	}

	// 防穿越：拒绝绝对路径与 .. 路径
	if _, err := a.GaeaZipDeliverables([]string{`C:\Windows\win.ini`}); err == nil || !strings.Contains(err.Error(), "都不存在") {
		t.Fatalf("全部拒绝时应该报错，得到 %v", err)
	}
	// 全部路径非法 → 明确报错
	if _, err := a.GaeaZipDeliverables([]string{filepath.ToSlash(filepath.Join("..", "..", "etc", "passwd"))}); err == nil {
		t.Fatal(".. 穿越应被拒绝")
	}
}
