package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// F2 doc_render 只读渲染工具：参数面全量断言 + 真实渲染走门控（本机须有
// soffice+pdftoppm——门控纪律与 OCR/视觉 diff 用例同源，缺件跳过不假绿）。

func TestDocRender_ArgValidation(t *testing.T) {
	d := docRender{workDir: t.TempDir()}
	cases := []struct {
		name string
		args string
		want string // 错误消息应包含的子串
	}{
		{"不支持格式", `{"path":"a.txt"}`, "不支持的格式"},
		{"文件不存在", `{"path":"nope.docx"}`, "读取文档失败"},
		{"逗号页区间拒绝", `{"path":"a.pdf"}`, "pages"}, // 文件不存在先报，先造文件见下
	}
	for _, tc := range cases {
		if tc.name == "逗号页区间拒绝" {
			continue // 下一组覆盖（需真实文件走通到 pages 校验）
		}
		t.Run(tc.name, func(t *testing.T) {
			_, err := d.Execute(context.Background(), json.RawMessage(tc.args))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("期望错误含 %q，实得 %v", tc.want, err)
			}
		})
	}
}

func TestDocRender_PageSpanReject(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.pdf")
	if err := writeMinimalPDF(src); err != nil {
		t.Skipf("构造最小 PDF 失败: %v", err)
	}
	d := docRender{workDir: dir}
	_, err := d.Execute(context.Background(), json.RawMessage(`{"path":"a.pdf","pages":"1,3"}`))
	if err == nil || !strings.Contains(err.Error(), "不支持逗号列表") {
		t.Fatalf("逗号区间应拒绝，实得 %v", err)
	}
	if _, err := d.Execute(context.Background(), json.RawMessage(`{"path":"a.pdf","pages":"5-2"}`)); err == nil {
		t.Fatal("倒置区间应拒绝")
	}
}

// writeMinimalPDF 造一个内容极小但结构合法的 PDF（页渲染由门控用例覆盖，
// 这里只到参数校验层）。
func writeMinimalPDF(path string) error {
	body := "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF"
	return os.WriteFile(path, []byte(body), 0o644)
}

// TestDocRender_ReadOnlyAndReadTarget 契约面：只读 + 读目标解析。
func TestDocRender_ReadOnlyAndReadTarget(t *testing.T) {
	d := docRender{workDir: `W:\ws`}
	if !d.ReadOnly() {
		t.Fatal("doc_render 必须只读（F2 存在理由）")
	}
	tgt, ok := d.ReadTarget(json.RawMessage(`{"path":"docs/a.docx"}`))
	if !ok || !strings.Contains(filepath.ToSlash(tgt), "docs/") {
		t.Fatalf("ReadTarget=%q ok=%v", tgt, ok)
	}
	if _, ok := d.ReadTarget(json.RawMessage(`{}`)); ok {
		t.Fatal("空参不应有读目标")
	}
}
