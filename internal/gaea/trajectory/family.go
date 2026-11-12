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
	// Totals 是家族用量/费用聚合（1:1 上游 Context Stats 编队行：编队合计/
	// 当前 Agent/子 Agent ×N 三行；费用按各自 usage 事件自报单价计得，
	// 无价事件诚实零贡献）。
	Totals FamilyTotals `json:"totals"`
	// Agents 是成员卡行（刀11，1:1 上游 Fleet 成员卡树：每 Agent 费用+
	// 任务/消息计数；main 在前，子代理按首条消息时间升序）。
	Agents []FamilyAgentRow `json:"agents,omitempty"`
}

// FamilyTotals 是家族用量聚合。
type FamilyTotals struct {
	RootTokens   int64   `json:"rootTokens"`
	RootCost     float64 `json:"rootCost,omitempty"`
	SubTokens    int64   `json:"subTokens"`
	SubCost      float64 `json:"subCost,omitempty"`
	SubCount     int     `json:"subCount"`
	OutputTokens int64   `json:"outputTokens,omitempty"`
}

// FamilyAgentRow 是成员卡（刀11，1:1 上游 Fleet 成员卡树）的一行：每 Agent
// 的用量/费用/任务与消息计数。从各自 transcript 源日志直算（不吃事件流
// 400 条截断，大家族计数不缩水）。
type FamilyAgentRow struct {
	Ref    string  `json:"ref"` // "main" 或 sa_ 引用
	Tokens int64   `json:"tokens"`
	Cost   float64 `json:"cost,omitempty"`
	Output int64   `json:"output,omitempty"`
	// Dispatches：main=已派发委派数（task/run_skill 的 tool_dispatch）；
	// 子代理=1（transcript 首条 user 即委派任务）。
	Dispatches int `json:"dispatches"`
	// Messages 是该 Agent transcript 内消息数（main 含用户+助手；子代理
	// 除首条委派外同口径）。
	Messages int `json:"messages"`
	// Reports 是主会话收到的该子代理完成回投数（main=0）。
	Reports int `json:"reports,omitempty"`
}

// FamilyAgentRows 折叠父会话与子代理 transcript 为成员卡行（main 在前，
// 子代理按首条消息时间升序）。纯函数，与 FoldFamilyEvents 同输入。
func FamilyAgentRows(parent []session.LogEntry, subs map[string][]session.LogEntry) []FamilyAgentRow {
	main := FamilyAgentRow{Ref: FamilyAgentMain}
	subTs := map[string]int64{}
	for _, e := range parent {
		switch e.Kind {
		case "user_message", "message", "assistant_message":
			main.Messages++
		case "tool_dispatch":
			var p struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(e.Payload, &p); err == nil && (p.Name == "task" || p.Name == "run_skill") {
				main.Dispatches++
			}
		}
	}
	main.Tokens, main.Cost, main.Output = UsageOf(parent)

	rows := make([]FamilyAgentRow, 0, len(subs)+1)
	for ref, entries := range subs {
		row := FamilyAgentRow{Ref: ref, Dispatches: 1}
		firstUser := true // 首条 user=委派本体（与 FoldFamilyEvents 同口径）
		for _, e := range entries {
			switch e.Kind {
			case "user_message":
				if firstUser {
					firstUser = false
					continue
				}
				row.Messages++
			case "message", "assistant_message":
				row.Messages++
			}
			if subTs[ref] == 0 && e.Ts > 0 {
				subTs[ref] = e.Ts
			}
		}
		row.Tokens, row.Cost, row.Output = UsageOf(entries)
		rows = append(rows, row)
	}
	// 回报归属：父会话 subagent_message 的 ref 即子代理引用。
	reports := map[string]int{}
	for _, e := range parent {
		if e.Kind != "subagent_message" {
			continue
		}
		var p struct {
			Ref string `json:"ref"`
		}
		if err := json.Unmarshal(e.Payload, &p); err == nil && p.Ref != "" {
			reports[p.Ref]++
		}
	}
	for i := range rows {
		rows[i].Reports = reports[rows[i].Ref]
	}
	sort.Slice(rows, func(i, j int) bool {
		ti, tj := subTs[rows[i].Ref], subTs[rows[j].Ref]
		if ti != tj {
			return ti < tj
		}
		return rows[i].Ref < rows[j].Ref
	})
	return append([]FamilyAgentRow{main}, rows...)
}

// usageOf 折叠一份日志的计费输入/输出 tokens 与费用（usage 事件自报口径，
// 与 contextview fold 同式）。
func UsageOf(entries []session.LogEntry) (tokens int64, cost float64, out int64) {
	for _, e := range entries {
		if e.Kind != "usage" {
			continue
		}
		var p struct {
			PromptTokens     int64   `json:"promptTokens"`
			CompletionTokens int64   `json:"completionTokens"`
			CacheHitTokens   int64   `json:"cacheHitTokens"`
			CacheMissTokens  int64   `json:"cacheMissTokens"`
			Input            float64 `json:"input"`
			Output           float64 `json:"output"`
			CacheHitPrice    float64 `json:"cacheHitPrice"`
		}
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			continue
		}
		tokens += p.PromptTokens + p.CompletionTokens
		out += p.CompletionTokens
		if p.CacheHitPrice > 0 || p.Input > 0 || p.Output > 0 {
			cost += (float64(p.CacheHitTokens)*p.CacheHitPrice +
				float64(p.CacheMissTokens)*p.Input +
				float64(p.CompletionTokens)*p.Output) / 1e6
		}
	}
	return tokens, cost, out
}
