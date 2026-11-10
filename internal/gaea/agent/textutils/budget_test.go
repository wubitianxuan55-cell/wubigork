package textutils

// D1 输出预算三件套测试（dsh-ops 蒸馏 2026-11-10 第二波）。textutils 此前
// 零测试，本文件一并钉 TruncateToolOutputWith 基础契约。

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEscapeJSONLenMatchesEncodingJSON(t *testing.T) {
	// 性质测试：与 encoding/json（SetEscapeHTML(false)）实际编码长度逐例相等。
	cases := []string{
		"", "plain ascii", `quotes "inside" and \backslash`,
		"中文与English混排", "line1\nline2\r\nline3\ttab",
		"ctl\x00\x01\x1f\x7f", "emoji 🎉emoji", strings.Repeat("x", 1000),
	}
	for _, s := range cases {
		var b strings.Builder
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(s); err != nil {
			t.Fatal(err)
		}
		// Encode 产出 `"...."\n`：减首尾引号与换行即字面量内容。
		encoded := strings.TrimSuffix(b.String(), "\n")
		want := len(encoded) - 2
		if got := EscapeJSONLen(s); got != want {
			t.Errorf("EscapeJSONLen(%q)=%d want %d", s, got, want)
		}
	}
}

func TestTruncateToEscapedBudgetConverges(t *testing.T) {
	// 引号/换行密集且内容多变的 payload（同文案会被 Normalize 折叠成一行）：
	// 转义后远超预算，迭代收敛后必须 ≤ 预算。
	var b strings.Builder
	for i := 0; i < 3000; i++ {
		b.WriteString(`line ` + itoaTest(i) + ` "quoted" with \\ stuff` + "\n")
	}
	const budget = 8 * 1024
	got, _ := TruncateToEscapedBudget(b.String(), 320, budget)
	if el := EscapeJSONLen(got); el > budget {
		t.Fatalf("转义后 %d 仍超预算 %d", el, budget)
	}
	if !strings.Contains(got, "elided") {
		t.Fatalf("截断标记缺失: %q", got[:min(200, len(got))])
	}
	// 控制符病态输入（\x00 转义 ×6，内容多变防折叠）：仍要收敛。
	var ctl strings.Builder
	for i := 0; i < 3000; i++ {
		ctl.WriteString("a" + itoaTest(i) + "\x00b\n")
	}
	got2, _ := TruncateToEscapedBudget(ctl.String(), 320, budget)
	if el := EscapeJSONLen(got2); el > budget+64 { // 4 轮未全收敛允许小超差（backstop 兜底）
		t.Fatalf("控制符病态输入超差过大: %d > %d", el, budget+64)
	}
}

func TestTruncateToolOutputPagedKeepsHeadAndMarker(t *testing.T) {
	// 499 行分页读+尾行续读标记：保头 319 行+尾标记，中段丢弃——模型按尾行
	// offset 续页不再静默跳过被砍中段（head+tail 旧口径缺陷）。
	var b strings.Builder
	for i := 1; i <= 499; i++ {
		b.WriteString("row" + itoaTest(i) + "\n")
	}
	b.WriteString("[more lines available; pass offset=2000 to continue]")
	out, _ := TruncateToolOutputPaged(b.String(), 320, 32*1024)
	lines := strings.Split(out, "\n")
	if len(lines) > 322 { // 319 头行+续读标记+截断标记
		t.Fatalf("行数超帽: %d", len(lines))
	}
	if !strings.Contains(out, "pass offset=2000") {
		t.Fatal("续读标记尾行必须存活")
	}
	if !strings.HasPrefix(lines[0], "row1") || !strings.Contains(out, "row2\n") {
		t.Fatal("头部行必须保留")
	}
	// 中段（第 320~498 行）确实被省。
	if strings.Contains(out, "row400\n") || strings.Contains(out, "row498\n") {
		t.Fatal("中段应被省略（保头方向）")
	}
}

func TestTruncateToolOutputBasicContract(t *testing.T) {
	// 既有口径钉死：帽内原样返回；超帽给 elided 标记（内容多变防 Normalize 折叠）。
	small := "one\ntwo\nthree"
	if got, notice := TruncateToolOutput(small); got != small || notice != "" {
		t.Fatalf("帽内应原样: %q %q", got, notice)
	}
	var big strings.Builder
	for i := 0; i < 1000; i++ {
		big.WriteString("error line " + itoaTest(i) + " that matters for signal weighting\n")
	}
	got, _ := TruncateToolOutput(big.String())
	if !strings.Contains(got, "elided") {
		t.Fatal("超帽应有 elided 标记")
	}
}

func itoaTest(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}
