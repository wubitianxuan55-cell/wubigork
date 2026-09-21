package agent

// ── 锚定式 token 计量（v4.381，dsh packages/llm/token-meter 蒸馏）──────────
//
// 病根：mid-turn 压力用「全量字符×全局比率」估算——比率虽自上次真实 usage
// 校准，但整个表面按线性缩放重算，比率漂一点全表面跟着漂，压缩触发时机
// （high-water）随之漂移：早触发=白砸缓存前缀，晚触发=靠溢出自愈兜底。
//
// 上游口径：压力=上次真实 usage 锚点 + 当前表面的有符号差值；per-message
// 估价缓存让「差值」只统计新增/删除/改写的消息——锚点里那部分消息的估价
// 逐字不变，剪枝/压缩重写后估计不再漂移。
//
// 落点：maybeCompact 收到真实 usage（PromptTokens>0）时落锚——此刻会话表面
// 恰为刚被 answered 的请求（新 assistant/tool 结果尚未入账），锚点无系统偏
// 差。EstimateContextTokens 有锚走「锚点+差值」，无锚（首轮前）回退裸字符
// 估算（midTurnMaybeCompact 本就等首锚后才工作）。
//
// 线程面：计量只在 run-loop goroutine 读写（maybeCompact/midTurn/
// compactIfOver 同 goroutine）；mu 护导出面（EstimateContextTokens 是导出
// 方法）。tokPerChar 读 lastUsage/session 与既有口径一致，不另加锁。

import (
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// tokenMeter 是锚定压力表：per-message 估价 memo + 锚点（真实 tokens、锚点
// 时点的估价和）。
type tokenMeter struct {
	mu         sync.Mutex
	memo       map[uint64]int
	anchored   bool
	anchorReal int
	anchorEst  int
}

func newTokenMeter() *tokenMeter { return &tokenMeter{memo: make(map[uint64]int)} }

// msgKey 是 per-message 估价缓存键：role+content+toolcalls 的 FNV-64（与
// msgChars 同一记账面——ReasoningContent 不回传 API，不入账）。
func msgKey(m provider.Message) uint64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\x00%s\x00", m.Role, m.Content)
	for _, tc := range m.ToolCalls {
		fmt.Fprintf(h, "%s\x00%s\x00", tc.Name, tc.Arguments)
	}
	return h.Sum64()
}

// msgEstLocked 返回单条消息的 token 估价。memo 化是锚定不漂移的全部前提：
// 同一消息逐次取同一值，锚点和当前和里的「未变消息」逐项相消。调用方持锁。
func (m *tokenMeter) msgEstLocked(a *AgentRunner, msg provider.Message) int {
	key := msgKey(msg)
	if v, ok := m.memo[key]; ok {
		return v
	}
	v := int(float64(msgChars(msg)) * a.tokPerChar())
	m.memo[key] = v
	return v
}

// sumEstLocked 返回消息表面的估价和。调用方持锁。
func (m *tokenMeter) sumEstLocked(a *AgentRunner, msgs []provider.Message) int {
	total := 0
	for _, msg := range msgs {
		total += m.msgEstLocked(a, msg)
	}
	return total
}

// anchor 在真实 usage 到手时落锚。
func (m *tokenMeter) anchor(a *AgentRunner, promptTokens int, msgs []provider.Message) {
	if promptTokens <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.anchorReal = promptTokens
	m.anchorEst = m.sumEstLocked(a, msgs)
	m.anchored = true
}

// estimate 返回当前压力：有锚=锚点+差值（差值可负——删得多估得狠——压力
// 钳非负）；无锚=裸字符估算（旧口径，行为不变）。
func (m *tokenMeter) estimate(a *AgentRunner) int {
	if a.session == nil {
		return 0
	}
	msgs := a.session.Messages
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.anchored {
		return int(float64(charsOfMessages(msgs)) * a.tokPerChar())
	}
	p := m.anchorReal + (m.sumEstLocked(a, msgs) - m.anchorEst)
	if p < 0 {
		p = 0
	}
	return p
}
