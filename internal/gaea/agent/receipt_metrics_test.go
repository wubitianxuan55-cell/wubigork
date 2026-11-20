package agent

import (
	"fmt"
	"strings"
	"testing"
)

// 实测反馈④：计量行由父侧 Go 确定性计算——chars=rune 数（wc -m 口径）、
// hanzi=CJK 汉字数。子代理自报字符数不可信（四种量纲混编的根源）。
func TestFormatReceiptMetrics(t *testing.T) {
	cases := []struct {
		in           string
		chars, hanzi int
	}{
		{"", 0, 0},
		{"abc", 3, 0},
		{"对象与范围", 5, 5},
		{"表1-4 与 图1-11", 12, 3}, // 汉字=表/与/图，其余数字/符号计入 chars
		{"café café", 9, 0},      // é 各占 1 rune
	}
	for _, tc := range cases {
		got := FormatReceiptMetrics(tc.in)
		want := fmt.Sprintf("\n[metrics] chars=%d hanzi=%d", tc.chars, tc.hanzi)
		if got != want {
			t.Fatalf("FormatReceiptMetrics(%q)=%q, want %q", tc.in, got, want)
		}
	}
}

// finalizeRun/persisted_run 拼装顺序：metrics 行在引用行之前（引用行保持
// 回执最后一行——前端 isDelegationCall 按 sa_ 引用行分派不受影响）。
func TestReceiptMetricsBeforeReference(t *testing.T) {
	m := FormatReceiptMetrics("正文")
	r := FormatSubagentReference(&SubagentRun{Ref: "sa_test123"})
	if !strings.HasPrefix(m, "\n[metrics] ") {
		t.Fatalf("metrics 行格式: %q", m)
	}
	if !strings.Contains(r, "Subagent reference: sa_test123") {
		t.Fatalf("引用行格式: %q", r)
	}
	combined := "正文" + m + r
	if strings.LastIndex(combined, "[metrics]") > strings.LastIndex(combined, "Subagent reference:") {
		t.Fatal("metrics 行应在引用行之前（引用行保持末行）")
	}
}
