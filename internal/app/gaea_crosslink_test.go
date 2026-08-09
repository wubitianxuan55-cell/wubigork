package app

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func writeChartXLSX(t *testing.T) string {
	t.Helper()
	path := filepath.Join(".gaea", "uploads", "chart-data.xlsx")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "预算")
	f.SetCellValue("预算", "A1", "项目")
	f.SetCellValue("预算", "B1", "金额")
	f.SetCellValue("预算", "A2", "设备")
	f.SetCellValue("预算", "B2", 120)
	f.SetCellValue("预算", "A3", "人工")
	f.SetCellValue("预算", "B3", 80)
	f.SetCellValue("预算", "A4", "材料")
	f.SetCellValue("预算", "B4", 200)
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

func TestGaeaCrossEmbed_Validation(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	if _, err := a.GaeaCrossEmbed(CrossEmbedInput{Into: "docx"}); err == nil || !strings.Contains(err.Error(), "xlsx") {
		t.Fatalf("期望缺数据源错误，得到 %v", err)
	}
	if _, err := a.GaeaCrossEmbed(CrossEmbedInput{XlsxRel: "a.xlsx", Into: "pdf"}); err == nil || !strings.Contains(err.Error(), "目标") {
		t.Fatalf("期望目标错误，得到 %v", err)
	}
	if _, err := a.GaeaCrossEmbed(CrossEmbedInput{XlsxRel: "a.xlsx", Into: "docx", ChartType: "gauge"}); err == nil || !strings.Contains(err.Error(), "图表类型") {
		t.Fatalf("期望图表类型错误，得到 %v", err)
	}
}

// TestSmokeCrossEmbed 真实图表嵌入 docx/pptx（默认跳过）：
//   GAEA_SMOKE_CROSS=1 go test ./internal/app -run TestSmokeCrossEmbed -v
func TestSmokeCrossEmbed(t *testing.T) {
	if os.Getenv("GAEA_SMOKE_CROSS") == "" {
		t.Skip("未设置 GAEA_SMOKE_CROSS")
	}
	t.Chdir(t.TempDir())
	rel := writeChartXLSX(t)
	a := &App{}
	for _, into := range []string{"docx", "pptx"} {
		got, err := a.GaeaCrossEmbed(CrossEmbedInput{
			XlsxRel: filepath.ToSlash(rel), Sheet: "预算", ChartType: "bar",
			Title: "预算构成", Into: into,
		})
		if err != nil {
			t.Fatalf("%s 嵌入失败: %v", into, err)
		}
		if got.ChartPath == "" {
			t.Fatalf("%s 缺少图表路径", into)
		}
		if _, err := os.Stat(got.ChartPath); err != nil {
			t.Fatalf("图表文件不存在: %v", err)
		}
		zr, err := zip.OpenReader(got.Path)
		if err != nil {
			t.Fatalf("%s 不是合法 zip: %v", into, err)
		}
		hasMedia := false
		for _, f := range zr.File {
			if strings.Contains(f.Name, "media/") || strings.Contains(f.Name, "image") {
				hasMedia = true
				break
			}
		}
		zr.Close()
		if !hasMedia {
			t.Fatalf("%s 中未找到嵌入的图表图片", into)
		}
		t.Logf("%s 产物: %s（%d 字节），图表 %s", into, got.Name, got.Size, got.ChartPath)
	}
}
