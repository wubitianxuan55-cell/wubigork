package agent

// 刀A（v4.245）性能基线：逐消息摘要成本。DigestMessages 每次全量重哈希
// （改前 stream() 每步的实际成本，O(会话总字节)）；DigestCache 前缀身份
// 复用后只重哈希新增/改写消息（改后 stream() 路径，步进成本 O(新增字节)）。

import (
	"fmt"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// benchHistory 构造近似真实形态的长历史：system + 多轮 user/assistant/tool。
func benchHistory(turns int) []provider.Message {
	msgs := []provider.Message{{Role: provider.RoleSystem, Content: "系统提示词，长度中等，包含若干段落。\n\n第二段。"}}
	for i := 0; i < turns; i++ {
		msgs = append(msgs,
			provider.Message{Role: provider.RoleUser, Content: fmt.Sprintf("第 %d 个用户请求，描述一个中等复杂度的任务，几百字节的正文……", i)},
			provider.Message{Role: provider.RoleAssistant, Content: fmt.Sprintf("第 %d 个回合的思考与回复，包含多段文本与结论。", i)},
			provider.Message{Role: provider.RoleTool, ToolCallID: fmt.Sprintf("call_%d", i), Name: "read_file",
				Content: fmt.Sprintf("第 %d 个工具结果：约 2KB 的文件内容 %s", i, string(make([]byte, 1800)))},
		)
	}
	return msgs
}

// BenchmarkDigestMessages 全量重哈希基线（改前行为）。
func BenchmarkDigestMessages(b *testing.B) {
	for _, turns := range []int{50, 300} {
		msgs := benchHistory(turns)
		b.Run(fmt.Sprintf("turns=%d", turns), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				DigestMessages(msgs)
			}
		})
	}
}

// BenchmarkDigestCacheStep 增量步进（改后行为）：每步只追加一条消息。
func BenchmarkDigestCacheStep(b *testing.B) {
	for _, turns := range []int{50, 300} {
		b.Run(fmt.Sprintf("turns=%d", turns), func(b *testing.B) {
			base := benchHistory(turns)
			c := &DigestCache{}
			extra := provider.Message{Role: provider.RoleTool, ToolCallID: "step", Name: "read_file",
				Content: "步进新增的工具结果：" + string(make([]byte, 1800))}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				msgs := append(base, extra) // 模拟本步新增一条消息
				c.Digest(msgs)
				base = msgs
			}
		})
	}
}
