package builtin

import (
	"context"
	"os"
	"strings"
	"testing"
)

// bg 返回 Execute 的上下文（evidence 台账缺位即静默跳过，本文件无需台账）。
func bg() context.Context { return context.Background() }

// mustRead 读文件并在出错时终止测试。
func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// T1 三档行匹配兜底的内核用例（codex 蒸馏）：精确整串失配后按上下文定位。

// 施工单验收例：文件被追加无关行后，edit_file 仍成功且结果内容正确。
func TestEditFileSeekAfterAppend(t *testing.T) {
	dir := t.TempDir()
	original := "header\nalpha\nbeta\ngamma\nfooter\n"
	path := writeTemp(t, dir, "app.txt", original)
	// 模拟外部改动：追加无关行，块内容本身没变——逐字节替换失配，三档兜底应命中。
	appended := original + "unrelated extra line\n"
	if err := os.WriteFile(path, []byte(appended), 0o644); err != nil {
		t.Fatalf("append: %v", err)
	}
	out := execTool(t, editFile{}, map[string]any{
		"path": path, "old_string": "alpha\nbeta\ngamma", "new_string": "alpha2\nbeta2\ngamma2",
	})
	if !strings.Contains(out, "1 occurrence(s)") {
		t.Fatalf("应命中一次: %s", out)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "header\nalpha2\nbeta2\ngamma2\nfooter\nunrelated extra line\n"
	if string(b) != want {
		t.Fatalf("拼接结果不符:\n got %q\nwant %q", string(b), want)
	}
}

// 档 1（行尾空白）命中并在结果里带定位说明。
func TestEditFileSeekTrailingWhitespaceNote(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "w.txt", "one\ntwo  \nthree\n")
	out := execTool(t, editFile{}, map[string]any{
		"path": path, "old_string": "two\nthree", "new_string": "TWO\nthree",
	})
	if !strings.Contains(out, "按第 1 档") || !strings.Contains(out, "第 2 行") {
		t.Fatalf("结果应带档位定位说明: %s", out)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "one\nTWO\nthree\n" {
		t.Fatalf("内容不符: %q", string(b))
	}
}

// 档 2（两侧空白）命中（缩进漂移场景）。
func TestEditFileSeekIndentDrift(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "i.txt", "func f() {\n\t\tbody()\n}\n")
	out := execTool(t, editFile{}, map[string]any{
		"path": path, "old_string": "    body()", "new_string": "\t\tbody2()",
	})
	if !strings.Contains(out, "按第 2 档") {
		t.Fatalf("应按档 2 命中: %s", out)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "func f() {\n\t\tbody2()\n}\n" {
		t.Fatalf("内容不符: %q", string(b))
	}
}

// 精确命中优先于兜底：能逐字节替换就绝不走行级拼接（结果不带定位说明）。
func TestEditFileExactStillPreferred(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "e.txt", "aa\nbb\n")
	out := execTool(t, editFile{}, map[string]any{
		"path": path, "old_string": "aa", "new_string": "xx",
	})
	if strings.Contains(out, "档") {
		t.Fatalf("精确命中不应带兜底说明: %s", out)
	}
	if string(mustRead(t, path)) != "xx\nbb\n" {
		t.Fatalf("内容不符: %q", mustRead(t, path))
	}
}

// 兜底多命中且未开 replace_all：拒绝并说明数量。
// （两块内容逐字节失配但空白容差下各中一次；若精确串就多命中，走的是
// 精确路径的既有报错，不进本用例。）
func TestEditFileSeekAmbiguous(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "m.txt", "foo one \nx\ny\nfoo one  \nx\nz\n")
	_, err := editFile{}.Execute(bg(), mustArgs(t, map[string]any{
		"path": path, "old_string": "foo one\nx", "new_string": "bar",
	}))
	if err == nil || !strings.Contains(err.Error(), "whitespace-tolerant") {
		t.Fatalf("多命中应拒绝: %v", err)
	}
	if string(mustRead(t, path)) != "foo one \nx\ny\nfoo one  \nx\nz\n" {
		t.Fatalf("拒绝后文件不得改动")
	}
}

// 完全未命中：错误附「最接近的候选」行号。
func TestEditFileSeekMissWithCandidates(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "c.txt", "func main() {\nbody\n}\nother stuff\nfunc mainx() {\n")
	// 目标串不在文件里（mainxx ≠ mainx），但与第 1/5 行首行高度相似。
	_, err := editFile{}.Execute(bg(), mustArgs(t, map[string]any{
		"path": path, "old_string": "func mainxx() {", "new_string": "z",
	}))
	if err == nil {
		t.Fatalf("未命中应报错")
	}
	// 既有断言意图保持：not found 前缀不动；新增候选提示（mainxx 与
	// 第 5 行 mainx 首行匹配度最高，其次第 1 行 main）。
	if !strings.Contains(err.Error(), "old_string not found") {
		t.Fatalf("错误应保留 not found 前缀: %v", err)
	}
	if !strings.Contains(err.Error(), "最接近的候选在第 5、1 行") {
		t.Fatalf("错误应附候选行号: %v", err)
	}
}

// 行尾无关（2026-11-28 黄甲反馈，取代此前「裸 \n 对 CRLF 文件仍报错」的口径）：
// 裸 \n 的多行锚点对 CRLF 文件照常命中，替换块沿用 CRLF。
func TestEditFileBareLFAnchorOnCRLFFileSucceeds(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "crlf.txt", "line one\r\nline two\r\n")
	if _, err := (editFile{}).Execute(bg(), mustArgs(t, map[string]any{
		"path": path, "old_string": "line one\nline two", "new_string": "x",
	})); err != nil {
		t.Fatalf("裸 \\n 锚点对 CRLF 文件应命中: %v", err)
	}
	if content, _, _ := readFileEncoded(path); content != "x\r\n" {
		t.Fatalf("替换块应沿用 CRLF: %q", content)
	}
}
