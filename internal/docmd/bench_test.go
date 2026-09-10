package docmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carmel/gooxml/document"
	"github.com/carmel/gooxml/spreadsheet"
)

// buildLargeDocx 生成约 300 段正文 + 20 张 20x6 表格的 docx（百页量级）。
func buildLargeDocx(t testing.TB) string {
	t.Helper()
	doc := document.New()
	for i := 0; i < 300; i++ {
		p := doc.AddParagraph()
		p.AddRun().AddText(fmt.Sprintf("第 %d 节 修复技术方案段落，描述污染场地修复目标与工艺路线选择依据。", i))
		p.AddRun().AddText("补充说明：固化稳定化工艺适用于重金属污染土壤。")
		p.AddRun().AddText("验收标准参照 GB 36600-2018 执行。")
	}
	for tIdx := 0; tIdx < 20; tIdx++ {
		table := doc.AddTable()
		for r := 0; r < 20; r++ {
			row := table.AddRow()
			for c := 0; c < 6; c++ {
				cell := row.AddCell()
				para := cell.AddParagraph()
				para.AddRun().AddText(fmt.Sprintf("R%dC%d", r, c))
			}
		}
	}
	path := filepath.Join(t.TempDir(), "large.docx")
	if err := doc.SaveToFile(path); err != nil {
		t.Fatalf("save large docx: %v", err)
	}
	return path
}

// buildLargeXlsx 生成 1 张 1000x10 的 xlsx。
func buildLargeXlsx(t testing.TB) string {
	t.Helper()
	ss := spreadsheet.New()
	sheet := ss.AddSheet()
	for r := 0; r < 1000; r++ {
		row := sheet.AddRow()
		for c := 0; c < 10; c++ {
			cell := row.AddCell()
			if r == 0 {
				cell.SetString(fmt.Sprintf("列%d", c))
			} else {
				cell.SetNumber(float64(r*10 + c))
			}
		}
	}
	path := filepath.Join(t.TempDir(), "large.xlsx")
	if err := ss.SaveToFile(path); err != nil {
		t.Fatalf("save large xlsx: %v", err)
	}
	return path
}

// buildLargePDF 生成约 1500 页含 BT/ET 文本的合成 PDF（纯文本流，模拟文本型 PDF）。
func buildLargePDF(t testing.TB) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	pageText := `BT /F1 12 Tf 72 720 Td (污染场地修复技术方案第 ) Tj (120 日历天内完成) Tj ET
BT /F2 10 Tf 72 700 Td [(固化稳定化) -20 (修复目标) ] TJ ET`
	for i := 0; i < 1500; i++ {
		fmt.Fprintf(&sb, "/Type /Page\n%s\n", pageText)
	}
	path := filepath.Join(t.TempDir(), "large.pdf")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write large pdf: %v", err)
	}
	return path
}

func BenchmarkDocxToMarkdown_Large(b *testing.B) {
	path := buildLargeDocx(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Convert(path, ""); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkXlsxToMarkdown_Large(b *testing.B) {
	path := buildLargeXlsx(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Convert(path, ""); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPDFToMarkdown_Large(b *testing.B) {
	path := buildLargePDF(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Convert(path, ""); err != nil {
			b.Fatal(err)
		}
	}
}
