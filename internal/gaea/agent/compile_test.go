package agent

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// 消息摘要：同内容同摘要、内容变化摘要变化、工具调用参与摘要（5.2 编译链）。
func TestDigestMessages(t *testing.T) {
	a := provider.Message{Role: provider.RoleUser, Content: "你好"}
	b := provider.Message{Role: provider.RoleUser, Content: "你好"}
	da, db := DigestMessages([]provider.Message{a})[0], DigestMessages([]provider.Message{b})[0]
	if da.Hash != db.Hash || da.Bytes != db.Bytes {
		t.Fatalf("同内容摘要不一致: %+v vs %+v", da, db)
	}
	c := provider.Message{Role: provider.RoleUser, Content: "你好!"}
	dc := DigestMessages([]provider.Message{c})[0]
	if dc.Hash == da.Hash {
		t.Fatalf("内容变化摘要应变化")
	}
	// 拼接歧义防护：「AB|C」与「A|BC」不同摘要（长度前缀口径）。
	x := DigestMessages([]provider.Message{{Role: "u", Content: "AB"}, {Role: "s", Content: "C"}})
	y := DigestMessages([]provider.Message{{Role: "u", Content: "A"}, {Role: "s", Content: "BC"}})
	if SequenceHash(x) == SequenceHash(y) {
		t.Fatalf("拼接歧义：不同边界得到同序列摘要")
	}
	// 助手消息的工具调用参与摘要。
	tc1 := provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "1", Name: "read_file", Arguments: "{}"}}}
	tc2 := provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "1", Name: "read_file", Arguments: "{\"path\":\"x\"}"}}}
	if SequenceHash(DigestMessages([]provider.Message{tc1})) == SequenceHash(DigestMessages([]provider.Message{tc2})) {
		t.Fatalf("工具调用参数变化摘要应变化")
	}
}

// 前缀稳定判定：全等 / 追加 / 改写 / 混合（前缀稳定+尾部追加）四种形态。
func TestDiffMessages(t *testing.T) {
	m := func(cs ...string) []provider.Message {
		out := make([]provider.Message, 0, len(cs))
		for _, c := range cs {
			out = append(out, provider.Message{Role: provider.RoleUser, Content: c})
		}
		return out
	}
	digest := DigestMessages

	// 追加：prev 是 curr 的前缀 → Stable、Rewrite=0、Append=差值。
	prev := digest(m("a", "b"))
	curr := digest(m("a", "b", "c"))
	d := DiffMessages(prev, curr)
	if !d.Stable || d.RewriteCount != 0 || d.AppendCount != 1 || d.PrefixBytes <= 0 {
		t.Fatalf("追加判定不符: %+v", d)
	}

	// 全等：Append=0。
	d = DiffMessages(prev, digest(m("a", "b")))
	if !d.Stable || d.AppendCount != 0 || d.PrefixBytes <= 0 {
		t.Fatalf("全等判定不符: %+v", d)
	}

	// 改写：第二条消息变化 → 首条仍是稳定前缀：Rewrite=1、Append=1、
	// Stable=false（前缀在第二条处断裂）。
	d = DiffMessages(prev, digest(m("a", "B")))
	if d.Stable || d.RewriteCount != 1 || d.AppendCount != 1 {
		t.Fatalf("改写判定不符: %+v", d)
	}

	// 混合：首条改写且尾部追加 → Rewrite=2、Append=3。
	d = DiffMessages(prev, digest(m("A", "b", "c")))
	if d.Stable || d.RewriteCount != 2 || d.AppendCount != 3 {
		t.Fatalf("混合判定不符: %+v", d)
	}

	// 首请求（prev 空）：Stable=false、无改写（调用方据此不下发判定字段）。
	d = DiffMessages(nil, curr)
	if d.Stable || d.RewriteCount != 0 || d.AppendCount != 3 {
		t.Fatalf("首请求判定不符: %+v", d)
	}
}
