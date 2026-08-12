package docmd

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestConvertPptxViaMarkItDown 验证 pptx → Markdown（markitdown 后端）。
// markitdown 或 python-pptx 不可用时跳过，不阻塞无 Python 的 CI。
func TestConvertPptxViaMarkItDown(t *testing.T) {
	if !markitdownAvailable() {
		t.Skip("markitdown 不可用")
	}
	dir := t.TempDir()
	pptx := filepath.Join(dir, "deck.pptx")
	script := `
from pptx import Presentation
from pptx.util import Inches
p = Presentation()
slide = p.slides.add_slide(p.slide_layouts[1])
slide.shapes.title.text = "季度汇报"
slide.placeholders[1].text = "完成 A/B/C 三项"
p.save(r"` + pptx + `")
`
	if out, err := exec.Command("python", "-c", script).CombinedOutput(); err != nil {
		t.Skipf("python-pptx 不可用，跳过: %v %s", err, out)
	}
	md, err := Convert(pptx, "")
	if err != nil {
		t.Fatalf("Convert(pptx): %v", err)
	}
	for _, want := range []string{"季度汇报", "完成 A/B/C 三项"} {
		if !strings.Contains(md, want) {
			t.Errorf("pptx 转换缺少 %q，输出: %s", want, md)
		}
	}
}

// TestConvertDocxFallbackToBuiltin 验证 markitdown 不可用时 docx 回退内置解析器。
func TestConvertDocxFallbackToBuiltin(t *testing.T) {
	path := buildDocxWithTable(t)
	md, err := Convert(path, "")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	for _, want := range []string{"项目周报", "| 指标 | 本周 | 上周 |"} {
		if !strings.Contains(md, want) {
			t.Errorf("docx 转换缺少 %q，输出: %s", want, md)
		}
	}
}
