package app

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 2026-11-21 反馈 P1-5 回归钉：pageBreakBefore 必须按 w:val 语义读——
// OOXML 里元素缺省=开，w:val="0"/"false"/"off"=关；此前按元素出现即读
// 「另起页=是」，val="0"（语义=不另起页）被反向读成另起页，工具驱动了
// 错误的 XML 写法。
func TestLongdocVerifyReadStylesPageBreakVal(t *testing.T) {
	dir := t.TempDir()
	n := 0
	mk := func(tag string) string {
		n++
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		f, _ := zw.Create("word/styles.xml")
		f.Write([]byte(`<w:styles><w:style w:styleId="3"><w:name w:val="heading 1"/>` + tag + `</w:style></w:styles>`))
		_ = zw.Close()
		p := filepath.Join(dir, fmt.Sprintf("s_%d.docx", n))
		if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := []struct {
		name string
		tag  string
		want string
	}{
		{"无 val 缺省=另起页", `<w:pPr><w:pageBreakBefore/></w:pPr>`, "另起页=是"},
		{"val=0 显式不另起页", `<w:pPr><w:pageBreakBefore w:val="0"/></w:pPr>`, "另起页=否"},
		{"val=false 不另起页", `<w:pPr><w:pageBreakBefore w:val="false"/></w:pPr>`, "另起页=否"},
		{"val=1 显式另起页", `<w:pPr><w:pageBreakBefore w:val="1"/></w:pPr>`, "另起页=是"},
		{"无元素=不另起页", `<w:pPr></w:pPr>`, "另起页=否"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := ldvReadStyles(mk(tc.tag))
			if len(lines) != 1 {
				t.Fatalf("应读出 1 个 heading 样式: %v", lines)
			}
			if !strings.Contains(lines[0], tc.want) {
				t.Fatalf("读数应含 %q，实得: %s", tc.want, lines[0])
			}
		})
	}
}
