package agent

// 阶段九 9.4「KV 缓存友好」出口判据：前缀字节级稳定判据**延续并覆盖压缩后
// 首轮**——压缩是结构性改写（如实判不稳定+RewriteCount 点名），压缩后第二轮
// 起恢复稳定（[system+firstUser+digest] 固定前缀全量 cache hit 的设计承诺）。

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func TestPrefixStableAcrossCompaction(t *testing.T) {
	long := strings.Repeat("内容填充", 300) // 撑起窗口压力
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		// 回合1：一次答复（请求1）。
		{{Type: provider.ChunkText, Text: "回合1完成"}, {Type: provider.ChunkDone}},
		// CompactNow 的摘要请求（请求2）。
		{{Type: provider.ChunkText, Text: "## Goal\n短摘要"}, {Type: provider.ChunkDone}},
		// 压缩后首轮（请求3）。
		{{Type: provider.ChunkText, Text: "压缩后首轮"}, {Type: provider.ChunkDone}},
		// 压缩后第二轮（请求4）。
		{{Type: provider.ChunkText, Text: "压缩后次轮"}, {Type: provider.ChunkDone}},
	}}
	reg := tool.NewRegistry()
	sink := &collectorSink{}
	sess := NewSession("system prompt here")
	a := New(prov, reg, sess, Options{DisableVerify: true,
		Compaction: CompactionConfig{Window: 2000, Ratio: 0.8}}, sink)

	if _, err := a.Run(context.Background(), "第一问 "+long); err != nil {
		t.Fatalf("run1: %v", err)
	}
	if err := a.CompactNow(context.Background(), ""); err != nil {
		t.Fatalf("CompactNow: %v", err)
	}
	if _, err := a.Run(context.Background(), "压缩后第一问"); err != nil {
		t.Fatalf("run after compact: %v", err)
	}
	if _, err := a.Run(context.Background(), "压缩后第二问"); err != nil {
		t.Fatalf("run2 after compact: %v", err)
	}

	hs := headersOf(sink)
	// RequestHeader 只覆盖主请求路径（summarize 直连 prov.Stream 不 emit）：
	// 主1 + 压缩后 2 = 3。
	if len(hs) != 3 {
		t.Fatalf("request_header 数 = %d, want 3（主1+压缩后2）", len(hs))
	}
	// 压缩后首轮：digest 替换折叠区=结构性改写，如实判不稳定+RewriteCount 点名
	// （诚实判漂移——此时谎报稳定才是 bug）。
	first := hs[1]
	if first.PrefixStable == nil || *first.PrefixStable {
		t.Fatalf("压缩后首轮应判不稳定（结构性改写）: %+v", first)
	}
	if first.RewriteCount <= 0 {
		t.Fatalf("压缩后首轮应标出改写消息数: %+v", first)
	}
	// 摘要链接续：CompileHash 链不断（可回放）。
	for i := 1; i < len(hs); i++ {
		if hs[i].PrevCompileHash != hs[i-1].CompileHash {
			t.Fatalf("请求%d 摘要链断裂: prev=%q want %q", i+1, hs[i].PrevCompileHash, hs[i-1].CompileHash)
		}
	}
	// 压缩后第二轮：digest 固定前缀 → 恢复稳定（cache hit 设计承诺的判据面）。
	second := hs[2]
	if second.PrefixStable == nil || !*second.PrefixStable {
		t.Fatalf("压缩后次轮应恢复稳定: %+v", second)
	}
	if second.RewriteCount != 0 {
		t.Fatalf("压缩后次轮不应再改写: %+v", second)
	}
}

// 9.4 cache-aligned 传输面判据：摘要请求的 messages 前缀（system+折叠区）与
// 压缩前会话逐字一致（Letta self_compact 口径——前缀命中 provider 热 KV，摘
// 要输入按 cache-read 计价），压缩指令只允许出现在最后一条 user 消息。
func TestSummarizeRequestCacheAlignedPrefix(t *testing.T) {
	s := NewSession("system prompt here")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "user goal xxx"})
	for i := 0; i < 12; i++ {
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "result " + strings.Repeat("r", 200)})
	}
	// 压缩前会话快照（fold 重放的逐字基准）。
	before := append([]provider.Message(nil), s.Messages...)
	a := testAgentC(2000, 0.8, 5)
	a.session = s
	prov := &mockProvider{name: "lean", chunks: []provider.Chunk{{Type: provider.ChunkText, Text: "## Goal\nshort"}}}
	a.prov = prov

	if err := a.CompactNow(context.Background(), ""); err != nil {
		t.Fatalf("CompactNow: %v", err)
	}
	req := prov.lastReq.Messages
	if len(req) < 3 {
		t.Fatalf("摘要请求消息数不足: %d", len(req))
	}
	// 前缀逐字：system 与会话一致。
	if req[0].Role != provider.RoleSystem || req[0].Content != "system prompt here" {
		t.Fatalf("摘要请求应复用会话 system: %+v", req[0])
	}
	// 中间消息 = 压缩前会话的逐字子序列（原样原序重放，不重排不截断）。
	j := 0
	for i := 1; i < len(req)-1; i++ {
		found := false
		for ; j < len(before); j++ {
			if before[j].Role == req[i].Role && before[j].Content == req[i].Content {
				found = true
				j++
				break
			}
		}
		if !found {
			t.Fatalf("摘要请求消息 %d 不是压缩前会话的逐字重放: %+v", i, req[i])
		}
	}
	// 指令只允许出现在最后一条 user 消息。
	last := req[len(req)-1]
	if last.Role != provider.RoleUser || !strings.Contains(last.Content, summaryInstruction) {
		t.Fatalf("压缩指令应为最后一条 user 消息: %+v", last)
	}
	for i, m := range req[:len(req)-1] {
		if strings.Contains(m.Content, summaryInstruction) {
			t.Fatalf("指令泄漏进前缀消息 %d（破坏缓存对齐）: %+v", i, m)
		}
	}
}
