// Package whisper — turn_bubble_queue.go
// 100% 对齐 ackem chat/turnBubbleQueue.ts
// 跨波气泡排序队列：Wave N 任务仅在 Wave N-1 生成完成且队列排空后执行

package whisper

import (
	"sync"
)

// BubbleTask 气泡展示任务
type BubbleTask func() error

// TurnBubbleQueue 跨波并行生成的有序展示队列
// 100% 对齐 ackem turnBubbleQueue.ts TurnBubbleQueue
type TurnBubbleQueue struct {
	mu                 sync.Mutex
	pending            map[int][]BubbleTask
	generationComplete map[int]bool
	displayCursor      int
	draining           bool
	aborted            bool
	displayDone        map[int]bool
	waveCount          int
	waiters            []chan struct{}
	allDone            chan struct{}
	allDoneOnce        sync.Once
}

// NewTurnBubbleQueue 创建气泡队列
func NewTurnBubbleQueue(waveCount int) *TurnBubbleQueue {
	if waveCount < 1 {
		waveCount = 1
	}
	q := &TurnBubbleQueue{
		pending:            make(map[int][]BubbleTask),
		generationComplete: make(map[int]bool),
		displayDone:        make(map[int]bool),
		waveCount:          waveCount,
		allDone:            make(chan struct{}),
	}
	return q
}

// Enqueue 入队一个展示任务
func (q *TurnBubbleQueue) Enqueue(waveIndex int, task BubbleTask) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.aborted {
		return
	}
	q.pending[waveIndex] = append(q.pending[waveIndex], task)
	go q.scheduleDrain()
}

// MarkGenerationComplete 标记某波的 LLM 生成已完成
func (q *TurnBubbleQueue) MarkGenerationComplete(waveIndex int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.aborted {
		return
	}
	q.generationComplete[waveIndex] = true
	go q.scheduleDrain()
}

// Abort 终止队列
func (q *TurnBubbleQueue) Abort() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.aborted = true
	q.pending = make(map[int][]BubbleTask)
	q.notifyWaiters()
}

// WaitUntilDisplayed 等待所有波次展示完成
func (q *TurnBubbleQueue) WaitUntilDisplayed() {
	q.mu.Lock()
	if q.aborted {
		q.mu.Unlock()
		return
	}
	if q.isAllDisplayed() {
		q.mu.Unlock()
		return
	}

	waiter := make(chan struct{})
	q.waiters = append(q.waiters, waiter)
	q.mu.Unlock()

	select {
	case <-waiter:
	case <-q.allDone:
	}
}

// ─── 内部逻辑 ──────────────────────────────────────────────────

func (q *TurnBubbleQueue) isAllDisplayed() bool {
	for i := 0; i < q.waveCount; i++ {
		if !q.displayDone[i] {
			return false
		}
	}
	return true
}

func (q *TurnBubbleQueue) notifyWaiters() {
	if !q.isAllDisplayed() {
		return
	}
	q.allDoneOnce.Do(func() {
		close(q.allDone)
	})
	for _, w := range q.waiters {
		select {
		case w <- struct{}{}:
		default:
		}
	}
	q.waiters = nil
}

func (q *TurnBubbleQueue) waveQueueEmpty(waveIndex int) bool {
	return len(q.pending[waveIndex]) == 0
}

func (q *TurnBubbleQueue) canAdvanceCursor() bool {
	if q.displayCursor >= q.waveCount {
		return false
	}
	if !q.generationComplete[q.displayCursor] {
		return false
	}
	return q.waveQueueEmpty(q.displayCursor)
}

func (q *TurnBubbleQueue) scheduleDrain() {
	q.mu.Lock()
	if q.draining || q.aborted {
		q.mu.Unlock()
		return
	}
	q.draining = true
	q.mu.Unlock()

	defer func() {
		q.mu.Lock()
		q.draining = false
		hasWork := (len(q.pending[q.displayCursor]) > 0) || q.canAdvanceCursor()
		q.mu.Unlock()
		if hasWork {
			q.scheduleDrain()
		}
	}()

	for {
		q.mu.Lock()
		if q.aborted || q.displayCursor >= q.waveCount {
			q.mu.Unlock()
			return
		}

		wave := q.displayCursor
		tasks := q.pending[wave]

		for len(tasks) > 0 && !q.aborted {
			task := tasks[0]
			q.pending[wave] = tasks[1:]
			tasks = q.pending[wave]
			q.mu.Unlock()

			// 执行任务
			_ = task()

			q.mu.Lock()
			if q.aborted {
				q.mu.Unlock()
				return
			}
		}

		if q.canAdvanceCursor() {
			q.displayDone[wave] = true
			q.displayCursor++
			q.notifyWaiters()
			q.mu.Unlock()
			// 继续尝试推进下一个 wave
			continue
		}

		q.mu.Unlock()
		return
	}
}
