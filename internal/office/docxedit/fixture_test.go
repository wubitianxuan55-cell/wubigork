package docxedit

import (
	"os"
	"testing"
)

// TestWriteRevisionFixture 输出一个含 w:del/w:ins 的最小 docx 到 GAEA_FIXTURE_OUT，
// 供前端渲染验证用（默认跳过）。
func TestWriteRevisionFixture(t *testing.T) {
	out := os.Getenv("GAEA_FIXTURE_OUT")
	if out == "" {
		t.Skip("未设置 GAEA_FIXTURE_OUT")
	}
	body := `
    <w:p><w:r><w:t>合同期限为 </w:t></w:r><w:del w:id="1" w:author="gaea AI" w:date="2026-08-09T00:00:00Z"><w:r><w:delText>30 天</w:delText></w:r></w:del><w:ins w:id="2" w:author="gaea AI" w:date="2026-08-09T00:00:00Z"><w:r><w:t>60 天</w:t></w:r></w:ins><w:r><w:t>。</w:t></w:r></w:p>`
	path := buildDocx(t, body)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
