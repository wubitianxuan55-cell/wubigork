package tool

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// T5（codex 蒸馏）验收：名字总超预算 / 描述均分 / UTF-8 中文截断不切字符 /
// 同输入两次调用结果逐字节相同。

// 名字总和超预算：从尾部整行省略，占位行吃预算，剩余给保留行描述。
func TestTruncateRowsNameOverflow(t *testing.T) {
	long := strings.Repeat("n", 15) // 名字行成本 = 2+15+1 = 18
	rows := make([]BudgetRow, 5)
	for i := range rows {
		rows[i] = BudgetRow{Name: long, Desc: "x"}
	}
	// 名字总额 90 > 可用（预算 60 - 预留 10 = 50）：3 行=54 仍超 → 从尾部
	// 省略 3 行（2 行=36 ≤ 50）。剩余 = 60 - 36 - 占位行(21 字节) = 3 →
	// 字符轮转：行0 首字符 1+2(": ") = 3 恰好吃满，行1 得 0。
	got := TruncateRows(rows, 60, 10)
	for i := 2; i < 5; i++ {
		if got[i] != omittedRow {
			t.Fatalf("尾部三行应整行省略: %v", got)
		}
	}
	if got[0] != 1 || got[1] != 0 {
		t.Fatalf("保留行份额应 [1 0]: %v", got)
	}
}

// 剩余预算在描述间按字符轮转均分：每轮每行 1 字符（首字符加计 ": " 2 字节），
// 同长 ASCII 描述三行各得相等份额，分隔成本计入预算但不计入保留字节数。
func TestTruncateRowsFairShare(t *testing.T) {
	rows := []BudgetRow{
		{Name: "a", Desc: strings.Repeat("x", 100)},
		{Name: "b", Desc: strings.Repeat("y", 100)},
		{Name: "c", Desc: strings.Repeat("z", 100)},
	}
	// 名字成本 = 3 × (2+1+1) = 12；无省略 → 描述预算 = budget - 12 = 48。
	// 首轮各领 1 字符（1+2），后 39 字节轮转 13 轮 → 每行 14 字节。
	got := TruncateRows(rows, 60, 0)
	sum := 0
	for _, k := range got {
		if k == omittedRow {
			t.Fatalf("不该省略: %v", got)
		}
		sum += k
	}
	if sum != 14*3 {
		t.Fatalf("描述份额总和 = %d, want %d", sum, 42)
	}
	if got[0] != 14 || got[1] != 14 || got[2] != 14 {
		t.Fatalf("份额应均等 [14 14 14]: %v", got)
	}
	if sum+len(": ")*3 != 60-12 {
		t.Fatalf("预算守恒（含分隔成本）: %d + 6 != %d", sum, 48)
	}
}

// 字符轮转不切 CJK：中文描述按 3 字节/rune 领取，retained 字节数始终落在
// 字符边界上（下游 TruncateDescBytes 不再需要二次回退）。
func TestTruncateRowsRuneAligned(t *testing.T) {
	rows := []BudgetRow{
		{Name: "a", Desc: strings.Repeat("工", 10)},
		{Name: "b", Desc: strings.Repeat("乙", 10)},
	}
	got := TruncateRows(rows, 24, 0)
	for i, k := range got {
		if k <= 0 || k > 24 {
			t.Fatalf("行 %d 份额异常: %v", i, got)
		}
		if k%3 != 0 {
			t.Fatalf("份额应落在 rune 边界（3 的倍数）: %v", got)
		}
	}
	// 轮转公平：两行 3 字节/rune 同节奏，份额差 ≤ 3。
	if got[0]-got[1] > 3 || got[1]-got[0] > 3 {
		t.Fatalf("CJK 份额不均: %v", got)
	}
}

// UTF-8 中文截断不切字符，且追加 "..." 后仍在预算内。
func TestTruncateDescBytesRespectsRuneBoundary(t *testing.T) {
	desc := strings.Repeat("工", 100) // 每 rune 3 字节
	out := TruncateDescBytes(desc, 20)
	if len(out) > 20 {
		t.Fatalf("超预算: %d", len(out))
	}
	if !utf8.ValidString(out) {
		t.Fatalf("切出非法 UTF-8: %q", out)
	}
	if !strings.HasSuffix(out, "...") {
		t.Fatalf("缺尾标: %q", out)
	}
	if TruncateDescBytes(desc, 0) != "" || TruncateDescBytes(desc, 2) != "" {
		t.Fatalf("预算不足尾标应返回空")
	}
	if TruncateDescBytes("短", 20) != "短" {
		t.Fatalf("短描述应原样")
	}
}

// 同输入两次调用逐字节相同（可缓存、可比对）；截断后的描述合法。
func TestApplySchemaBudgetDeterministic(t *testing.T) {
	build := func() []provider.ToolSchema {
		return []provider.ToolSchema{
			{Name: "tool_a", Description: strings.Repeat("甲", 500)},
			{Name: "tool_b", Description: strings.Repeat("乙", 500)},
			{Name: "tool_c", Description: "tiny"},
			{Name: "tool_d", Description: strings.Repeat("丁", 500)},
		}
	}
	r1 := ApplySchemaBudget(build(), 512)
	r2 := ApplySchemaBudget(build(), 512)
	if len(r1) != len(r2) {
		t.Fatalf("长度不一致: %d vs %d", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].Name != r2[i].Name || r1[i].Description != r2[i].Description {
			t.Fatalf("两次结果不一致 @%d: %q/%q vs %q/%q", i,
				r1[i].Name, r1[i].Description, r2[i].Name, r2[i].Description)
		}
		if !utf8.ValidString(r1[i].Description) {
			t.Fatalf("非法 UTF-8: %q", r1[i].Description)
		}
	}
}

// 预算关闭（<=0）路径等价现状：Registry 默认不下水。
func TestRegistrySchemaBudgetOffByDefault(t *testing.T) {
	reg := NewRegistry()
	if reg.SchemaBudget() != 0 {
		t.Fatalf("默认应为关闭, got %d", reg.SchemaBudget())
	}
	// SetSchemaBudget 负值归零。
	reg.SetSchemaBudget(-5)
	if reg.SchemaBudget() != 0 {
		t.Fatalf("负值应归零")
	}
}
