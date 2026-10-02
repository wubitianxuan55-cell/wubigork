package ooxml

// ooxml 测试：①WML/DML 双方言转义字节口径钉死；②zip 读写往返保条目序
// （AP3-02 字节冻结的反向证据基座：条目序/压缩方法逐字节可断言）。

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEscapeXMLDialects(t *testing.T) {
	const in = `&<>"'南`
	if got := EscapeXML(in, true); got != `&amp;&lt;&gt;&quot;'南` {
		t.Errorf("WML 口径: got %s", got)
	}
	if got := EscapeXML(in, false); got != `&amp;&lt;&gt;&#34;&#39;南` {
		t.Errorf("DML 口径: got %s", got)
	}
}

func TestTextElementDialects(t *testing.T) {
	// WML：tab 也算首尾空白 → 补 preserve；tab 字节原样不转义
	if got, want := TextElement("w:t", "\tx", `w:rsidRPr="A"`, true),
		"<w:t w:rsidRPr=\"A\" xml:space=\"preserve\">\tx</w:t>"; got != want {
		t.Errorf("WML tab preserve: got %s", got)
	}
	// DML：tab 不算空白 → 不补 preserve；撇号/双引号按 &#34;/&#39; 转义
	if got, want := TextElement("a:t", "\tx", `lang="zh-CN"`, false),
		"<a:t lang=\"zh-CN\">\tx</a:t>"; got != want {
		t.Errorf("DML tab 不 preserve: got %s", got)
	}
	// DML 空格仍算首尾空白
	if got, want := TextElement("a:t", " x \"s\" 'q' ", "", false),
		`<a:t xml:space="preserve"> x &#34;s&#34; &#39;q&#39; </a:t>`; got != want {
		t.Errorf("DML 空格 preserve+转义: got %s", got)
	}
	// WML 双引号 &quot; 口径
	if got, want := TextElement("w:t", `说"话"`, "", true),
		`<w:t>说&quot;话&quot;</w:t>`; got != want {
		t.Errorf("WML 引号转义: got %s", got)
	}
	// 已带 xml:space 不重复补；attrs 为空不加空格
	if got := TextElement("w:t", " x ", `xml:space="preserve"`, true); got != `<w:t xml:space="preserve"> x </w:t>` {
		t.Errorf("已有 preserve: got %s", got)
	}
	if got := TextElement("w:t", "中", "", true); got != "<w:t>中</w:t>" {
		t.Errorf("无 attrs: got %s", got)
	}
}

func buildZip(t *testing.T, names []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "src.bin")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("body:" + n)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadZipWriteAtomicRoundTrip(t *testing.T) {
	src := buildZip(t, []string{"c-part.xml", "a-part.xml", "b-part.xml"})
	files, order, err := ReadZip(src, "docx")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(files); got != 3 {
		t.Fatalf("条目数=%d want 3", got)
	}
	wantOrder := []string{"c-part.xml", "a-part.xml", "b-part.xml"} // 原始条目序
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("条目序丢失: %v", order)
		}
	}
	// 改写其中一个 part，其余字节不动
	files["a-part.xml"] = []byte("body:a-part.xml:patched")

	dst := filepath.Join(t.TempDir(), "dst.bin")
	if err := WriteAtomic(dst, files, order, ".gaea-ooxml-*.bin", "docx"); err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if len(r.File) != 3 {
		t.Fatalf("写回条目数=%d", len(r.File))
	}
	for i, f := range r.File {
		if f.Name != wantOrder[i] {
			t.Errorf("写回条目序[%d]=%s want %s", i, f.Name, wantOrder[i])
		}
		if f.Method != zip.Deflate {
			t.Errorf("写回压缩方法=%d want Deflate", f.Method)
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		want := "body:" + f.Name
		if f.Name == "a-part.xml" {
			want = "body:a-part.xml:patched"
		}
		if string(got) != want {
			t.Errorf("条目 %s 内容=%q want %q", f.Name, got, want)
		}
	}
}

func TestWriteAtomicErrorLabel(t *testing.T) {
	// 目标目录不存在 → CreateTemp 失败，错误前缀原文
	files := map[string][]byte{"x": []byte("x")}
	err := WriteAtomic(filepath.Join(t.TempDir(), "no-such-dir", "f.bin"), files, []string{"x"}, ".gaea-ooxml-*.bin", "docx")
	if err == nil || !strings.Contains(err.Error(), "创建临时文件失败") {
		t.Errorf("got %v", err)
	}
}
