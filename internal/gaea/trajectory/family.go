package trajectory

import (
	"encoding/json"
	"sort"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// 家族事件流（v4.542，对标 dsh-context Fleet 事件检查完全体）：主会话与全部
// 子代理 transcript 的消息/委派/回报汇入同一条可筛选时间线。纯函数——输入
// 父会话日志与各子代理日志，输出按时间升序的合并流；I/O 与缓存在 App 绑定层
// （GaeaFamilyEvents）。
//
// 诚实口径：
//   - Agent="main"=主会话，其余为 sa_ 引用（子代理 transcript 首条 user 消息
//     即委派任务，kind=dispatch；assistant 消息=子代理内部产出，kind=message；
//     最终答复由主会话 subagent_message 回投，kind=report——一处产出两视角，
//     与 dsh「运行时结算 vs 子代理写入」的分辨口径一致）；
//   - kind=dispatch 仅含 task/run_skill 两委派元工具（其余工具调用属洞察页
//     工具统计面，不进家族流）；
//   - Text 截断预览（全文在各自日志）；上限 maxFamilyEvents 丢最旧。

const (
	FamilyAgentMain = "main"

	FamilyKindMessage   = "message"
	FamilyKindDispatch  = "dispatch"
	FamilyKindReport    = "report"

	maxFamilyEvents  = 400
	maxFamilyPreview = 160
)

// FamilyEvent 是家族事件流的一条。
type FamilyEvent struct {
	Seq  int64  `json:"seq"`
	Ts   int64  `json:"ts"`
	Agent string `json:"agent"` // "main" 或 sa_ 引用
	Kind  string `json:"kind"`  // message | dispatch | report
	// Role 是 message 专有的发言方（user|assistant）。
	Role string `json:"role,omitempty"`
	// Text 是截断预览（message/repot 的正文、dispatch 的任务摘要）。
	Text string `json:"text,omitempty"`
	// Ref 是 report 专有：完成回投所属的子代理引用（=Agent）。
	Ref string `json:"ref,omitempty"`
}

// FoldFamilyEvents 合并父会话与子代理日志为按时间升序的家族事件流
// （超出 maxFamilyEvents 丢最旧）。subs 键=子代理引用。
func FoldFamilyEvents(parent []session.LogEntry, subs map[string][]session.LogEntry) []FamilyEvent {
	events := make([]FamilyEvent, 0, len(parent)/4+16)
	add := func(e FamilyEvent) { events = append(events, e) }

	// ── 父会话：消息 + 委派派发 + 子代理完成回投 ──
	for _, e := range parent {
		switch e.Kind {
		case "user_message":
			var p struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil || p.Content == "" {
				continue
			}
			add(FamilyEvent{Seq: e.Seq, Ts: e.Ts, Agent: FamilyAgentMain, Kind: FamilyKindMessage, Role: "user", Text: preview(p.Content, maxFamilyPreview)})
		case "message", "assistant_message":
			var p struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil || p.Text == "" {
				continue
			}
			add(FamilyEvent{Seq: e.Seq, Ts: e.Ts, Agent: FamilyAgentMain, Kind: FamilyKindMessage, Role: "assistant", Text: preview(p.Text, maxFamilyPreview)})
		case "tool_dispatch":
			var p struct {
				Name string `json:"name"`
				Args string `json:"args"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				continue
			}
			if p.Name != "task" && p.Name != "run_skill" {
				continue
			}
			add(FamilyEvent{Seq: e.Seq, Ts: e.Ts, Agent: FamilyAgentMain, Kind: FamilyKindDispatch, Text: preview(p.Name+" "+p.Args, maxFamilyPreview)})
		case "subagent_message":
			// 完成回投：payload {text, ref, parentId}——ref 空=临时子代理，
			// Agent 诚实标 main（无稳定归属可查）。
			var p struct {
				Text string `json:"text"`
				Ref  string `json:"ref"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				continue
			}
			agent := p.Ref
			if agent == "" {
				agent = FamilyAgentMain
			}
			add(FamilyEvent{Seq: e.Seq, Ts: e.Ts, Agent: agent, Kind: FamilyKindReport, Ref: p.Ref, Text: preview(p.Text, maxFamilyPreview)})
		}
	}

	// ── 子代理 transcript：首条 user=委派任务（dispatch），其余消息照常 ──
	for ref, entries := range subs {
		firstUser := true
		for _, e := range entries {
			switch e.Kind {
			case "user_message":
				var p struct {
					Content string `json:"content"`
				}
				if err := json.Unmarshal(e.Payload, &p); err != nil || p.Content == "" {
					continue
				}
				if firstUser {
					add(FamilyEvent{Seq: e.Seq, Ts: e.Ts, Agent: ref, Kind: FamilyKindDispatch, Text: preview(p.Content, maxFamilyPreview)})
					firstUser = false
					continue
				}
				add(FamilyEvent{Seq: e.Seq, Ts: e.Ts, Agent: ref, Kind: FamilyKindMessage, Role: "user", Text: preview(p.Content, maxFamilyPreview)})
			case "message", "assistant_message":
				var p struct {
					Text string `json:"text"`
				}
				if err := json.Unmarshal(e.Payload, &p); err != nil || p.Text == "" {
					continue
				}
				add(FamilyEvent{Seq: e.Seq, Ts: e.Ts, Agent: ref, Kind: FamilyKindMessage, Role: "assistant", Text: preview(p.Text, maxFamilyPreview)})
			}
		}
	}

	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Ts != events[j].Ts {
			return events[i].Ts < events[j].Ts
		}
		return events[i].Seq < events[j].Seq
	})
	if len(events) > maxFamilyEvents {
		events = append([]FamilyEvent(nil), events[len(events)-maxFamilyEvents:]...)
	}
	if events == nil {
		events = []FamilyEvent{}
	}
	return events
}

// FamilyEventRefs 列出事件流中出现的全部 agent 身份（main 在前，子代理按
// 首次出现序）——前端 agent 筛选 chips 的数据源。
func FamilyEventRefs(events []FamilyEvent) []string {
	seen := map[string]bool{FamilyAgentMain: true}
	out := []string{FamilyAgentMain}
	for _, e := range events {
		if e.Agent == "" || seen[e.Agent] {
			continue
		}
		seen[e.Agent] = true
		out = append(out, e.Agent)
	}
	return out
}

// FamilyEventStream 是家族事件流的整值快照（GaeaFamilyEvents 返回值）。
type FamilyEventStream struct {
	Ok      bool          `json:"ok"`
	Events  []FamilyEvent `json:"events"`
	Scanned int           `json:"scanned"` // 折叠的会话数（main+子代理）
	// Truncated 是因上限未折叠的子代理 transcript 数（诚实边界）。
	Truncated int `json:"truncated,omitempty"`
	// Refs 是事件流中出现的 agent 身份清单（main 在前）——前端筛选 chips 单源。
	Refs []string `json:"refs"`
}
