package agent

// 刀A（v4.245）增量摘要缓存：与 DigestMessages 逐元素等价（任何序列形态），
// 前缀指针身份相符部分复用（hits 计数），身份断裂处重哈希。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func msgsOf(cs ...string) []provider.Message {
	out := make([]provider.Message, 0, len(cs))
	for _, c := range cs {
		out = append(out, provider.Message{Role: provider.RoleUser, Content: c})
	}
	return out
}

// 等价性：缓存路径与全量路径逐元素一致——追加 / 截断 / 中间改写 / 整体
// 换血 / 首次调用 / 空序列，全部形态。
func TestDigestCacheEquivalence(t *testing.T) {
	base := msgsOf("一", "二", "三", "四")
	shapes := map[string][]provider.Message{
		"首次":   base,
		"追加":   append(append([]provider.Message{}, base...), msgsOf("五")...),
		"截断":   base[:2],
		"整体换血": msgsOf("甲", "乙"),
		"空":    nil,
		"工具调用追加": append(append([]provider.Message{}, base...),
			provider.Message{Role: provider.RoleAssistant,
				ToolCalls: []provider.ToolCall{{ID: "t1", Name: "read_file", Arguments: "{}"}}}),
	}

	c := &DigestCache{}
	var last []provider.Message
	for name, m := range shapes {
		got := c.Digest(m)
		want := DigestMessages(m)
		if len(got) != len(want) {
			t.Fatalf("%s: 长度不符 got=%d want=%d", name, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: msg %d 不符 got=%+v want=%+v", name, i, got[i], want[i])
			}
		}
		_ = last
	}
}

// 复用计数：追加=全前缀复用；中间改写=复用到断裂处；换血=零复用。
func TestDigestCacheReuse(t *testing.T) {
	c := &DigestCache{}
	a, b, c3 := "甲", "乙", "丙"
	first := []provider.Message{
		{Role: provider.RoleUser, Content: a},
		{Role: provider.RoleUser, Content: b},
	}
	c.Digest(first)
	if c.hits != 0 {
		t.Fatalf("首次调用 hits = %d, want 0", c.hits)
	}

	// 追加：前两条同一份字符串 ⇒ 全前缀复用。
	appended := append(append([]provider.Message{}, first...),
		provider.Message{Role: provider.RoleUser, Content: c3})
	c.Digest(appended)
	if c.hits != 2 {
		t.Fatalf("追加 hits = %d, want 2", c.hits)
	}

	// 中间改写：第一条同一份字符串复用，第二条换新内容断裂。
	rewritten := []provider.Message{
		{Role: provider.RoleUser, Content: a},
		{Role: provider.RoleUser, Content: b + "（改）"},
		{Role: provider.RoleUser, Content: c3},
	}
	c.Digest(rewritten)
	if c.hits != 1 {
		t.Fatalf("中间改写 hits = %d, want 1", c.hits)
	}

	// 整体换血（模拟会话切换）：首条即断裂。
	c.Digest(msgsOf("子", "丑"))
	if c.hits != 0 {
		t.Fatalf("换血 hits = %d, want 0", c.hits)
	}

	// 工具调用消息：参数同串但切片重建 ⇒ 保守重哈希（等价性不受影响）。
	tc := []provider.ToolCall{{ID: "1", Name: "read_file", Arguments: "{}"}}
	c.Digest([]provider.Message{{Role: provider.RoleAssistant, ToolCalls: tc}})
	c.Digest([]provider.Message{{Role: provider.RoleAssistant, ToolCalls: tc}})
	if c.hits != 1 {
		t.Fatalf("同一切片复用 hits = %d, want 1", c.hits)
	}
	c.Digest([]provider.Message{{Role: provider.RoleAssistant,
		ToolCalls: []provider.ToolCall{{ID: "1", Name: "read_file", Arguments: "{}"}}}})
	if c.hits != 0 {
		t.Fatalf("重建切片应重哈希 hits = %d, want 0", c.hits)
	}
}
