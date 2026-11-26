package app

import (
	"encoding/json"
	"strings"
	"testing"
)

// 2026-11-26 造价反馈（旺平 #1）：模型把 write_range 发成 "wrİte_range"（İ=带点
// 大写 I），helper 按序数比对报 Unknown Excel operation，四次全废、连只写一格
// 的小负载也中招。这是模型输出侧的怪癖——Go 侧下发前归一，并在回执里报出改写。
func TestNormalizeOpName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"wrİte_range", "write_range"}, // 实测形态
		{"  Write_Range ", "write_range"},
		{"write_range", "write_range"}, // 已规范的原样
		{"set_fórmula", "set_formula"}, // 拉丁附加符
		{"", ""},
		{"写范围", "写范围"}, // 非 ASCII 不动
	}
	for _, tc := range cases {
		if got := normalizeOpName(tc.in); got != tc.want {
			t.Errorf("normalizeOpName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeOpNames(t *testing.T) {
	ops := []json.RawMessage{
		json.RawMessage(`{"op": "wrİte_range", "range": "封面!A16", "values": [["x"]]}`),
		json.RawMessage(`{"op":"write_range","range":"B2","values":[[1],[2]]}`),
	}
	out, notes := normalizeOpNames(ops)
	if len(notes) != 1 || !strings.Contains(notes[0], "→write_range") {
		t.Fatalf("应只报出被改写的一条，实得 %v", notes)
	}
	// 下发文本必须是合法 JSON 数组，且操作名已归一、其余字段原样。
	var sent []map[string]any
	if err := json.Unmarshal([]byte(out), &sent); err != nil {
		t.Fatalf("下发文本不是合法 JSON: %v\n%s", err, out)
	}
	if len(sent) != 2 || sent[0]["op"] != "write_range" || sent[0]["range"] != "封面!A16" {
		t.Fatalf("首条应归一并保留其余字段，实得 %v", sent[0])
	}
	if sent[1]["op"] != "write_range" || sent[1]["values"] == nil {
		t.Fatalf("未改写的 op 应原样透传，实得 %v", sent[1])
	}
	// 没有改写时逐字节等于原 ops 拼接（不惊动正常调用）。
	same, none := normalizeOpNames(ops[1:])
	if len(none) != 0 || same != `[{"op":"write_range","range":"B2","values":[[1],[2]]}]` {
		t.Fatalf("无改写时应原样透传，实得 %s / %v", same, none)
	}
}
