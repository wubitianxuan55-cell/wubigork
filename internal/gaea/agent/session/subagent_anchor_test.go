package session

// v4.34.0 线A：ProjectSubagentAnchors 锚点投影测试。
// 核心断言口径：锚点游标与 ProjectMessages 的消息条数逐项同拍（每组锚点
// 位置都可用同流投影出的消息条数交叉验证）。

import (
	"reflect"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
)

// subPayload 构造 subagent_message payload（与 log.go EntryFromEvent 写入端
// map[string]string{"text","ref","parentId"} 同形状）。
func subPayload(text, ref, parentID string) map[string]string {
	return map[string]string{"text": text, "ref": ref, "parentId": parentID}
}

// KindSubagentMessage 常量必须与写端（log.go KindString）和磁盘格式双锚定。
func TestKindSubagentMessageConstant(t *testing.T) {
	if got := KindString(event.SubagentMessage); got != KindSubagentMessage {
		t.Fatalf("KindString(SubagentMessage) = %q, want %q（常量与写端脱钩）", got, KindSubagentMessage)
	}
	if KindSubagentMessage != "subagent_message" {
		t.Fatalf("KindSubagentMessage = %q, want 磁盘 kind %q", KindSubagentMessage, "subagent_message")
	}
}

// 混合流：user/assistant 流式/tool/subagent_message 穿插，逐项断言锚点位置
// 与计数值；末尾连续两个 subagent_message 验证同 K 保序。
func TestProjectSubagentAnchorsMixedStream(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: KindUserMessage, Payload: raw(t, userLogPayload{Content: "u1"})},                          // 投影 1
		{Seq: 2, Kind: KindSubagentMessage, Payload: raw(t, subPayload("早退子代理", "", ""))},                        // 锚点 K=1
		{Seq: 3, Kind: KindAssistantStarted, Payload: raw(t, map[string]string{})},                               // 投影 2（pending 打开）
		{Seq: 4, Kind: KindAssistantDelta, Payload: raw(t, map[string]string{"text": "思"})},                      // +0
		{Seq: 5, Kind: "text", Payload: raw(t, map[string]string{"text": "考"})},                                  // +0
		{Seq: 6, Kind: "tool_dispatch", Payload: raw(t, toolCallLogPayload{ID: "t1", Name: "task", Args: "{}"})}, // +0
		{Seq: 7, Kind: KindSubagentMessage, Payload: raw(t, subPayload("子答复A", "sa_a", "t1"))},                   // 锚点 K=2
		{Seq: 8, Kind: KindToolResult, Payload: raw(t, toolResultLogPayload{ID: "t1", Output: "o"})},             // 投影 3（pending 关闭）
		{Seq: 9, Kind: KindAssistantMessage, Payload: raw(t, assistantLogPayload{ID: "m1", Text: "最终"})},         // pending<0 → 投影 4
		{Seq: 10, Kind: "message", Payload: raw(t, assistantLogPayload{ID: "m1", Text: "事件级全文"})},                // pending>=0 → 原地替换 +0
		{Seq: 11, Kind: "reasoning", Payload: raw(t, map[string]string{"reasoning": "r"})},                       // +0
		{Seq: 12, Kind: "turn_done", Payload: raw(t, map[string]string{})},                                       // 非消息事件 +0
		{Seq: 13, Kind: KindUserMessage, Payload: raw(t, userLogPayload{Content: "u2"})},                         // 投影 5
		{Seq: 14, Kind: KindSubagentMessage, Payload: raw(t, subPayload("子答复B", "sa_b", ""))},                    // 锚点 K=5
		{Seq: 15, Kind: KindSubagentMessage, Payload: raw(t, subPayload("子答复C", "", ""))},                        // 锚点 K=5（连续，保序）
	}
	// 同口径交叉验证：同流投影出的消息条数 = 最后一个锚点的游标值。
	if got := len(ProjectMessages(entries)); got != 5 {
		t.Fatalf("ProjectMessages = %d, want 5（口径漂移）", got)
	}
	anchors := ProjectSubagentAnchors(entries)
	want := []SubagentAnchor{
		{Text: "早退子代理", AfterMsgIndex: 1},
		{Text: "子答复A", Ref: "sa_a", ParentToolID: "t1", AfterMsgIndex: 2},
		{Text: "子答复B", Ref: "sa_b", AfterMsgIndex: 5},
		{Text: "子答复C", AfterMsgIndex: 5},
	}
	if !reflect.DeepEqual(anchors, want) {
		t.Fatalf("anchors = %+v, want %+v", anchors, want)
	}
}

// 空流 / 无 subagent 流 → 空锚点集。
func TestProjectSubagentAnchorsEmpty(t *testing.T) {
	if got := ProjectSubagentAnchors(nil); len(got) != 0 {
		t.Fatalf("空流 anchors = %+v, want 空", got)
	}
	noSub := []LogEntry{
		{Seq: 1, Kind: KindUserMessage, Payload: raw(t, userLogPayload{Content: "u"})},
		{Seq: 2, Kind: KindAssistantMessage, Payload: raw(t, assistantLogPayload{Text: "a"})},
	}
	if got := ProjectSubagentAnchors(noSub); len(got) != 0 {
		t.Fatalf("无 subagent 流 anchors = %+v, want 空", got)
	}
}

// 连续两个 subagent_message：保序、游标不变。
func TestProjectSubagentAnchorsConsecutivePreserveOrder(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: KindUserMessage, Payload: raw(t, userLogPayload{Content: "u"})},
		{Seq: 2, Kind: KindSubagentMessage, Payload: raw(t, subPayload("第一个", "sa_1", ""))},
		{Seq: 3, Kind: KindSubagentMessage, Payload: raw(t, subPayload("第二个", "sa_2", ""))},
	}
	anchors := ProjectSubagentAnchors(entries)
	if len(anchors) != 2 {
		t.Fatalf("anchors = %d 条, want 2", len(anchors))
	}
	if anchors[0].Text != "第一个" || anchors[0].Ref != "sa_1" || anchors[0].AfterMsgIndex != 1 {
		t.Fatalf("anchor0 = %+v", anchors[0])
	}
	if anchors[1].Text != "第二个" || anchors[1].Ref != "sa_2" || anchors[1].AfterMsgIndex != 1 {
		t.Fatalf("anchor1 = %+v（同 K 应保序且游标不变）", anchors[1])
	}
}

// 隐式 assistant 建立：无 assistant_started 时首个 text/tool_call 会先补一条
// assistant 消息（投影 ensurePending），游标同拍 +1。
func TestProjectSubagentAnchorsImplicitAssistant(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: "text", Payload: raw(t, map[string]string{"text": "a"})},                               // 隐式 assistant → 投影 1
		{Seq: 2, Kind: KindToolCall, Payload: raw(t, toolCallLogPayload{ID: "t1", Name: "bash", Args: "{}"})}, // +0
		{Seq: 3, Kind: KindSubagentMessage, Payload: raw(t, subPayload("子答复", "sa_x", "t1"))},                 // 锚点 K=1
	}
	if got := len(ProjectMessages(entries)); got != 1 {
		t.Fatalf("ProjectMessages = %d, want 1（隐式 assistant 口径漂移）", got)
	}
	anchors := ProjectSubagentAnchors(entries)
	if len(anchors) != 1 || anchors[0].AfterMsgIndex != 1 || anchors[0].ParentToolID != "t1" {
		t.Fatalf("anchors = %+v, want K=1 且 parentId=t1", anchors)
	}
}
