package agent

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// headersOf 提取收集到的 RequestHeader 事件。
func headersOf(sink *collectorSink) []event.RequestHeaderInfo {
	var out []event.RequestHeaderInfo
	for _, e := range sink.events {
		if e.Kind == event.RequestHeader {
			out = append(out, e.Header)
		}
	}
	return out
}

// v4.212（5.2 出口判据）：同会话相邻两轮请求——首请求无判定（nil），其后
// 每个请求相对上一请求前缀字节级稳定（Stable=true），新增消息数正确；
// CompileHash/PrevCompileHash 构成可回放的摘要链。
func TestAdjacentRequestPrefixStable(t *testing.T) {
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		// 回合1：工具轮 → 工具结果 → 最终答复（同一回合 3 次请求）。
		{toolCallChunk("c1", "get_time", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "工具完毕"}, toolCallChunk("c2", "get_time", `{}`), {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "回合1完成"}, {Type: provider.ChunkDone}},
		// 回合2：直接答复（第 4 次请求）。
		{{Type: provider.ChunkText, Text: "回合2完成"}, {Type: provider.ChunkDone}},
	}}
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "get_time", readOnly: true})
	sink := &collectorSink{}
	a := New(prov, reg, NewSession(""), Options{DisableVerify: true}, sink)

	if _, err := a.Run(context.Background(), "第一问"); err != nil {
		t.Fatalf("run1: %v", err)
	}
	if _, err := a.Run(context.Background(), "第二问"); err != nil {
		t.Fatalf("run2: %v", err)
	}
	hs := headersOf(sink)
	if len(hs) != 4 {
		t.Fatalf("request_header 数 = %d, want 4", len(hs))
	}
	// 首请求：有 CompileHash，无判定字段（nil）。
	if hs[0].CompileHash == "" || hs[0].PrefixStable != nil {
		t.Fatalf("首请求形态不符: hash=%q stable=%v", hs[0].CompileHash, hs[0].PrefixStable)
	}
	// 其后每个请求：前缀稳定 + 摘要链接续。
	for i := 1; i < len(hs); i++ {
		h, p := hs[i], hs[i-1]
		if h.PrefixStable == nil || !*h.PrefixStable {
			t.Fatalf("请求%d 应前缀稳定: %+v", i+1, h)
		}
		if h.RewriteCount != 0 {
			t.Fatalf("请求%d 不应有改写: %+v", i+1, h)
		}
		if h.AppendCount <= 0 {
			t.Fatalf("请求%d 应有新增消息: %+v", i+1, h)
		}
		if h.PrevCompileHash != p.CompileHash {
			t.Fatalf("请求%d 摘要链断裂: prev=%q want %q", i+1, h.PrevCompileHash, p.CompileHash)
		}
	}
}

// 改写归因：两次请求之间外部改写历史（模拟压缩/rewind 的结构性变化）→
// Stable=false 且 RewriteCount 标出被改写的消息数（诚实判漂移，不谎报稳定）。
func TestPrefixRewriteDetected(t *testing.T) {
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{{Type: provider.ChunkText, Text: "一"}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "二"}, {Type: provider.ChunkDone}},
	}}
	sink := &collectorSink{}
	sess := NewSession("")
	a := New(prov, tool.NewRegistry(), sess, Options{DisableVerify: true}, sink)
	if _, err := a.Run(context.Background(), "问1"); err != nil {
		t.Fatalf("run1: %v", err)
	}
	// 结构性改写：篡改最早的用户消息（compaction/rewind 同类形态）。
	sess.Messages[0].Content = "问1（已压缩改写）"
	if _, err := a.Run(context.Background(), "问2"); err != nil {
		t.Fatalf("run2: %v", err)
	}
	hs := headersOf(sink)
	if len(hs) != 2 {
		t.Fatalf("request_header 数 = %d, want 2", len(hs))
	}
	if hs[1].PrefixStable == nil || *hs[1].PrefixStable {
		t.Fatalf("改写后应判不稳定: %+v", hs[1])
	}
	if hs[1].RewriteCount < 1 {
		t.Fatalf("应标出被改写消息数: %+v", hs[1])
	}
}
