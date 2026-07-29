// Package whisper — memory_write_job.go
// 100% 对齐 ackem memory/memoryWriteJob.ts
// 异步记忆写入队列：每会话串行化 → LLM抽取 → 摄入 → 终态化 → 主动遗忘

package whisper

import (
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

// EnqueueMemoryWrite 入队异步记忆写入（对齐 ackem enqueueMemoryWrite）
// 每会话串行化执行，不阻塞聊天主流程
func EnqueueMemoryWrite(llm LlmClient, payload MemoryWritePayload) {
	q := getSessionQueue(payload.SessionID)
	go func() {
		<-q.chain
		defer func() { q.chain <- struct{}{} }()
		runMemoryWriteJob(llm, payload)
	}()
}

func runMemoryWriteJob(llm LlmClient, payload MemoryWritePayload) {
	// 1. Tier B 摄入跳过检查
	if payload.SkipIngest {
		return
	}

	// 2. 运行记忆摄入管线
	ingest := NewMemoryIngestPipeline(llm)
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
		<-q.chain
		q.chain <- struct{}{}
	}
}

func ResetMemoryWriteQueues() {
	sessionQueuesMu.Lock()
	defer sessionQueuesMu.Unlock()
	sessionQueues = make(map[string]*sessionQueue)
}
