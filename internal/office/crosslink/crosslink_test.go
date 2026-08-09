package crosslink

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func buildXLSX(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.xlsx")
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
	f.SetCellValue("预算", "A5", "其他")
	f.SetCellValue("预算", "B5", 50)
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

func TestExtractChartData(t *testing.T) {
	labels, values, header, rows, err := ExtractChartData(buildXLSX(t), "预算", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) != 4 || len(values) != 4 {
		t.Fatalf("labels/values 长度异常: %d/%d", len(labels), len(values))
	}
	if labels[0] != "设备" || values[1] != 80 {
		t.Errorf("数据提取异常: %v %v", labels, values)
	}
	if len(header) != 2 || header[0] != "项目" {
		t.Errorf("表头异常: %v", header)
	}
	if len(rows) != 5 {
		t.Errorf("rows 应含表头+4 数据行: %d", len(rows))
	}
}

func TestExtractChartData_Range(t *testing.T) {
	labels, values, _, _, err := ExtractChartData(buildXLSX(t), "预算", "A2:B3")
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) != 2 || labels[0] != "设备" || labels[1] != "人工" || values[0] != 120 || values[1] != 80 {
		t.Errorf("范围提取异常: %v %v", labels, values)
	}
}

func TestBuildSpecs(t *testing.T) {
	labels, values, header, rows, err := ExtractChartData(buildXLSX(t), "预算", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = labels
	_ = values
	docx := BuildDocxSpec("预算分析", "chart.png", header, rows)
	b, _ := json.Marshal(docx)
	if !strings.Contains(string(b), `"type":"image"`) || !strings.Contains(string(b), `"type":"table"`) {
		t.Errorf("docx spec 缺少图片/表格: %s", b)
	}
	pptx := BuildPptxSpec("预算分析", "chart.png", header, rows)
	b2, _ := json.Marshal(pptx)
	if !strings.Contains(string(b2), `"image":"chart.png"`) || !strings.Contains(string(b2), "数据明细") {
		t.Errorf("pptx spec 异常: %s", b2)
	}
}

// TestSmokeGenerateChart 真实 matplotlib 生成图表（默认跳过）：
//   GAEA_SMOKE_CHART=1 go test ./internal/office/crosslink -run TestSmokeGenerateChart -v
func TestSmokeGenerateChart(t *testing.T) {
	if os.Getenv("GAEA_SMOKE_CHART") == "" {
		t.Skip("未设置 GAEA_SMOKE_CHART")
	}
	out := filepath.Join(t.TempDir(), "chart.png")
	if err := GenerateChartPNG([]string{"设备", "人工", "材料"}, []float64{120, 80, 200}, "bar", "预算分析", out); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("图表为空文件")
	}
	t.Logf("chart png: %d bytes", info.Size())
}
