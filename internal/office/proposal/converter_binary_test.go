package proposal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLooksBinary(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"%PDF-1.4\n1 0 obj\n<<\n", true},
		{"hello\nworld\n", false},
		{"text\x00with-nul", true},
		{"# 招标文件\n\n> 来源：a.pdf | 共 3 页\n\n", false},
	}
	for _, c := range cases {
		if got := looksBinary(c.in); got != c.want {
			t.Errorf("looksBinary(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestConvertToMarkdownRejectsRawBytes 验证转换结果若仍是原始文件字节
// （历史版本把 PDF 原始字节当 markdown 入库），必须报错而不是入库。
func TestConvertToMarkdownRejectsRawBytes(t *testing.T) {
	dir := t.TempDir()
	// 历史 bug 场景：文件内容就是 PDF 原始字节，却被当作文本转换结果。
	p := filepath.Join(dir, "伪装成文本.txt")
	if err := os.WriteFile(p, []byte("%PDF-1.4\n1 0 obj\n<<\n/Title (x)\n>>\nendobj\n%%EOF\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	md, err := ConvertToMarkdown(p)
	if err == nil {
		t.Fatalf("期望二进制输出被拒绝，却返回 markdown %q", md)
	}
	if !strings.Contains(err.Error(), "原始文件字节") {
		t.Errorf("错误信息不清晰: %v", err)
	}
}
