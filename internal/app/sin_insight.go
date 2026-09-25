package app

// sin_insight.go — 原罪板块的「轨迹 / 上下文」只读看板数据面（v4.412）：
// 把 sin 话题（chat.store 消息 + extra 工具轨迹）折叠成与办公板块同构的
// trajectory.Trajectory / contextview.ContextTimeline，前端 TrajectoryView /
// ContextView 原组件复用（fetchTrajectory / fetchTimeline 自定义数据源）。
//
// 口径（与办公折叠管线的差异，全部诚实标注不伪造）：
//   - sin 无 usage 上报 → 请求一律 Estimated=true，token 按 bytes×0.25 估算
//     （与 contextview/agent 同一口径）；
//   - 窗口大小未知 → Window=0（前端水位百分比显示「—」而非 0%）；
//   - 每次请求 = system + 角色卡/底稿注入 + 截至该轮的全部落库内容
//     （sinStreamRound 语义：buildSinUserPrompt 带全量前情），逐请求累计；
//   - 无压缩/剪枝/文件活动事件 → events/files 恒空。

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/gaea/contextview"
	"github.com/gaea/gaea/internal/gaea/trajectory"
)

// sinEstTokens 按仓库既有口径估算文本 token（bytes×0.25）。
func sinEstTokens(s string) int64 {
	return int64(float64(len(s)) * 0.25)
}

// sinBrief 单行截断预览（与 contextview briefOf 同语义：取首行、按 rune 截断）。
func sinBrief(s string, max int) string {
	one := s
	if i := strings.IndexByte(one, '\n'); i >= 0 {
		one = one[:i]
	}
	r := []rune(one)
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return one
}

// sinTs 解析 chat.store 的本地时间戳（"2006-01-02 15:04:05"）为 unix 秒；
// 空/坏值返回 0（前端显示「—」，不伪造时间）。
func sinTs(s string) int64 {
	if s == "" {
		return 0
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t.Unix()
	}
	return 0
}

// sinMsgExtra 是助手消息 extra 的读取投影（写入端见 sin_handler extraMap）。
type sinMsgExtra struct {
	Reasoning     string            `json:"reasoning"`
	Tools         []sinToolTrace    `json:"tools"`
	Illustrations map[string]string `json:"illustrations"`
}

func parseSinMsgExtra(raw string) sinMsgExtra {
	var out sinMsgExtra
	if s := strings.TrimSpace(raw); s != "" {
		_ = json.Unmarshal([]byte(s), &out) // 坏 extra 按空处理（读侧容错）
	}
	return out
}

// sinNodeSeq 消息内节点的确定性 seq 槽位（view 与 detail 共用同一编址）：
// user=base+0、tool=base+1..8（超出并入第 8 槽，轮上限 4 不会触达）、
// assistant=base+9；base=消息 ID×10（消息 ID 全局自增，编址稳定无碰撞）。
// system/底稿/角色卡三个非消息节点用 1/2/3 固定槽（不可展开详情）。
const (
	sinSeqSystem = int64(1)
	sinSeqDraft  = int64(2)
	sinSeqCast   = int64(3)
)

func sinNodeSeq(msgID int64, slot int) int64 {
	return msgID*10 + int64(slot)
}

// sinInsight 是一次折叠的全部中间产物（Trajectory 与 ContextTimeline 同源
// 折叠，保证两看板对同一话题的内容一致）。
type sinInsight struct {
	msgs    []chat.Message
	tools   map[int64][]sinToolTrace // 消息 ID → 工具轨迹
	illus   map[int64]int            // 消息 ID → 插图数
	system  string                   // 系统提示词现值
	draft   string                   // 底稿（便签+大纲）块现值
	castBlk string                   // 角色卡块现值
}

func (a *App) foldSinInsight(topicID string) (*sinInsight, error) {
	if err := a.sinTopicGuard(topicID); err != nil {
		return nil, err
	}
	msgs, err := a.chatStore.ListMessages(topicID)
	if err != nil {
		return nil, err
	}
	ins := &sinInsight{
		msgs:  msgs,
		tools: map[int64][]sinToolTrace{},
		illus: map[int64]int{},
	}
	for _, m := range msgs {
		ex := parseSinMsgExtra(m.Extra)
		if len(ex.Tools) > 0 {
			ins.tools[m.ID] = ex.Tools
		}
		ins.illus[m.ID] = len(ex.Illustrations)
	}
	ins.system = sinSystemPrompt()
	draft := a.sinDraftForPrompt(topicID)
	ins.draft = sinDraftBlock(draft)
	ins.castBlk = sinCastBlock(a.sinCastCharacters(topicID))
	return ins, nil
}

// foldSinTrajectory 把落库消息折叠成轨迹轮次：user 起轮，工具轨迹随助手
// 消息 extra 还原在回复前（与真实时序一致）；孤儿助手消息并入当前轮或轮间。
func foldSinTrajectory(ins *sinInsight) trajectory.Trajectory {
	out := trajectory.Trajectory{Ok: true, Turns: []trajectory.Turn{}}
	var cur *trajectory.Turn
	flush := func() {
		if cur != nil {
			out.Turns = append(out.Turns, *cur)
			cur = nil
		}
	}
	for _, m := range ins.msgs {
		ts := sinTs(m.CreatedAt)
		if m.Role == "user" {
			flush()
			cur = &trajectory.Turn{Turn: len(out.Turns) + 1, StartedAt: ts, Records: []trajectory.Record{}}
			cur.Records = append(cur.Records, trajectory.Record{
				Seq: sinNodeSeq(m.ID, 0), Kind: "user", Ts: ts,
				User: &trajectory.UserRec{Text: m.Content},
			})
			continue
		}
		ex := parseSinMsgExtra(m.Extra)
		recs := make([]trajectory.Record, 0, len(ex.Tools)+1)
		for i, t := range ex.Tools {
			slot := i + 1
			if slot > 8 {
				slot = 8
			}
			status := "ok"
			if t.Error != "" {
				status = "error"
			}
			recs = append(recs, trajectory.Record{
				Seq: sinNodeSeq(m.ID, slot), Kind: "tool", Ts: ts,
				DurationMs: t.ElapsedMS,
				Tool: &trajectory.ToolRec{
					ID: t.ID, Name: t.Name, Args: t.Args, Output: t.Output,
					Err: t.Error, ReadOnly: t.ReadOnly, Status: status,
				},
			})
		}
		recs = append(recs, trajectory.Record{
			Seq: sinNodeSeq(m.ID, 9), Kind: "assistant", Ts: ts,
			Assistant: &trajectory.AssistantRec{Text: m.Content, Reasoning: ex.Reasoning},
		})
		if cur == nil {
			out.BetweenTurns = append(out.BetweenTurns, recs...)
			continue
		}
		cur.Records = append(cur.Records, recs...)
		end := recs[len(recs)-1]
		cur.End = &trajectory.TurnEnd{Seq: end.Seq, Ts: end.Ts}
		if end.Ts > cur.StartedAt {
			cur.DurationMs = (end.Ts - cur.StartedAt) * 1000
		}
	}
	flush()
	return out
}

// foldSinContext 把落库消息折叠成上下文构成快照：逐请求累计（每轮请求 =
// system + 注入 + 截至该轮全部落库内容，与 sinStreamRound 的 buildSinUserPrompt
// 全量前情语义对齐），current = 最后一根请求柱；节点与 brief 锚点同编址。
func foldSinContext(ins *sinInsight) contextview.ContextTimeline {
	tl := contextview.EmptyTimeline()
	tl.Window = 0 // 窗口大小未知：前端水位显示「—」，不伪造 0%

	sysTok := sinEstTokens(ins.system)
	injTok := sinEstTokens(ins.draft) + sinEstTokens(ins.castBlk)

	// 节点：system/注入（现值快照，与办公 request_header 聚合节点同位——
	// 不提供详情展开）+ 逐消息（user/assistant/tool）。
	tl.Nodes = append(tl.Nodes,
		contextview.SurfaceNode{Seq: sinSeqSystem, Cat: "system", Tokens: sysTok, Text: sinBrief(ins.system, 120)},
	)
	if d := strings.TrimSpace(ins.draft); d != "" {
		tl.Nodes = append(tl.Nodes, contextview.SurfaceNode{Seq: sinSeqDraft, Cat: "inject", Tokens: sinEstTokens(ins.draft), Text: sinBrief(d, 120)})
	}
	if c := strings.TrimSpace(ins.castBlk); c != "" {
		tl.Nodes = append(tl.Nodes, contextview.SurfaceNode{Seq: sinSeqCast, Cat: "inject", Tokens: sinEstTokens(ins.castBlk), Text: sinBrief(c, 120)})
	}

	var cumUser, cumAsst, cumTool int64
	var toolCalls int
	toolsMs := map[string]struct {
		calls int
		ms    int64
	}{}
	images := 0
	turn := 0
	prevCat := contextview.Category{}
	prevNodes := len(tl.Nodes)
	first := true
	var curUser chat.Message

	// appendRequest 追加本轮请求柱：累计构成 + 对比上一步（首个请求基线=空；
	// system/inject 取现值恒定，天然不出现在 delta 里）+ user 侧 brief 锚点
	// （assistant 侧由调用方回填）。
	appendRequest := func(ts int64, cat contextview.Category) {
		delta := contextview.RequestDelta{
			Items:  int64(len(tl.Nodes) - prevNodes),
			First:  first,
			Tokens: cat.Total(),
		}
		if !first {
			delta.Tokens = cat.Total() - prevCat.Total()
			byCat := make([]contextview.CatDelta, 0, 3)
			for _, p := range []struct {
				cat        string
				prev, curt int64
			}{
				{"user", prevCat.User, cat.User},
				{"assistant", prevCat.Assistant, cat.Assistant},
				{"tool", prevCat.Tool, cat.Tool},
			} {
				if d := p.curt - p.prev; d != 0 {
					byCat = append(byCat, contextview.CatDelta{Cat: p.cat, Tokens: d})
				}
			}
			sort.Slice(byCat, func(i, j int) bool {
				pi, pj := abs64(byCat[i].Tokens), abs64(byCat[j].Tokens)
				if pi != pj {
					return pi > pj
				}
				return byCat[i].Cat < byCat[j].Cat
			})
			delta.ByCat = byCat
		}
		tl.Requests = append(tl.Requests, contextview.RequestRecord{
			Seq: int64(len(tl.Requests) + 1), Ts: ts, Turn: turn, Step: 1,
			Category: cat, Estimated: true, Delta: &delta,
			BriefUser:    sinBrief(curUser.Content, 48),
			BriefUserSeq: sinNodeSeq(curUser.ID, 0),
		})
		prevCat, prevNodes, first = cat, len(tl.Nodes), false
	}

	for _, m := range ins.msgs {
		ts := sinTs(m.CreatedAt)
		switch m.Role {
		case "user":
			turn++
			curUser = m
			cumUser += sinEstTokens(m.Content)
			tl.Nodes = append(tl.Nodes, contextview.SurfaceNode{
				Seq: sinNodeSeq(m.ID, 0), Cat: "user", Tokens: sinEstTokens(m.Content), Text: sinBrief(m.Content, 120),
			})
		case "assistant":
			ex := parseSinMsgExtra(m.Extra)
			images += len(ex.Illustrations)
			for i, t := range ex.Tools {
				slot := i + 1
				if slot > 8 {
					slot = 8
				}
				tok := sinEstTokens(t.Output)
				cumTool += tok
				toolCalls++
				agg := toolsMs[t.Name]
				agg.calls++
				agg.ms += t.ElapsedMS
				toolsMs[t.Name] = agg
				tl.Nodes = append(tl.Nodes, contextview.SurfaceNode{
					Seq: sinNodeSeq(m.ID, slot), Cat: "tool", Tokens: tok,
					Text: sinBrief(t.Output, 120), Tool: t.Name, Err: t.Error != "",
				})
			}
			cumAsst += sinEstTokens(m.Content)
			tl.Nodes = append(tl.Nodes, contextview.SurfaceNode{
				Seq: sinNodeSeq(m.ID, 9), Cat: "assistant", Tokens: sinEstTokens(m.Content), Text: sinBrief(m.Content, 120),
			})
			if turn == 0 {
				continue // 孤儿助手消息：计内容不计请求柱（无对应回合起点）
			}
			cat := contextview.Category{
				System: sysTok, Inject: injTok,
				User: cumUser, Assistant: cumAsst, Tool: cumTool,
			}
			appendRequest(ts, cat)
			last := len(tl.Requests) - 1
			tl.Requests[last].BriefResp = sinBrief(m.Content, 48)
			tl.Requests[last].BriefRespSeq = sinNodeSeq(m.ID, 9)
		}
	}

	if n := len(tl.Requests); n > 0 {
		tl.Current = tl.Requests[n-1].Category
		tl.Stats = contextview.Stats{Turns: turn, Steps: n, ToolCalls: toolCalls, Images: images}
	}
	if len(toolsMs) > 0 {
		names := make([]string, 0, len(toolsMs))
		for name := range toolsMs {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			a, b := toolsMs[names[i]], toolsMs[names[j]]
			if a.ms != b.ms {
				return a.ms > b.ms
			}
			return names[i] < names[j]
		})
		var ms int64
		timing := &contextview.ContextTiming{ToolCalls: toolCalls, Tools: make([]contextview.ToolTiming, 0, len(names))}
		for _, name := range names {
			agg := toolsMs[name]
			ms += agg.ms
			timing.Tools = append(timing.Tools, contextview.ToolTiming{Name: name, Calls: agg.calls, Ms: agg.ms})
		}
		timing.ToolsMs = ms
		tl.Timing = timing
	}
	return tl
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// clampSinDetail 详情全文截断（对齐 contextview clampDetail 语义：超上限截断
// 并诚实标注 Clamped，绝不静默丢弃后半段）。
func clampSinDetail(s string) (string, bool) {
	if len(s) <= contextview.MaxDetailBytes {
		return s, false
	}
	// 按 rune 边界回退，避免截出半个多字节字符。
	b := s[:contextview.MaxDetailBytes]
	for len(b) > 0 && !isRuneStart(b[len(b)-1]) {
		b = b[:len(b)-1]
	}
	return b, true
}

func isRuneStart(b byte) bool {
	return b&0xC0 != 0x80
}

func sinCountLines(s string) int {
	n := strings.Count(s, "\n") + 1
	if strings.HasSuffix(s, "\n") {
		n--
	}
	if n < 1 {
		n = 1
	}
	return n
}

// ── App 绑定（原罪看板三口：轨迹 / 上下文 / 节点详情） ─────────────

// SinTrajectory 返回故事的轨迹时间线（与办公 GaeaTrajectory 同构，前端
// TrajectoryView 复用渲染；话题非法时报错，空故事返回空轨迹）。
func (a *App) SinTrajectory(topicID string) (trajectory.Trajectory, error) {
	ins, err := a.foldSinInsight(topicID)
	if err != nil {
		return trajectory.EmptyTrajectory(), err
	}
	return foldSinTrajectory(ins), nil
}

// SinContextView 返回故事的上下文构成快照（估算口径：Estimated 全标注、
// Window=0；与办公 GaeaContextView 同构，前端 ContextView 复用渲染）。
func (a *App) SinContextView(topicID string) (contextview.ContextTimeline, error) {
	ins, err := a.foldSinInsight(topicID)
	if err != nil {
		return contextview.EmptyTimeline(), err
	}
	return foldSinContext(ins), nil
}

// SinContextNodeDetail 返回上下文浏览器节点的「完整调用」详情：user/
// assistant 取消息全文，tool 取 extra.tools 里的参数与输出；system/注入
// 节点与办公同口径不提供详情（报错，前端诚实显示不可展开）。
func (a *App) SinContextNodeDetail(topicID string, seq int64) (contextview.NodeDetail, error) {
	ins, err := a.foldSinInsight(topicID)
	if err != nil {
		return contextview.NodeDetail{}, err
	}
	for _, m := range ins.msgs {
		switch {
		case m.Role == "user" && sinNodeSeq(m.ID, 0) == seq:
			text, clamped := clampSinDetail(m.Content)
			return contextview.NodeDetail{
				Seq: seq, Kind: "user_message", Ts: sinTs(m.CreatedAt),
				Text: text, Lines: sinCountLines(text), Clamped: clamped,
			}, nil
		case m.Role == "assistant" && sinNodeSeq(m.ID, 9) == seq:
			text, clamped := clampSinDetail(m.Content)
			return contextview.NodeDetail{
				Seq: seq, Kind: "assistant_message", Ts: sinTs(m.CreatedAt),
				Text: text, Lines: sinCountLines(text), Clamped: clamped,
			}, nil
		case m.Role == "assistant":
			tools := parseSinMsgExtra(m.Extra).Tools
			for i, t := range tools {
				slot := i + 1
				if slot > 8 {
					slot = 8
				}
				if sinNodeSeq(m.ID, slot) != seq {
					continue
				}
				out, clamped := clampSinDetail(t.Output)
				return contextview.NodeDetail{
					Seq: seq, Kind: "tool_result", Ts: sinTs(m.CreatedAt),
					Tool: t.Name, Args: t.Args, Output: out, Err: t.Error,
					Truncated: clamped, Lines: sinCountLines(out), Clamped: clamped,
				}, nil
			}
		}
	}
	return contextview.NodeDetail{}, errSinNodeDetail
}

var errSinNodeDetail = errors.New("未找到可展开的节点")
