// Package whisper — paced_stream.go
// 100% 对齐 ackem chat/pacedStreamEmitter.ts
// 流式节奏控制：累积文本 → 检测完整句子 → 按间隔回调展示

package whisper

import (
	"strings"
	"sync"
	"time"
)

// SPLIT_MARKER LLM 节奏分隔符
const SplitMarker = "[SPLIT]"

// ReplyInterSentenceGapMs 默认句子间隔（对齐 ackem REPLY_INTER_SENTENCE_GAP_MS）
const ReplyInterSentenceGapMs = 900

// StripSplitMarkers 展示/持久化前去掉 LLM 节奏分隔符
// 100% 对齐 ackem pacedStreamEmitter.ts stripSplitMarkers
func StripSplitMarkers(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, SplitMarker, ""))
}

// FirstDisplayUnitLen 下一段可完整展示的内容长度（整句或 [SPLIT] 标记）
// 100% 对齐 ackem pacedStreamEmitter.ts firstDisplayUnitLen
func FirstDisplayUnitLen(unsent string) int {
	if unsent == "" {
		return -1
	}
	if strings.HasPrefix(unsent, SplitMarker) {
		return len(SplitMarker)
	}
	splitAt := strings.Index(unsent, SplitMarker)
	if splitAt > 0 {
		return splitAt
	}
	end := FindSafeSentenceBreak(unsent)
	if end > 0 {
		// FindSafeSentenceBreak 返回 rune 索引；调用方按字节切片，需转字节偏移
		runes := []rune(unsent)
		if end <= len(runes) {
			return len(string(runes[:end]))
		}
		return end
	}
	return -1
}

// PacedStreamCallbacks 节奏流回调
type PacedStreamCallbacks struct {
	// OnStart 流开始时回调（仅一次）
	OnStart func()
	// OnChunk 每次有新文本展示时回调
	OnChunk func(chunk string)
	// OnBubbleStart 新气泡开始时回调
	OnBubbleStart func(waveIndex int, newBubble bool)
	// OnBubbleEnd 气泡结束时回调（text 为去分隔符后的完整文本）
	OnBubbleEnd func(waveIndex int, text string)
}

// PacedStreamEmitter 节奏流发射器
// 100% 对齐 ackem pacedStreamEmitter.ts PacedStreamEmitter
type PacedStreamEmitter struct {
	cb    PacedStreamCallbacks
	gapMs int

	mu            sync.Mutex
	received      strings.Builder
	sentLen       int
	streamStarted bool
	streamDone    bool
	pumping       bool
	pendingPump   bool

	pauseBeforeNext bool
	openNextBubble  bool

	sentenceIndex   int
	bubbleOpen      bool
	bubbleSentStart int

	done chan struct{}
	stop chan struct{}
}

// NewPacedStreamEmitter 创建节奏流发射器
func NewPacedStreamEmitter(cb PacedStreamCallbacks, gapMs int) *PacedStreamEmitter {
	if gapMs <= 0 {
		gapMs = ReplyInterSentenceGapMs
	}
	return &PacedStreamEmitter{
		cb:    cb,
		gapMs: gapMs,
		done:  make(chan struct{}),
		stop:  make(chan struct{}),
	}
}

// OnDelta 接收新的 delta 文本
func (p *PacedStreamEmitter) OnDelta(delta string) {
	p.mu.Lock()
	p.received.WriteString(delta)
	p.mu.Unlock()
	go p.pump()
}

// MarkDone 标记流结束
func (p *PacedStreamEmitter) MarkDone() {
	p.mu.Lock()
	p.streamDone = true
	p.mu.Unlock()
	go p.pump()

	// 等待全部排空
	for {
		p.mu.Lock()
		done := p.sentLen >= p.received.Len() && !p.bubbleOpen && !p.pumping
		p.mu.Unlock()
		if done {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Close 关闭发射器
func (p *PacedStreamEmitter) Close() {
	close(p.stop)
}

// ─── 内部逻辑 ──────────────────────────────────────────────────

func (p *PacedStreamEmitter) emitStart() {
	if p.streamStarted {
		return
	}
	p.streamStarted = true
	if p.cb.OnStart != nil {
		p.cb.OnStart()
	}
}

func (p *PacedStreamEmitter) beginBubble(newBubble bool) {
	p.skipSplitMarkers()
	if p.cb.OnBubbleStart != nil {
		p.cb.OnBubbleStart(p.sentenceIndex, newBubble)
	}
	p.bubbleOpen = true
	p.bubbleSentStart = p.sentLen
}

func (p *PacedStreamEmitter) finishBubble() {
	if !p.bubbleOpen {
		return
	}
	text := StripSplitMarkers(p.received.String()[p.bubbleSentStart:p.sentLen])
	if p.cb.OnBubbleEnd != nil {
		p.cb.OnBubbleEnd(p.sentenceIndex, text)
	}
	p.bubbleOpen = false
	p.sentenceIndex++
}

func (p *PacedStreamEmitter) ensureBubbleOpen() {
	if p.bubbleOpen {
		return
	}
	p.beginBubble(p.sentenceIndex > 0)
}

func (p *PacedStreamEmitter) emitChunk(chunk string) {
	if chunk == "" {
		return
	}
	p.emitStart()
	p.ensureBubbleOpen()
	if p.cb.OnChunk != nil {
		p.cb.OnChunk(chunk)
	}
}

func (p *PacedStreamEmitter) skipSplitMarkers() {
	received := p.received.String()
	for strings.HasPrefix(received[p.sentLen:], SplitMarker) {
		p.sentLen += len(SplitMarker)
	}
}

func (p *PacedStreamEmitter) schedulePumpAfterGap() {
	p.openNextBubble = true
	time.AfterFunc(time.Duration(p.gapMs)*time.Millisecond, func() {
		p.pump()
	})
}

func (p *PacedStreamEmitter) pump() {
	p.mu.Lock()
	if p.pumping {
		p.pendingPump = true
		p.mu.Unlock()
		return
	}
	p.pumping = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.pumping = false
		hasPending := p.pendingPump
		if hasPending {
			p.pendingPump = false
		}
		p.mu.Unlock()
		if hasPending {
			p.pump()
		}
	}()

	for {
		p.mu.Lock()

		if p.pauseBeforeNext {
			p.pauseBeforeNext = false
			if p.bubbleOpen {
				p.finishBubble()
			}
			p.mu.Unlock()
			select {
			case <-time.After(time.Duration(p.gapMs) * time.Millisecond):
			case <-p.stop:
				return
			}
			p.mu.Lock()
		}

		if p.openNextBubble || (!p.bubbleOpen && p.sentLen < p.received.Len()) {
			p.openNextBubble = false
			p.beginBubble(p.sentenceIndex > 0)
		}

		p.skipSplitMarkers()

		received := p.received.String()
		if p.sentLen >= len(received) {
			if p.streamDone {
				p.mu.Unlock()
				return
			}
			p.mu.Unlock()
			return
		}

		unsent := received[p.sentLen:]
		unitLen := FirstDisplayUnitLen(unsent)

		if unitLen > 0 {
			p.emitChunk(unsent[:unitLen])
			p.sentLen += unitLen
			if p.sentLen < len(received) {
				p.finishBubble()
				p.mu.Unlock()
				p.schedulePumpAfterGap()
				return
			}
			p.pauseBeforeNext = true
			p.mu.Unlock()
			return
		}

		if !p.streamDone {
			p.emitChunk(unsent)
			p.sentLen += len(unsent)
			p.mu.Unlock()
			return
		}

		p.emitChunk(unsent)
		p.sentLen += len(unsent)
		if p.sentLen >= len(received) {
			p.finishBubble()
		}
		p.mu.Unlock()
		return
	}
}
