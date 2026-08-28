package control

import "testing"

func TestSetModeRoundTrip(t *testing.T) {
	c := New(Options{})
	if got := c.Mode(); got != "default" {
		t.Fatalf("default mode = %q, want %q", got, "default")
	}
	c.SetMode("plan")
	if got := c.Mode(); got != "plan" {
		t.Fatalf("after SetMode(plan) = %q, want %q", got, "plan")
	}
	// Unknown values fall back to default.
	c.SetMode("yolo")
	if got := c.Mode(); got != "default" {
		t.Fatalf("invalid mode should fall back to default, got %q", got)
	}
}

func TestShouldPlanModeSemantics(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		autoPlan bool
		input    string
		want     bool
	}{
		{"default off simple", "default", false, "帮我查一下资料", false},
		{"default off complex", "default", false, longInput, false},
		{"default on simple", "default", true, "帮我查一下资料", false},
		{"default on complex", "default", true, longInput, true},
		{"plan forces simple", "plan", false, "帮我查一下资料", true},
		{"plan forces complex", "plan", false, longInput, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(Options{AutoPlan: tt.autoPlan})
			c.mu.Lock()
			c.mode = tt.mode
			c.mu.Unlock()
			if got := c.shouldPlan(tt.input); got != tt.want {
				t.Fatalf("shouldPlan(%q) mode=%s autoPlan=%v = %v, want %v",
					tt.input, tt.mode, tt.autoPlan, got, tt.want)
			}
		})
	}
}

// longInput 超过 IsSimpleQuery 的 100 rune 上限，按启发式视为复杂任务。
const longInput = "请准备下个季度的成本测算表格，把所有供应商的报价按月份汇总成对比图表，" +
	"附上价差原因分析，并输出 docx 交付物保存到工作区。" +
	"这是一段超过一百个字符的长输入以确保它不被视为简单查询。"
