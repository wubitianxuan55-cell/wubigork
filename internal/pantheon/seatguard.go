package pantheon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 席位运行护栏（2026-11-06 卡死根因实证后加）：
//
// 众神殿把「一席陈词」外包给完整 agent CLI，而 CLI 是自主循环：它会为了核验
// 一句「版本号是多少」连开几十次工具。2026-11-06 实弹会话
// pan_1793963799148_6aecf8f40f0a 里 DSH 一席四轮共发起 209 次工具调用
// （104 次 pwsh + 69 次 read + 30 次 grep），单轮 87 次，四轮耗时
// 368s/147s/316s/403s——用户看到的就是「卡死」。更糟的是 DSH 自己的工具
// 超时是 30s，一条长命令就白等半分钟，而外面只有 8 分钟总上限兜底。
//
// 所以这里补三道确定性护栏。它们不改变「席位可以核验」的语义，只规定上限：
//   - 工具预算：一次调用最多 seatToolBudget 次工具（按工具 ID 去重），超了
//     就终止本次调用并如实报错——不是静默截断，也不触发「空结果自动重试」。
//   - 无进展看门狗：连续 seatStallLimit 没有任何输出流向（正文/thinking/
//     status/工具都算）就终止。工具超时、审批卡住、CLI 静默这类「进程还活着
//     但什么都没发生」的病，全部由这一道闸收口；长推理有事件持续喂活性，
//     不会被误杀。
//   - 开局静默闸：进程一直在动、却迟迟不写正文（纯探索）也要收口；这条闸
//     只认正文，工具与 thinking 不算。
var (
	// 变量而非常量：单测要把它压到毫秒级才能在 CI 里跑看门狗。
	//
	// 2026-11-06 二刀修正：初版把这个值定成 16，事后用 138 次实弹调用复核发现
	// **工具数与耗时几乎无关**——≤40 次工具的调用平均 211s，>60 次的平均 242s，
	// 而中位数是 34 次（116/138 次超过 16）。按 16 硬停会把大多数正常核验判死，
	// 却拦不住真正慢的那些。所以它降级为「明显跑偏的天花板」（历史 P99≈151），
	// 时间约束交给下面两道闸与 8 分钟总上限；日常核验靠提示词纪律（≤8 次）。
	seatToolBudget = 120
	// 120s 的依据：DSH 单次工具调用自身超时 30s，长推理期会有 thinking/status
	// 事件持续喂活性；连续两分钟连一个事件都没有，只可能是审批/管道/进程层
	// 卡住，继续等下去就是用户眼里的「卡死」。
	seatStallLimit = 120 * time.Second
	// 开局静默上限：进程一直在动（工具、thinking 都有事件），但迟迟不写正文。
	// 实弹里 DSH 一轮 87 次工具调用、6 分钟才开口，用户看到的就是「卡死」。
	// 5 分钟没开始写正文即终止：真需要核验的席位通常早就在边查边写。
	seatFirstTextLimit = 5 * time.Minute
)

var (
	// errSeatToolBudget 与 errSeatStalled 都不是「空结果」：上层 runAgentWithRetry
	// 的自动重试只认「返回空结果」，这两类预算/活性失败重试没有意义（同样的
	// 行为只会再撞一次墙），必须直接进 agent_error。
	errSeatToolBudget  = errors.New("席位工具调用超出议事预算，已终止本轮")
	errSeatStalled     = errors.New("席位长时间没有任何进展，已终止本轮")
	errSeatNoFirstText = errors.New("席位长时间没有开始输出正文，已终止本轮")
)

// seatGuard 是一次外部 CLI 调用的运行护栏。cancel 来自带超时的 runCtx，
// 护栏只负责「什么时候该提前取消」，进程树回收仍走 runner 里既有的
// cmd.Cancel（taskkill /T）。
type seatGuard struct {
	cancel    context.CancelFunc
	budget    int
	stall     time.Duration
	firstText time.Duration
	activity  chan struct{}
	stop      chan struct{}
	done      chan struct{}

	mu      sync.Mutex
	seen    map[string]bool
	count   int
	sawText atomic.Bool
	toolHit atomic.Bool
	stalled atomic.Bool
	noText  atomic.Bool
}

func newSeatGuard(cancel context.CancelFunc, budget int, stall, firstText time.Duration) *seatGuard {
	guard := &seatGuard{
		cancel:    cancel,
		budget:    budget,
		stall:     stall,
		firstText: firstText,
		activity:  make(chan struct{}, 1),
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
		seen:      make(map[string]bool, budget),
	}
	go guard.watch()
	return guard
}

// NoteActivity 记一次「席位还在动」。非阻塞：护栏自己的节奏不能拖慢流式输出。
func (g *seatGuard) NoteActivity() {
	if g == nil {
		return
	}
	select {
	case g.activity <- struct{}{}:
	default:
	}
}

// NoteText 记一次「正文开播」：开局静默闸只认正文，工具与 thinking 不算。
func (g *seatGuard) NoteText() {
	if g == nil {
		return
	}
	g.sawText.Store(true)
	g.NoteActivity()
}

// NoteTool 记一次工具事件并按 ID 去重计数；超出预算时取消整次调用。
// 同一工具的 started/completed 两个事件只算一次——否则预算会被事件形态
// 而不是真实调用次数决定。
func (g *seatGuard) NoteTool(id string) {
	if g == nil {
		return
	}
	g.NoteActivity()
	id = strings.TrimSpace(id)
	g.mu.Lock()
	if id != "" && g.seen[id] {
		g.mu.Unlock()
		return
	}
	if id != "" {
		g.seen[id] = true
	}
	g.count++
	count := g.count
	g.mu.Unlock()
	if g.budget > 0 && count > g.budget && g.toolHit.CompareAndSwap(false, true) {
		g.cancel()
	}
}

// Err 返回护栏先于引擎失败的事实。必须优先于 ctx.Err() 判读：两种护栏都是
// 通过 cancel 生效的，先判 ctx 会把「预算耗尽」误报成「用户取消」。
func (g *seatGuard) Err() error {
	if g == nil {
		return nil
	}
	if g.toolHit.Load() {
		g.mu.Lock()
		count := g.count
		g.mu.Unlock()
		return fmt.Errorf("%w（%d 次工具 > 上限 %d）", errSeatToolBudget, count, g.budget)
	}
	if g.stalled.Load() {
		return fmt.Errorf("%w（连续 %s 无输出、无工具事件）", errSeatStalled, g.stall)
	}
	if g.noText.Load() {
		return fmt.Errorf("%w（%s 内没有出现任何正文）", errSeatNoFirstText, g.firstText)
	}
	return nil
}

// Close 停止看门狗并等它退出，避免每次调用漏一个 goroutine。
func (g *seatGuard) Close() {
	if g == nil {
		return
	}
	select {
	case <-g.stop:
	default:
		close(g.stop)
	}
	<-g.done
}

func (g *seatGuard) watch() {
	defer close(g.done)
	timer := time.NewTimer(g.stall)
	defer timer.Stop()
	var firstText <-chan time.Time
	if g.firstText > 0 {
		firstTimer := time.NewTimer(g.firstText)
		defer firstTimer.Stop()
		firstText = firstTimer.C
	}
	for {
		select {
		case <-g.stop:
			return
		case <-g.activity:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(g.stall)
		case <-firstText:
			// 开局静默闸只开一次：正文已开播就不再管。
			firstText = nil
			if !g.sawText.Load() {
				g.noText.Store(true)
				g.cancel()
				return
			}
		case <-timer.C:
			// 只有「确实还要继续干」的取消才值得记：ctx 已结束时不再补刀。
			g.stalled.Store(true)
			g.cancel()
			return
		}
	}
}

// parseSeatRunStats 从外部 CLI 的事件流里抽取引擎自报的可核对账目。
//
// 两种引擎两种形状（2026-11-06 实弹）：
//
// DSH headless --json：逐步上报
//
//	{"type":"session","sessionId":"session-…"}
//	{"type":"status","phase":"step_end","usage":{"inputTokens":606,"cacheReadTokens":5632,…}}
//
// ZCode --output-format stream-json：收尾一次性上报
//
//	{"type":"result","sessionId":"sess_…","response":"…",
//	 "usage":{"modelRequestCount":2,"inputTokens":67412,"cacheReadTokens":37504,…}}
//
// 只认引擎自己写的账，不做估算。两种形状的 token 都按「本次调用合计」语义：
// DSH 逐步累加，ZCode 取 final result（同一调的 turn.completed 也有 usage，
// 不能重复计）。codex 只报 thread.started 的会话 ID（A5② 续接链靠它），
// token 账目暂无同形载体，保持零值。
func parseSeatRunStats(raw string) SeatUsage {
	var usage SeatUsage
	sawStepUsage := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		switch strings.ToLower(traceString(obj["type"])) {
		case "session":
			if id := traceString(obj["sessionId"]); id != "" {
				usage.EngineSessionID = id
			}
		case "thread.started":
			// codex exec --json 的会话 ID 载体（2026-11-07 A5② 探针实锤：
			// {"type":"thread.started","thread_id":"01a1b609-…"}）。续接链
			// 靠它落回 Participant.EngineSessionID。
			if id := traceString(obj["thread_id"]); id != "" {
				usage.EngineSessionID = id
			}
		case "status":
			if !strings.EqualFold(traceString(obj["phase"]), "step_end") {
				continue
			}
			payload, _ := obj["usage"].(map[string]interface{})
			if payload == nil {
				continue
			}
			sawStepUsage = true
			usage.Steps++
			usage.InputTokens += traceInt64(payload["inputTokens"])
			usage.CacheReadTokens += traceInt64(payload["cacheReadTokens"])
			usage.OutputTokens += traceInt64(payload["outputTokens"])
		case "result":
			if id := traceString(obj["sessionId"]); id != "" {
				usage.EngineSessionID = id
			}
			if sawStepUsage {
				// DSH 不走 result；万一将来有，逐步账已经是权威，不重复计。
				continue
			}
			payload, _ := obj["usage"].(map[string]interface{})
			if payload == nil {
				continue
			}
			// 口径归一（2026-11-06 实弹逮住）：ZCode/Anthropic 风格的
			// inputTokens 是「含缓存的 prompt 总量」（input 67412 里 37504 是
			// cacheRead，totalTokens=67721=67412+309 可证）；DSH 的 inputTokens
			// 是「未命中部分」。统一存成「未命中 / 命中」两栏，命中率分母才不会
			// 被算两遍（否则 55.6% 会被算成 35.7%）。
			input := traceInt64(payload["inputTokens"])
			cached := traceInt64(payload["cacheReadTokens"])
			if cached > input {
				cached = input
			}
			usage.InputTokens = input - cached
			usage.CacheReadTokens = cached
			usage.OutputTokens = traceInt64(payload["outputTokens"])
			// 步数用「模型请求数」而不是工具数：决定墙钟的是模型往返次数。
			if steps := traceInt64(payload["modelRequestCount"]); steps > 0 {
				usage.Steps = int(steps)
			}
		}
	}
	return usage
}

// traceInt64 容忍 float64（encoding/json 默认数字形态）与字符串两种上报形态。
func traceInt64(value interface{}) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return parsed
	case string:
		var parsed json.Number = json.Number(strings.TrimSpace(typed))
		out, _ := parsed.Int64()
		return out
	default:
		return 0
	}
}
