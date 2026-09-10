package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// v4.211 最终文本定稿闸（Options.FinalizeText / SetFinalizeText）：收尾全文
// 在 Message 事件发出前改写，改写值同时是 Run 返回摘要与 session 历史内容——
// 悬空 [MEM:] 引用到不了持久层（记忆 5.1 出口判据「回复发出前全部解析」）。
func TestFinalizeTextRewritesMessageAndSession(t *testing.T) {
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{{Type: provider.ChunkText, Text: "答案 [MEM:ghost] 完"}, {Type: provider.ChunkDone}},
	}}
	sink := &collectorSink{}
	a := New(prov, tool.NewRegistry(), NewSession(""), Options{
		DisableVerify: true,
		FinalizeText:  func(text string) string { return strings.ReplaceAll(text, " [MEM:ghost]", "") },
	}, sink)
	res, err := a.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// 返回摘要（TurnResult.Summary → turnLastSummary）已改写。
	if res.Summary != "答案 完" {
		t.Fatalf("Summary = %q, want 改写后文本", res.Summary)
	}
	// Message 全文事件携带改写后文本（前端整泡替换的来源）。
	var msg *event.Event
	for i := range sink.events {
		if sink.events[i].Kind == event.Message {
			msg = &sink.events[i]
		}
	}
	if msg == nil {
		t.Fatal("缺 Message 收尾事件")
	}
	if msg.Text != "答案 完" {
		t.Fatalf("Message.Text = %q, want 改写后文本", msg.Text)
	}
	// session 历史里的助手消息已改写（下一轮模型不再看到幻觉键）。
	last := a.session.Messages[len(a.session.Messages)-1]
	if last.Content != "答案 完" {
		t.Fatalf("session 内容 = %q, want 改写后文本", last.Content)
	}
}

// nil FinalizeText（子代理/测试缺省）：文本原样透传，行为与历史一致。
func TestFinalizeTextNilIsNoop(t *testing.T) {
	prov := &scriptedProvider{name: "p", turns: [][]provider.Chunk{
		{{Type: provider.ChunkText, Text: "引用 [MEM:any] 原样"}, {Type: provider.ChunkDone}},
	}}
	sink := &collectorSink{}
	a := New(prov, tool.NewRegistry(), NewSession(""), Options{DisableVerify: true}, sink)
	res, err := a.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Summary != "引用 [MEM:any] 原样" {
		t.Fatalf("Summary = %q, want 原样", res.Summary)
	}
	for i := range sink.events {
		if sink.events[i].Kind == event.Message && sink.events[i].Text != "引用 [MEM:any] 原样" {
			t.Fatalf("Message.Text = %q, want 原样", sink.events[i].Text)
		}
	}
}
