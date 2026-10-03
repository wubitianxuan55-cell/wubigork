package contextview

// fold_timing.go — 耗时折叠（批 26 GA5-04 文件拆分，自 fold.go 原位搬移，
// 零逻辑改动）：对齐 dsh-context TimingTotals 的诚实近似版——步/轮/TTFT/
// 生成/工具五段计时 + token 事件 + 工具耗时排行。状态字段仍挂在 folding
// 结构体上（fold.go 定义），方法按域分文件；专属测试见 timing_test.go。

import (
	"encoding/json"
	"sort"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// ─── 耗时折叠（对齐 dsh-context TimingTotals 的诚实近似版） ──────────
// 日志 ts 为秒级（time.Now().Unix()），所有时长按秒差 ×1000 记 ms；只记
// 非负差值（时钟回拨/同秒日志静默跳过），无法支撑的指标留零不伪造。

// timingStepStart 记录当前步的起点（request_header ts = 请求组装发出时刻）。
func (f *folding) timingStepStart(e session.LogEntry) {
	f.stepStart = e.Ts
	f.genStart = 0
}

// timingTurnStart 开启一轮活跃时长窗口；步骤状态跨轮不残留。
func (f *folding) timingTurnStart(e session.LogEntry) {
	f.turnStart = e.Ts
	f.stepStart = 0
	f.genStart = 0
}

// timingToken 记该步首 token：首条 reasoning/text 增量。TTFT = 首 token −
// 步骤起点（request_header ts；旧迁移日志每轮只有一条 header，后续步骤退
// 化为「前一事件 ts」起算——两种口径都只计非负差值）。
func (f *folding) timingToken(e session.LogEntry) {
	if f.genStart != 0 {
		return
	}
	f.genStart = e.Ts
	if start, ok := f.timingStepBase(e.Ts); ok {
		f.timing.TTFTMs += (e.Ts - start) * 1000
	}
}

// timingAssistant 关闭一步生成：calls 计数（message/assistant_message 条数
// = 主回合模型调用次数）；gen = 消息收尾 − 首 token。该步无任何已落盘增量
// 时（纯工具调用步），消息本身即首个产物——等待整体记入 TTFT，gen 留 0。
func (f *folding) timingAssistant(e session.LogEntry) {
	f.timing.Calls++
	if f.genStart != 0 {
		if e.Ts > f.genStart {
			f.timing.GenMs += (e.Ts - f.genStart) * 1000
		}
	} else if start, ok := f.timingStepBase(e.Ts); ok {
		f.timing.TTFTMs += (e.Ts - start) * 1000
	}
	f.genStart = 0
}

// timingToolDispatch 记挂起工具调用（按 id 配对）。partial 为流式预发射
// （完整派发随后再来），与折叠口径一致跳过，避免从过早的 ts 起算。
func (f *folding) timingToolDispatch(e session.LogEntry) {
	var p toolDispatchPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil || p.Partial || p.ID == "" || p.Name == "" {
		return
	}
	if f.openTools == nil {
		f.openTools = map[string]toolOpen{}
	}
	f.openTools[p.ID] = toolOpen{name: p.Name, ts: e.Ts}
}

// timingToolResult 配对关闭工具调用：ms = 结果 − 派发，并行调用重复计
// （与 dsh 同口径，如实不去重）。迁移日志的 tool_result 没有对应 dispatch，
// 不配对不计数；中断轮的未配对派发同样不计。
func (f *folding) timingToolResult(e session.LogEntry) {
	var p toolResultPayload
	if err := json.Unmarshal(e.Payload, &p); err != nil || p.ID == "" {
		return
	}
	o, ok := f.openTools[p.ID]
	if !ok {
		return
	}
	delete(f.openTools, p.ID)
	if o.ts <= 0 || e.Ts < o.ts {
		return
	}
	ms := (e.Ts - o.ts) * 1000
	f.timing.ToolsMs += ms
	f.timing.ToolCalls++
	name := p.Name
	if name == "" {
		name = o.name
	}
	if f.toolAgg == nil {
		f.toolAgg = map[string]*toolAgg{}
	}
	agg := f.toolAgg[name]
	if agg == nil {
		agg = &toolAgg{}
		f.toolAgg[name] = agg
	}
	agg.calls++
	agg.ms += ms
}

// timingUsage 关闭一步（usage = 该次请求收尾），防止 header 缺失的日志把
// 上一步的起点残留进下一步。
func (f *folding) timingUsage(session.LogEntry) {
	f.stepStart = 0
	f.genStart = 0
}

// timingTurnDone 累加本轮活跃时长并复位步骤状态。已派发未收结果的工具
// 保留在配对表里（id 全局唯一，误配对不可能；结果若永不到来则不计）。
func (f *folding) timingTurnDone(e session.LogEntry) {
	f.addWall(f.turnStart, e.Ts)
	f.turnStart = 0
	f.stepStart = 0
	f.genStart = 0
}

// timingClose 收尾：末轮无 turn_done（完整中断，BalanceEntries 不走本路径）
// 时以最后一条日志 ts 收尾。
func (f *folding) timingClose() {
	f.addWall(f.turnStart, f.lastTs)
}

func (f *folding) addWall(start, end int64) {
	if start > 0 && end >= start {
		f.timing.WallMs += (end - start) * 1000
	}
}

// timingStepBase 返回当前步的起点 ts：优先 request_header，缺失时退化为
// 前一事件 ts（dsh「步骤起点」口径的日志可得近似）。
func (f *folding) timingStepBase(ts int64) (int64, bool) {
	switch {
	case f.stepStart > 0 && f.stepStart <= ts:
		return f.stepStart, true
	case f.stepStart == 0 && f.lastTs > 0 && f.lastTs <= ts:
		return f.lastTs, true
	}
	return 0, false
}

// timingToolsRanked 把每工具 {calls, ms} 累计按 ms 降序（并列按名称，保证
// 确定性）排为排行条目，截断 20（dsh 上限同款）。
func (f *folding) timingToolsRanked() []ToolTiming {
	if len(f.toolAgg) == 0 {
		return nil
	}
	names := make([]string, 0, len(f.toolAgg))
	for n := range f.toolAgg {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := f.toolAgg[names[i]], f.toolAgg[names[j]]
		if a.ms != b.ms {
			return a.ms > b.ms
		}
		return names[i] < names[j]
	})
	if len(names) > 20 {
		names = names[:20]
	}
	out := make([]ToolTiming, 0, len(names))
	for _, n := range names {
		agg := f.toolAgg[n]
		out = append(out, ToolTiming{Name: n, Calls: agg.calls, Ms: agg.ms})
	}
	return out
}

// nonZero 报告是否有任何可报的耗时数据（全零 → Timing 整体省略）。
func (t ContextTiming) nonZero() bool {
	return t.WallMs > 0 || t.TTFTMs > 0 || t.GenMs > 0 || t.Calls > 0 ||
		t.ToolsMs > 0 || t.ToolCalls > 0 || len(t.Tools) > 0
}
