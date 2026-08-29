// Package whisper — memory_write_job.go
// 100% 对齐 ackem memory/memoryWriteJob.ts
// 异步记忆写入队列：每会话串行化 → LLM抽取 → 摄入 → 终态化 → 主动遗忘

package whisper

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// ─── 遗忘触发词 ──────────────────────────────────────────────

var activeForgetTriggers = []string{
	"别提了", "不想聊这个", "过去了", "翻篇了", "别再说了",
	"忘了这件事", "当没说过", "跳过这个话题", "换个话题",
	"不要再问了", "别再提", "已经过去了",
}

// ─── 会话队列 ────────────────────────────────────────────────

var (
	sessionQueues   = make(map[string]*sessionQueue)
	sessionQueuesMu sync.Mutex
)

type sessionQueue struct {
	chain chan struct{}
	// pending 记「已入队、未完成」的任务数（含尚未启动的 worker goroutine）。
	// chain 令牌只在任务运行中被持有，排队中的任务对它是不可见的——drain
	// 必须靠 pending 才能等到所有已入队任务执行完，否则 shutdown 末轮
	// 抽取会被整体跳过（丢记忆）。
	pending sync.WaitGroup
}

func getSessionQueue(sessionID string) *sessionQueue {
	sessionQueuesMu.Lock()
	defer sessionQueuesMu.Unlock()
	if q, ok := sessionQueues[sessionID]; ok {
		return q
	}
	q := &sessionQueue{chain: make(chan struct{}, 1)}
	q.chain <- struct{}{}
	sessionQueues[sessionID] = q
	return q
}

// ─── MemoryWritePayload ─────────────────────────────────────

type MemoryWritePayload struct {
	SessionID       string
	TurnIndex       int
	UserMsg         string
	AssistantText   string
	L1              L1State
	L2              EmotionState
	FactStore       *FactStore
	EpisodicStore   *EpisodicStore
	KG              *KnowledgeGraph
	TotalTurns      int
	RecentExchanges []ExchangePair
	SkipIngest      bool
	AdultMode       bool
}

// MemoryWriteErrorSink 异步记忆写入错误回传（T6-5.3 可观测性）：
// sessionID 会话标识；phase 错误阶段（llm_extract/json_parse/panic 等）；
// err 原始错误。由 App 层（whisperState.recordMemoryWriteError）汇聚为
// WriteErrors 计数 + 最近错误摘要。
type MemoryWriteErrorSink func(sessionID, phase string, err error)

// EnqueueMemoryWrite 入队异步记忆写入（对齐 ackem enqueueMemoryWrite）
// 每会话串行化执行，不阻塞聊天主流程。
// sinks 为可选错误回传：任一错误路径（LLM 失败/JSON 解析失败/panic）都会
// 在记录 slog 后同步调用第一个非 nil sink，供上层计数/摘要。
func EnqueueMemoryWrite(llm LlmClient, payload MemoryWritePayload, sinks ...MemoryWriteErrorSink) {
	q := getSessionQueue(payload.SessionID)
	q.pending.Add(1) // 入队即计数：worker goroutine 尚未启动时 drain 也能等到它
	go func() {
		defer q.pending.Done() // 注册在最先 → 最后执行（token 归还之后）
		defer func() {
			if r := recover(); r != nil {
				slog.Error("whisper: memory write goroutine panic recovered", "panic", r)
				for _, s := range sinks {
					if s != nil {
						s(payload.SessionID, "panic", fmt.Errorf("memory write panic: %v", r))
					}
				}
			}
		}()
		<-q.chain
		defer func() { q.chain <- struct{}{} }()
		runMemoryWriteJob(llm, payload, sinks...)
	}()
}

func runMemoryWriteJob(llm LlmClient, payload MemoryWritePayload, sinks ...MemoryWriteErrorSink) {
	// 1. Tier B 摄入跳过检查
	if payload.SkipIngest {
		return
	}

	// 2. 运行记忆摄入管线
	ingest := NewMemoryIngestPipeline(llm)
	for _, s := range sinks { // T6-5.3：错误回传透传到摄入管线
		if s != nil {
			ingest.SetErrorSink(s)
			break
		}
	}
	privacyLevel := "normal"
	if payload.AdultMode {
		privacyLevel = resolveAdultPrivacy(payload)
	}

	ingest.AfterTurn(IngestTurnArgs{
		SessionID:       payload.SessionID,
		TurnIndex:       payload.TurnIndex,
		UserMsg:         payload.UserMsg,
		CompanionMsg:    payload.AssistantText,
		L1:              payload.L1,
		L2:              payload.L2,
		FactStore:       payload.FactStore,
		TotalTurns:      payload.TotalTurns,
		EpisodicStore:   payload.EpisodicStore,
		RecentExchanges: payload.RecentExchanges,
		KG:              payload.KG,
		Opts: IngestOptions{
			AdultPrivacyLevel: privacyLevel,
		},
	})

	// 3. 主动遗忘
	applyActiveForget(payload.UserMsg, payload.FactStore)
}

func resolveAdultPrivacy(payload MemoryWritePayload) string {
	text := strings.ToLower(payload.UserMsg + " " + payload.AssistantText)
	if strings.Contains(text, "操") || strings.Contains(text, "射") || strings.Contains(text, "插") {
		return "explicit"
	}
	return "intimate"
}

// ─── 主动遗忘 ────────────────────────────────────────────────

func applyActiveForget(userMsg string, fs *FactStore) {
	if fs == nil {
		return
	}
	triggered := false
	for _, t := range activeForgetTriggers {
		if strings.Contains(userMsg, t) {
			triggered = true
			break
		}
	}
	if !triggered {
		return
	}
	for _, f := range fs.ListActive() {
		if f.Sensitivity == "avoid" {
			continue
		}
		l := len(f.Summary)
		if l > 20 {
			l = 20
		}
		if strings.Contains(userMsg, f.Subject) || strings.Contains(userMsg, f.Summary[:l]) {
			f.Sensitivity = "avoid"
		}
	}
}

// ─── 测试辅助 ────────────────────────────────────────────────

func DrainAllMemoryWriteJobs() {
	sessionQueuesMu.Lock()
	queues := make([]*sessionQueue, 0, len(sessionQueues))
	for _, q := range sessionQueues {
		queues = append(queues, q)
	}
	sessionQueuesMu.Unlock()
	for _, q := range queues {
		// 等待所有已入队任务真正执行完（含还没拿到 chain 令牌的任务）。
		q.pending.Wait()
	}
}

func ResetMemoryWriteQueues() {
	sessionQueuesMu.Lock()
	defer sessionQueuesMu.Unlock()
	sessionQueues = make(map[string]*sessionQueue)
}
