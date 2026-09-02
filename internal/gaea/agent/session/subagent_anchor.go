package session

// v4.34.0 线A：subagent_message 的 UI 侧锚点投影。
//
// 根因：ProjectMessages 的 switch 没有 subagent_message case，该条目被整条
// 忽略（这是有意的——projected 消息直接喂给模型，运行期模型上下文同样不含
// 子代理答复，投影语义绝不能改）。代价是恢复会话后 provider History 里没有
// 子代理答复，「子代理」徽标气泡整条消失。
//
// 本文件提供一条与投影并行的只读游标：ProjectSubagentAnchors 按与
// ProjectMessages 完全一致的遍历口径统计「该条目出现时已投影出的消息条数」，
// 把每个 subagent_message 记录为 {Text, Ref, ParentToolID, AfterMsgIndex}
// 锚点。消费方（internal/app 的 GaeaHistory）据此把子代理答复插回 UI 历史
// 流——模型可见上下文不受任何影响。

import "encoding/json"

// KindSubagentMessage 是 subagent_message 日志 kind。log.go 的 KindString 对
// event.SubagentMessage 返回该字面量（session 包此前未为其定义常量），这里
// 补上供读端引用；两侧必须一致（有 golden 断言的往返测试兜底）。
const KindSubagentMessage = "subagent_message"

// SubagentAnchor 是 subagent_message 条目的 UI 侧锚点：该答复气泡应插入在
// provider 消息流的第 AfterMsgIndex 条消息之后（AfterMsgIndex = 该条目出现时
// ProjectMessages 已投影出的消息条数，与 ProjectMessages 同口径遍历计数）。
type SubagentAnchor struct {
	Text string
	// Ref 是子代理 transcript 引用（"sa_..."）；空 = 临时子代理。
	Ref string
	// ParentToolID 是父 task 工具调用 ID（可空）。
	ParentToolID string
	// AfterMsgIndex 是锚点插入位置：在此条 provider 消息（0 起算的第
	// AfterMsgIndex-1 条）产出的全部 UI 条目之后。0 = 所有消息之前。
	AfterMsgIndex int
}

// ProjectSubagentAnchors 按与 ProjectMessages 完全相同的遍历口径，从日志条目
// 流中解析 subagent_message 锚点（纯函数，不改变 ProjectMessages 的投影语义）。
//
// 游标口径（与投影逐 case 对齐，out 增量即 projected 游标增量）：
//   - system_message / user_message / assistant_started        → +1；
//   - assistant_delta / text / reasoning / tool_call / tool_dispatch
//     → ensurePending：pending<0 时先补一条 assistant 消息（+1），否则 +0；
//   - assistant_message / message                              → pending<0 时
//     +1，pending>=0 时原地替换（+0）；两种情况都不关闭 pending；
//   - tool_result                                              → 终结 pending，+1；
//   - 其余 kind（含 subagent_message 等非消息事件）             → +0。
//
// 遇 subagent_message 时记录锚点，游标不变。返回顺序 = 日志顺序；同一位置的
// 多个锚点保持相对顺序。payload 形状对齐 log.go EntryFromEvent 写入端：
// {text, ref, parentId}。
func ProjectSubagentAnchors(entries []LogEntry) []SubagentAnchor {
	var out []SubagentAnchor
	projected := 0 // 当前已投影出的消息条数（= 投影中 len(out) 的镜像）
	pending := -1  // 当前正在累积的 assistant 消息下标；-1 = 无（与投影同状态机）
	for _, e := range entries {
		switch e.Kind {
		case KindAssistantStarted:
			// 投影侧：终结旧 pending 并追加一条 assistant 消息、打开新 pending。
			pending = projected
			projected++

		case KindSystemMessage, KindUserMessage:
			pending = -1
			projected++

		case KindAssistantDelta, "text", "reasoning", KindToolCall, "tool_dispatch":
			// 投影侧 ensurePending：无 pending 时先建立一条 assistant 消息。
			// 无论 payload 内容如何投影都会建立（unmarshal 失败不回退），这里
			// 同口径计数。
			if pending < 0 {
				pending = projected
				projected++
			}

		case KindAssistantMessage, "message":
			// 投影侧：有 pending 时原地替换（不增条数、不关闭），否则追加并
			// 打开 pending。
			if pending < 0 {
				pending = projected
				projected++
			}

		case KindToolResult:
			pending = -1
			projected++

		case KindSubagentMessage:
			var p struct {
				Text     string `json:"text"`
				Ref      string `json:"ref"`
				ParentID string `json:"parentId"`
			}
			_ = json.Unmarshal(e.Payload, &p)
			out = append(out, SubagentAnchor{
				Text:          p.Text,
				Ref:           p.Ref,
				ParentToolID:  p.ParentID,
				AfterMsgIndex: projected,
			})
		}
	}
	return out
}
