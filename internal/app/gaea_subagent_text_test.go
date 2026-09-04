package app

// v4.62.1 回归修复：子代理流式增量（SubagentText）绝不走 gaea-event 主通道。
//
// 回归机理（v4.62.0）：gaea-event 每条 payload 带会话内单调 seq，且 v4.26
// 缺口防线的前提是「seq 与磁盘账本 1:1 对应，丢件可经 GaeaResyncEvents 补拉」。
// SubagentText 是 wire-only（有意不落盘）的装饰性流，走 gaea-event 会消费
// seq 却永远无法补拉——Wails 密集流丢一件即产生不可愈合缺口，前端反复整体
// 重建对话视图，过程可见性被打断。修复：emitGaeaEvent 对 SubagentText 分道
// 到专用通道（无 seq），gaeaEventMap/gaeaKindName 不再含该 kind（不可达的
// 死映射是泄漏源，一并移除）。

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// TestGaeaSubagentTextPayloadRoutesToDedicatedChannel：SubagentText 映射为
// 专用通道 payload（无 seq），其余事件返回 nil 走主通道。
func TestGaeaSubagentTextPayloadRoutesToDedicatedChannel(t *testing.T) {
	m := gaeaSubagentTextPayload(event.Event{
		Kind: event.SubagentText, Text: "增量块", SubagentRef: "sa_1", ParentToolID: "call_9",
	})
	if m == nil {
		t.Fatal("SubagentText 应映射为专用通道 payload，got nil")
	}
	if _, has := m["seq"]; has {
		t.Error("专用通道 payload 不得携带 seq（缺口防线只认账本事件）")
	}
	if m["kind"] != "subagent_text" || m["text"] != "增量块" || m["subagentRef"] != "sa_1" || m["parentId"] != "call_9" {
		t.Fatalf("payload 字段不符：%v", m)
	}

	// 主代理文本增量等其余事件不劫走
	if gaeaSubagentTextPayload(event.Event{Kind: event.Text, Text: "主代理"}) != nil {
		t.Error("Text 事件不应走专用通道")
	}
	if gaeaSubagentTextPayload(event.Event{Kind: event.SubagentMessage, Text: "收尾"}) != nil {
		t.Error("SubagentMessage 应留在 gaea-event 主通道（账本事件，可补拉）")
	}
}

// TestGaeaEventMapHasNoSubagentText：gaea-event 转译层不含 subagent_text
// 映射——该 kind 经 emitGaeaEvent 分道后永不可达，留映射即死代码（泄漏源）。
func TestGaeaEventMapHasNoSubagentText(t *testing.T) {
	m := gaeaEventMap(event.Event{Kind: event.SubagentText, Text: "x"})
	if k, _ := m["kind"].(string); k == "subagent_text" {
		t.Error("gaeaEventMap 不应再映射 subagent_text（分道后不可达）")
	}
	if gaeaKindName(event.SubagentText) == "subagent_text" {
		t.Error("gaeaKindName 不应再映射 SubagentText（分道后不可达）")
	}
}

// TestForwarderSeqUnconsumedBySubagentText：分道后 SubagentText 不消费 wire
// seq——连续喂 SubagentText 后再喂主通道事件，seq 仍无断号（缺口防线前提）。
func TestForwarderSeqUnconsumedBySubagentText(t *testing.T) {
	f := newGaeaEventForwarder()
	// 模拟 emitGaeaEvent 分道路由：SubagentText 不过 payload
	events := []event.Event{
		{Kind: event.SubagentText, Text: "增1", SubagentRef: "sa_1"},
		{Kind: event.SubagentText, Text: "增2", SubagentRef: "sa_1"},
		{Kind: event.Text, Text: "主文本"},
		{Kind: event.SubagentText, Text: "增3", SubagentRef: "sa_1"},
		{Kind: event.Phase, Text: "执行中"},
	}
	var seqs []int64
	for _, e := range events {
		if gaeaSubagentTextPayload(e) != nil {
			continue // 专用通道，不占 seq
		}
		if m := f.payload(e); m != nil {
			seqs = append(seqs, m["seq"].(int64))
		}
	}
	// 只有 Text 进了主通道（Phase 同文案节流后首发也占一个 seq）；相邻主通道
	// 事件间无论夹多少 SubagentText，seq 必须连续无断号。
	if len(seqs) < 2 {
		t.Fatalf("主通道事件数 = %d，want ≥2", len(seqs))
	}
	for i := 1; i < len(seqs); i++ {
		if seqs[i] != seqs[i-1]+1 {
			t.Fatalf("seq 断号：第 %d 个 = %d，前一 = %d（SubagentText 泄漏进 seq 序列）", i, seqs[i], seqs[i-1])
		}
	}
}
