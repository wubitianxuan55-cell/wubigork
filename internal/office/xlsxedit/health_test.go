package xlsxedit

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestInspectSample 真实样表（LibreOffice 存过、带表级打印标题 / 打印区域 / 筛选
// 定义名）：打开性、定义名与表级归属、公式与缓存值都读得准。
func TestInspectSample(t *testing.T) {
	sample := filepath.Join("..", "..", "gaea", "skill", "assets", "cost-estimate-sample.xlsx")
	h, err := Inspect(sample)
	if err != nil {
		t.Fatalf("样表应可体检: %v", err)
	}
	if len(h.Sheets) != 6 || h.Sheets[0] != "封面" || h.Sheets[3] != "综合单价" {
		t.Fatalf("六张表读错：%v", h.Sheets)
	}
	if h.DefinedNames != 15 || h.SheetScopedNames != 15 {
		t.Errorf("样表有 15 个定义名且都是表级的：得到 %d（表级 %d）", h.DefinedNames, h.SheetScopedNames)
	}
	if h.Formulas < 1400 {
		t.Errorf("样表约 1405 个公式：得到 %d", h.Formulas)
	}
	// 样表重算过：未用槽位的取价公式是 t="str" 的空串结果，不算缺缓存值。
	if h.NoCachedValue != 0 || h.ErrorValues != 0 {
		t.Errorf("重算过的样表不应有缺缓存值 / 错误值：%d / %d", h.NoCachedValue, h.ErrorValues)
	}
	if h.DataValidations != 2 || h.Merges == 0 {
		t.Errorf("样表有 2 条数据验证、若干合并格：得到 %d / %d", h.DataValidations, h.Merges)
	}
	line := h.StructureLine()
	for _, want := range []string{"6 表", "定义名 15（表级 15）", "数据验证 2"} {
		if !strings.Contains(line, want) {
			t.Errorf("结构摘要应含 %q：%s", want, line)
		}
	}
	if diffs := StructureDiff(h, h); len(diffs) != 0 {
		t.Errorf("自己对自己不应有差异：%v", diffs)
	}
}

// TestInspectDetectsMissingCache excelize 写出的公式没有缓存值——「合法但退化」，
// 用户打开是空值；体检要点出来。
func TestInspectDetectsMissingCache(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", 2)
	_ = f.SetCellValue(sheet, "A2", 3)
	_ = f.SetCellFormula(sheet, "A3", "A1*A2")
	path := filepath.Join(t.TempDir(), "fresh.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	h, err := Inspect(path)
	if err != nil {
		t.Fatalf("应可体检: %v", err)
	}
	if h.Formulas != 1 || h.NoCachedValue != 1 {
		t.Fatalf("excelize 写出的公式格应判为缺缓存值：%+v", h)
	}
	if !strings.Contains(h.CacheLine(), "公式 1 格，其中 1 格没有缓存值") {
		t.Errorf("缓存值摘要：%s", h.CacheLine())
	}
	after := h
	after.DefinedNames, after.SheetScopedNames = 3, 2
	after.Sheets = append([]string{"封面"}, after.Sheets...)
	diffs := StructureDiff(h, after)
	joined := strings.Join(diffs, "；")
	for _, want := range []string{"工作表数 1 → 2", "定义名 0 → 3", "工作表级归属 0 → 2"} {
		if !strings.Contains(joined, want) {
			t.Errorf("结构对账应含 %q：%s", want, joined)
		}
	}
}

// TestInspectNotOpenable 打开性探测：zip 坏了 / 工作表 XML 坏了都报 ErrNotOpenable。
func TestInspectNotOpenable(t *testing.T) {
	junk := filepath.Join(t.TempDir(), "junk.xlsx")
	if err := os.WriteFile(junk, []byte("not a zip at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(junk); !errors.Is(err, ErrNotOpenable) {
		t.Fatalf("非 zip 应报 ErrNotOpenable：%v", err)
	}
	// 合法 zip、坏工作表 XML。
	bad := filepath.Join(t.TempDir(), "bad.xlsx")
	zw, err := os.Create(bad)
	if err != nil {
		t.Fatal(err)
	}
	w, finish := newZipWriter(t, zw)
	w("xl/workbook.xml", `<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="S1" r:id="rId1"/></sheets></workbook>`)
	w("xl/_rels/workbook.xml.rels", `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`)
	w("xl/worksheets/sheet1.xml", `<?xml version="1.0"?><worksheet><sheetData><row r="1"><c r="A1"><f>A2</f><v>1</v></c>`) // 没闭合
	finish()
	zw.Close()
	if _, err := Inspect(bad); !errors.Is(err, ErrNotOpenable) || !strings.Contains(err.Error(), "工作表 XML") {
		t.Fatalf("坏 XML 应报 ErrNotOpenable 并点名工作表 XML：%v", err)
	}
}

// newZipWriter 往 zw 里逐件写 zip 条目；第二个返回值写中央目录收尾。
func newZipWriter(t *testing.T, zw *os.File) (func(name, body string), func()) {
	t.Helper()
	z := zip.NewWriter(zw)
	return func(name, body string) {
			w, err := z.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}, func() {
			if err := z.Close(); err != nil {
				t.Fatal(err)
			}
		}
}
