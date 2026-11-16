package boot

// boot_toolgate_test.go — 板块装配闸单测（2026-11-16 图文瘦身刀）。
// 钉死 addGated 语义：filter 空=全放行（办公逐字节现状）；非空时名单内物理
// 不装、名单外照常；nil 工具安全跳过。

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

type gateProbeTool struct{ name string }

func (t gateProbeTool) Name() string        { return t.name }
func (t gateProbeTool) Description() string { return "probe" }
func (t gateProbeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}
func (t gateProbeTool) Execute(context.Context, json.RawMessage) (string, error) { return "", nil }
func (t gateProbeTool) ReadOnly() bool                                           { return true }

func TestAddGatedSemantics(t *testing.T) {
	// filter=放行谓词（true=允许，同 wenxueToolFilter 语义）：只放行 keep_me
	allowKeepMe := func(name string) bool { return name == "keep_me" }

	t.Run("filter 空=全放行", func(t *testing.T) {
		reg := tool.NewRegistry()
		addGated(reg, nil, gateProbeTool{name: "anything"})
		addGated(reg, nil, gateProbeTool{name: "other"})
		if reg.Len() != 2 {
			t.Fatalf("len = %d, want 2（办公现状语义）", reg.Len())
		}
	})

	t.Run("名单内物理不装", func(t *testing.T) {
		reg := tool.NewRegistry()
		addGated(reg, allowKeepMe, gateProbeTool{name: "denied_tool"})
		if reg.Len() != 0 {
			t.Fatalf("len = %d, want 0（名单内不装）", reg.Len())
		}
	})

	t.Run("名单外照常注册", func(t *testing.T) {
		reg := tool.NewRegistry()
		addGated(reg, allowKeepMe, gateProbeTool{name: "keep_me"})
		if reg.Len() != 1 {
			t.Fatalf("len = %d, want 1", reg.Len())
		}
	})

	t.Run("nil 工具安全", func(t *testing.T) {
		reg := tool.NewRegistry()
		addGated(reg, nil, nil)
		if reg.Len() != 0 {
			t.Fatalf("len = %d, want 0", reg.Len())
		}
	})
}
