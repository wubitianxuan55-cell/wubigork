package xlsxedit

// health.go — xlsx 健康体检与结构对账（2026-11-27 AI 反馈 #7 / #15③）。
//
// 两类「合法但退化」的写法此前都没有任何预警：
//   - openpyxl / excelize 保存会丢掉全部公式缓存值——用户打开是满表空值；
//   - LibreOffice 转存（含 xlsx_recalc 自己）会重写整包 XML，可能剥掉定义名的
//     工作表级归属（Print_Area / Print_Titles 变成簿级）、丢数据有效性。
//
// Inspect 只读、不拉起任何外部进程：zip 结构 + 工作簿/工作表 XML 是否完好
// （打开性探测），公式格里有多少没有可用缓存值，以及最容易被转存悄悄改掉的
// 结构面（定义名及其表级归属、数据验证、条件格式、合并格）。调用方拿它做三件事：
// xlsx_audit 的「健康体检」行、xlsx_recalc 前后的「结构对账」、bash 写出 xlsx 后的
// 缓存值提示。

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ErrNotOpenable 文件打不开：zip 结构坏了，或工作簿/工作表 XML 不完好。
var ErrNotOpenable = errors.New("不是能打开的 .xlsx")

// Health 一份工作簿的健康体检结果。
type Health struct {
	Sheets           []string `json:"sheets"`
	DefinedNames     int      `json:"definedNames"`     // 定义名总数
	SheetScopedNames int      `json:"sheetScopedNames"` // 其中有工作表级归属的（打印标题 / 打印区域 / 筛选范围）
	DataValidations  int      `json:"dataValidations"`  // 数据有效性规则条数
	CondFormats      int      `json:"condFormats"`      // 条件格式规则条数
	Merges           int      `json:"merges"`           // 合并单元格数
	Formulas         int      `json:"formulas"`         // 公式格数
	NoCachedValue    int      `json:"noCachedValue"`    // 公式格里没有可用缓存值的（没重算；t="str" 的空串结果不算）
	ErrorValues      int      `json:"errorValues"`      // 缓存值是错误值（#REF! / #DIV/0! …）
}

// Inspect 对 .xlsx 做只读体检。zip 或 XML 坏了返回 ErrNotOpenable 包裹的错误。
func Inspect(path string) (Health, error) {
	var h Health
	zr, err := zip.OpenReader(path)
	if err != nil {
		return h, fmt.Errorf("%w（%s）：%v", ErrNotOpenable, filepath.Base(path), err)
	}
	defer func() { _ = zr.Close() }()
	files := make(map[string]*zip.File, len(zr.File))
	for _, zf := range zr.File {
		files[zf.Name] = zf
	}
	// 工作簿清单：表名与定义名。
	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
		} `xml:"sheets>sheet"`
		DefinedNames []struct {
			Name       string `xml:"name,attr"`
			LocalSheet *int   `xml:"localSheetId,attr"`
		} `xml:"definedNames>definedName"`
	}
	if err := readPartXML(files, "xl/workbook.xml", &wb); err != nil {
		return h, fmt.Errorf("%w：%v", ErrNotOpenable, err)
	}
	for _, s := range wb.Sheets {
		h.Sheets = append(h.Sheets, s.Name)
	}
	for _, d := range wb.DefinedNames {
		h.DefinedNames++
		if d.LocalSheet != nil {
			h.SheetScopedNames++
		}
	}
	parts, err := sheetParts(&zr.Reader)
	if err != nil {
		return h, fmt.Errorf("%w：%v", ErrNotOpenable, err)
	}
	for _, part := range parts {
		zf, ok := files[part]
		if !ok {
			continue
		}
		rc, oerr := zf.Open()
		if oerr != nil {
			return h, fmt.Errorf("%w：%v", ErrNotOpenable, oerr)
		}
		sh, serr := scanSheetHealth(rc)
		_ = rc.Close()
		if serr != nil {
			return h, fmt.Errorf("%w：工作表 XML（%s）解析失败——%v", ErrNotOpenable, part, serr)
		}
		h.Formulas += sh.formulas
		h.NoCachedValue += sh.noCached
		h.ErrorValues += sh.errors
		h.DataValidations += sh.validations
		h.CondFormats += sh.condFormats
		h.Merges += sh.merges
	}
	return h, nil
}

func readPartXML(files map[string]*zip.File, name string, into any) error {
	zf, ok := files[name]
	if !ok {
		return fmt.Errorf("包里没有 %s", name)
	}
	rc, err := zf.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	return xml.NewDecoder(rc).Decode(into)
}

// sheetHealth 一张工作表的体检计数。
type sheetHealth struct {
	formulas, noCached, errors, validations, condFormats, merges int
}

// scanSheetHealth 流式读一张工作表 XML：公式格与缓存值形态、数据验证、条件
// 格式、合并格。XML 不完好在 Token() 处报错。
func scanSheetHealth(r io.Reader) (sheetHealth, error) {
	var sh sheetHealth
	dec := xml.NewDecoder(r)
	var (
		inCell, hasF, inV bool // 在 <c> 内 / 该格有 <f> / 当前在 <v> 内
		sawV              bool // 该格出现过 <v> 元素（内容可为空）
		ctype, vText      string
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sh, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case t.Name.Local == "c" && !inCell:
				inCell, hasF, sawV, vText, ctype = true, false, false, "", ""
				for _, a := range t.Attr {
					if a.Name.Local == "t" {
						ctype = a.Value
					}
				}
			case inCell && t.Name.Local == "f":
				hasF = true
			case inCell && t.Name.Local == "v":
				inV, sawV = true, true
			case !inCell && t.Name.Local == "dataValidation":
				sh.validations++
			case !inCell && t.Name.Local == "conditionalFormatting":
				sh.condFormats++
			case !inCell && t.Name.Local == "mergeCell":
				sh.merges++
			}
		case xml.CharData:
			if inV {
				vText += string(t)
			}
		case xml.EndElement:
			switch {
			case t.Name.Local == "v" && inCell:
				inV = false
			case t.Name.Local == "c" && inCell:
				if hasF {
					sh.formulas++
					switch {
					case ctype == "e":
						sh.errors++
					case strings.TrimSpace(vText) == "" && !(ctype == "str" && sawV):
						// 没有可用缓存值：openpyxl / excelize 写出的未重算公式格。
						// t="str" 且有 <v>（内容为空）= 重算过、结果就是空串，不算。
						sh.noCached++
					}
				}
				inCell = false
			}
		}
	}
	return sh, nil
}

// StructureLine 结构面一行摘要。
func (h Health) StructureLine() string {
	return fmt.Sprintf("%d 表｜定义名 %d（表级 %d）｜数据验证 %d｜条件格式 %d｜合并格 %d",
		len(h.Sheets), h.DefinedNames, h.SheetScopedNames, h.DataValidations, h.CondFormats, h.Merges)
}

// CacheLine 缓存值一行摘要。
func (h Health) CacheLine() string {
	s := fmt.Sprintf("公式 %d 格", h.Formulas)
	if h.NoCachedValue > 0 {
		s += fmt.Sprintf("，其中 %d 格没有缓存值", h.NoCachedValue)
	}
	if h.ErrorValues > 0 {
		s += fmt.Sprintf("，%d 格缓存值是错误值", h.ErrorValues)
	}
	return s
}

// StructureDiff 结构对账：重算/转存前后哪些结构面变了。空 = 没变。
func StructureDiff(before, after Health) []string {
	var out []string
	add := func(format string, a ...any) { out = append(out, fmt.Sprintf(format, a...)) }
	if len(after.Sheets) != len(before.Sheets) {
		add("工作表数 %d → %d", len(before.Sheets), len(after.Sheets))
	} else {
		for i := range before.Sheets {
			if before.Sheets[i] != after.Sheets[i] {
				add("工作表名「%s」→「%s」", before.Sheets[i], after.Sheets[i])
			}
		}
	}
	if after.DefinedNames != before.DefinedNames {
		add("定义名 %d → %d", before.DefinedNames, after.DefinedNames)
	}
	if after.SheetScopedNames != before.SheetScopedNames {
		add("定义名的工作表级归属 %d → %d（打印标题 / 打印区域 / 筛选范围归哪张表；丢了在 Excel/WPS 里失效）", before.SheetScopedNames, after.SheetScopedNames)
	}
	if after.DataValidations != before.DataValidations {
		add("数据有效性 %d → %d", before.DataValidations, after.DataValidations)
	}
	if after.CondFormats != before.CondFormats {
		add("条件格式 %d → %d", before.CondFormats, after.CondFormats)
	}
	if after.Merges != before.Merges {
		add("合并单元格 %d → %d", before.Merges, after.Merges)
	}
	return out
}
