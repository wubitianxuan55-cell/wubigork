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
	// 两保留行按轮次均分得 [2,1]。
	got := TruncateRows(rows, 60, 10)
	for i := 2; i < 5; i++ {
		if got[i] != omittedRow {
			t.Fatalf("尾部三行应整行省略: %v", got)
		}
	}
	for i := 0; i < 2; i++ {
		if got[i] <= 0 {
			t.Fatalf("保留行应有正份额: %v", got)
		}
	}
}

// 剩余预算在描述间按轮次均分：总描述字节守恒且各行份额差 ≤1（同长名字）。
func TestTruncateRowsFairShare(t *testing.T) {
	rows := []BudgetRow{
		{Name: "a", Desc: strings.Repeat("x", 100)},
		{Name: "b", Desc: strings.Repeat("y", 100)},
		{Name: "c", Desc: strings.Repeat("z", 100)},
	}
	// 名字成本 = 3 × (2+1+1) = 12；无省略 → 描述预算 = budget - 12。
	got := TruncateRows(rows, 60, 0)
	sum := 0
	for i, k := range got {
		if k == omittedRow {
			t.Fatalf("不该省略: %v", got)
		}
		sum += k
		_ = i
	}
	if sum != 60-12 {
		t.Fatalf("描述预算总和 = %d, want %d", sum, 48)
	}
	// 均分：三行同名额，差 ≤ 1。
	if got[0]-got[2] > 1 || got[2]-got[0] > 1 {
		t.Fatalf("份额不均: %v", got)
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
