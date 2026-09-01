package app

// v4.26 对话流式重造：事件补拉（防吞件）绑定面。
//
// Why：Wails 事件流在密集到达时会丢件（前端 store 注释认账，此前仅 turn_done
// 有最终回答兜底）。传输层丢件无法根治，gaea-event payload 现携带单调 seq
// （见 gaea_handler.go 的 gaeaEventForwarder），前端检测到断号后调用本绑定，
// 用当前会话的磁盘事件日志整体重建对话 items 视图——日志是持久真相源
// （轨迹面板 GaeaTrajectory 一直读它所以不受吞件影响），本绑定把同一份数据
// 以对话 Item 形态交还前端。

import (
	"encoding/json"
	"strconv"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// GaeaResyncItem 是补拉重建的单条对话项（前端 Item 联合类型的 Go 投影，
// 与 frontend/src/gaea/lib/store.ts 的 Item 对齐）。Kind 取四类（最小折叠器）：
//   - user:      Text
//   - assistant: Text / Reasoning（Streaming 恒 false，前端可直接按完整态渲染）
//   - tool:      Name / Args / ToolID（前后端合并键）/ Output / Err /
//     ReadOnly / Truncated / ParentID / 状态由 Status 表达
//   - notice:    Level（info|warn）/ Text
//
// ID 生成规则：user=`u<日志seq>`、assistant=`a<日志seq>`、notice=`n<日志seq>`、
// tool 直接用工具调用 ID（与实时流中的 tool 卡片合并键一致）。
//
// v4.26.2：全部字段去掉 omitempty、序列化恒全键。Why：前端 parseResyncItems
// 按「形状必须一致」严格校验（assistant 缺 reasoning / tool 缺 readOnly:false
// 即整快照判坏弃用），omitempty 会把空串/false 的键整个省略——真实流式回合
// 里几乎每个条目都缺键，导致补拉快照 100% 被前端拒绝、序号防线静默失效
// （对话窗只剩 WorkHeader 读秒，过程卡/文本卡全靠轨迹面板才可见）。序列化
// 全键 + 前端对缺省键宽容（类型错仍拒）双保险。
type GaeaResyncItem struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	Text      string `json:"text"`
	Reasoning string `json:"reasoning"`
	Streaming bool   `json:"streaming"`
	Level     string `json:"level"`
	Name      string `json:"name"`
	Args      string `json:"args"`
	ToolID    string `json:"toolId"`
	Output    string `json:"output"`
	Err       string `json:"err"`
	Status    string `json:"status"`
	ReadOnly  bool   `json:"readOnly"`
	Truncated bool   `json:"truncated"`
	ParentID  string `json:"parentId"`
}

// GaeaResyncResult 是 GaeaResyncEvents 的返回值（Wails 结构体返回，与
// GaeaTrajectory 同风格；Items 非 nil——Go nil 切片序列化成 null，前端按
// 数组消费会崩，与 trajectory.EmptyTrajectory 的兜底同理）。
type GaeaResyncResult struct {
	// Seq 是转发层当前最新 wire seq：前端整体替换 items 后以此为 lastSeq 续接。
	Seq int64 `json:"seq"`
	// Items 是当前会话磁盘日志折叠出的对话项（会话全量，非增量）。
	Items []GaeaResyncItem `json:"items"`
}

// GaeaResyncEvents 按前端最后收到的 wire seq 补拉当前会话对话视图。
// afterSeq 是前端最后成功处理的 seq（本次实现始终返回会话全量 items，前端
// 整体替换后以返回的 Seq 续接——全量重建对断号/会话切换/重复调用都收敛到
// 一致状态，afterSeq 保留在签名里供前端语义与后续增量优化）。
//
// 路径穿越守门：数据源恒为当前会话（gaeaCtrl().SessionPath() 派生日志路径，
// 经 session.LogPathFor 在会话目录内解析），不接受任何调用方路径参数，
// 无穿越面；与 GaeaTrajectory/GaeaSessionStats 同一守门模型。
func (a *App) GaeaResyncEvents(afterSeq int) GaeaResyncResult {
	res := GaeaResyncResult{Seq: ga.wire.last(), Items: []GaeaResyncItem{}}
	c := gaeaCtrl()
	if c == nil {
		return res
	}
	entries, err := session.ReadEntriesFor(c.SessionPath())
	if err != nil {
		// 日志缺失（新会话尚未落日志/legacy 会话）或读取失败：返回空 items
		//（非 nil），前端置空后继续依赖实时事件流，不阻塞对话。
		return res
	}
	res.Items = foldResyncItems(entries)
	return res
}

// ── 最小折叠器 ──────────────────────────────────────────────────

// 折叠器本地 payload 形状（与 session/log.go 的写入端 payload 一一对应；
// session 包未导出 payload 类型，这里按字段名解耦读取——未知字段忽略，
// 旧日志缺字段零值兜底）。
type (
	// resyncUserPayload 是 user_message 的 payload。
	resyncUserPayload struct {
		Content string `json:"content"`
	}
	// resyncTextPayload 是 text/message/assistant_message/reasoning 的 payload
	//（运行期 text/reasoning 事件只有 text 字段；assistant_message 另有
	// reasoning/tool_calls，tool_calls 不进对话视图，由 tool_dispatch 覆盖）。
	resyncTextPayload struct {
		Text      string `json:"text"`
		Reasoning string `json:"reasoning"`
	}
	// resyncToolCallPayload 是 tool_dispatch / tool_call 的 payload。
	resyncToolCallPayload struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Args     string `json:"args"`
		ReadOnly bool   `json:"readOnly,omitempty"`
		ParentID string `json:"parentId,omitempty"`
	}
	// resyncToolResultPayload 是 tool_result 的 payload。
	resyncToolResultPayload struct {
		ID        string `json:"id"`
		Name      string `json:"name,omitempty"`
		Output    string `json:"output,omitempty"`
		Err       string `json:"err,omitempty"`
		Truncated bool   `json:"truncated,omitempty"`
	}
	// resyncNoticePayload 是 notice 的 payload。
	resyncNoticePayload struct {
		Level string `json:"level,omitempty"`
		Text  string `json:"text,omitempty"`
	}
)

// foldResyncItems 把会话日志条目折叠为前端对话 Item 视图（纯函数，可独立
// 测试）。最小折叠语义（对齐 session.ProjectMessages 的会话回放口径与前端
// rebuildHistoryItems 的消费方式）：
//   - user_message                    → user 条目；
//   - text / reasoning                → 累积进当前 assistant 条目（流式 delta
//     合并，与实时渲染的单气泡形态一致）；
//   - assistant_message / message     → assistant 完整条目（全文是已累积 delta
//     的超集，原地替换内容，同投影语义）；
//   - tool_dispatch / tool_call       → tool 条目（status=running）；
//   - tool_result                     → 按 ID 合并进已建立的 tool 条目
//     （status=done/error + output/err）；无前置 dispatch 时自建完成态条目
//     （与 trajectory.FoldTrajectory 的容错一致）；
//   - notice                          → notice 条目（level info/warn）；
//   - 其余（turn 边界/usage/phase/request_header/retrying/steer/压缩/
//     subagent_message/未知 kind）一律跳过——它们不是对话气泡。
func foldResyncItems(entries []session.LogEntry) []GaeaResyncItem {
	items := make([]GaeaResyncItem, 0, len(entries))
	toolIdx := map[string]int{} // 工具调用 ID → items 下标（tool_result 合并用）
	pending := -1               // 正在累积 text/reasoning delta 的 assistant 条目下标；-1=无
	closePending := func() { pending = -1 }
	ensurePending := func(seq int64) *GaeaResyncItem {
		if pending < 0 {
			items = append(items, GaeaResyncItem{Kind: "assistant", ID: resyncID("a", seq)})
			pending = len(items) - 1
		}
		return &items[pending]
	}
	for _, e := range entries {
		switch e.Kind {
		case "user_message":
			var p resyncUserPayload
			if err := json.Unmarshal(e.Payload, &p); err != nil || p.Content == "" {
				continue
			}
			closePending()
			items = append(items, GaeaResyncItem{Kind: "user", ID: resyncID("u", e.Seq), Text: p.Content})

		case "assistant_message", "message":
			// 完整消息到达：若已累积流式 delta 则原地替换（全文是 delta 的超集，
			// 与 ProjectMessages 同语义），否则新建条目。
			var p resyncTextPayload
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				continue
			}
			it := ensurePending(e.Seq)
			it.Text = p.Text
			it.Reasoning = p.Reasoning

		case "text", "reasoning":
			// 运行期事件落盘形状（EntryFromEvent）：text 事件内容在 payload.text，
			// reasoning 事件内容在 payload.reasoning（text 字段为空）。分别累积
			// 到当前 assistant 条目的对应字段。
			var p resyncTextPayload
			if err := json.Unmarshal(e.Payload, &p); err != nil {
				continue
			}
			if e.Kind == "reasoning" {
				if p.Reasoning == "" {
					continue
				}
				it := ensurePending(e.Seq)
				it.Reasoning += p.Reasoning
				continue
			}
			if p.Text == "" {
				continue
			}
			it := ensurePending(e.Seq)
			it.Text += p.Text

		case "tool_dispatch", "tool_call":
			var p resyncToolCallPayload
			if err := json.Unmarshal(e.Payload, &p); err != nil || p.ID == "" {
				continue
			}
			items = append(items, GaeaResyncItem{
				Kind: "tool", ID: p.ID, ToolID: p.ID, Name: p.Name, Args: p.Args,
				Status: "running", ReadOnly: p.ReadOnly, ParentID: p.ParentID,
			})
			toolIdx[p.ID] = len(items) - 1

		case "tool_result":
			var p resyncToolResultPayload
			if err := json.Unmarshal(e.Payload, &p); err != nil || p.ID == "" {
				continue
			}
			status := "done"
			if p.Err != "" {
				status = "error"
			}
			if idx, ok := toolIdx[p.ID]; ok {
				it := &items[idx]
				it.Status = status
				it.Output = p.Output
				it.Err = p.Err
				it.Truncated = p.Truncated
				if p.Name != "" {
					it.Name = p.Name
				}
				continue
			}
			// 悬挂 result（日志被裁剪/仅存结果）：自建完成态条目，不丢事件。
			items = append(items, GaeaResyncItem{
				Kind: "tool", ID: p.ID, ToolID: p.ID, Name: p.Name,
				Status: status, Output: p.Output, Err: p.Err, Truncated: p.Truncated,
			})
			toolIdx[p.ID] = len(items) - 1

		case "notice":
			var p resyncNoticePayload
			if err := json.Unmarshal(e.Payload, &p); err != nil || p.Text == "" {
				continue
			}
			closePending()
			level := p.Level
			if level != "warn" {
				level = "info"
			}
			items = append(items, GaeaResyncItem{Kind: "notice", ID: resyncID("n", e.Seq), Level: level, Text: p.Text})
		}
	}
	return items
}

// resyncID 生成对话项 ID（前缀 + 日志 seq）。日志 seq 断号（torn-tail 修复/
// 外部删行）不影响唯一性——ID 只要求会话内不重复、保持事件序。
func resyncID(prefix string, seq int64) string {
	return prefix + strconv.FormatInt(seq, 10)
}
