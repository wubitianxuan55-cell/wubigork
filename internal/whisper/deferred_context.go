// Package whisper — deferred_context.go
// 100% 对齐 ackem chat/deferredContext.ts
// 延迟上下文丰富：Wave0 启动异步记忆摄取，Wave1+ 等待结果

package whisper

import (
	"sync"
	"time"
)

// DeferredEnrichArgs 延迟上下文丰富的参数
type DeferredEnrichArgs struct {
	TurnID            string
	Msg               string
	SessionID         string
	TurnIndex         int
	MemoryBudgetChars int
	DataRoot          string
	AdultMode         bool
}

// EnrichResult 丰富结果
type EnrichResult struct {
	TierBBlock string
}

type deferredEntry struct {
	result   EnrichResult
	done     chan struct{}
	started  bool
}

var (
	deferredStore   = make(map[string]*deferredEntry)
	deferredStoreMu sync.Mutex
)

// DeferredEnrichFunc 执行丰富逻辑的函数签名，由外部注入
type DeferredEnrichFunc func(args DeferredEnrichArgs) (EnrichResult, error)

// StartDeferredEnrich 开始异步延迟上下文丰富
// 100% 对齐 ackem deferredContext.ts startDeferredEnrich
func StartDeferredEnrich(args DeferredEnrichArgs, enrichFn DeferredEnrichFunc) {
	deferredStoreMu.Lock()
	entry := &deferredEntry{
		done:    make(chan struct{}),
		started: true,
	}
	deferredStore[args.TurnID] = entry
	deferredStoreMu.Unlock()

	go func() {
		result, err := enrichFn(args)
		if err != nil {
			result = EnrichResult{}
		}
		deferredStoreMu.Lock()
		entry.result = result
		deferredStoreMu.Unlock()
		close(entry.done)
	}()
}

// AwaitDeferredEnrich 等待延迟上下文丰富完成（带超时上限）
// 100% 对齐 ackem deferredContext.ts awaitDeferredEnrich
func AwaitDeferredEnrich(turnID string, timeout time.Duration) string {
	deferredStoreMu.Lock()
	entry, ok := deferredStore[turnID]
	deferredStoreMu.Unlock()
	if !ok {
		return ""
	}

	select {
	case <-entry.done:
		deferredStoreMu.Lock()
		result := entry.result.TierBBlock
		deferredStoreMu.Unlock()
		return result
	case <-time.After(timeout):
		return ""
	}
}

// ClearDeferredEnrich 清理延迟上下文
// 100% 对齐 ackem deferredContext.ts clearDeferredEnrich
func ClearDeferredEnrich(turnID string) {
	deferredStoreMu.Lock()
	delete(deferredStore, turnID)
	deferredStoreMu.Unlock()
}

// DefaultEnrichTimeout 默认等待超时（对齐 ackem ENRICH_WAIT_MS = 800ms）
const DefaultEnrichTimeout = 800 * time.Millisecond
