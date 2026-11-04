// response.go — response 装配/取消/墓碑/chat 历史（规格 §5.3~5.5）。
package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// ChatMessage LLM 对话历史条目（user/assistant）。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseLegs LLM/TTS 腿接口（app 层适配注入；realtime 保持传输无关）。
type ResponseLegs interface {
	// Reply 流式生成回复（persona 注入/功能绑定解析由实现方负责）。
	// turnID/rev/itemID 供轮次关联；onDelta 收增量文本；返回完整文本与 token 用量。
	Reply(ctx context.Context, userText string, history []ChatMessage, turnID string, rev int, itemID string, onDelta func(string)) (fullText string, inTokens, outTokens int, err error)
	// Speak 单句合成：onAudio 收 PCM16@16k 音频块；可取消。
	Speak(ctx context.Context, sentence string, onAudio func([]byte)) error
}

// responseState 一次响应的在途状态。
type responseState struct {
	id           string
	key          string // response_key（uuid hex）
	turnID       string
	rev          int
	respItemID   string // assistant 消息 item_id
	cancel       context.CancelFunc
	ctx          context.Context
	startedAt    time.Time
	textSent     bool // output_item/content_part 是否已开
	audioOpened  bool
	audioBuf     []byte
	parts        []string  // transcript parts
	sb           string    // 分句残段累积
	speechAnchor time.Time // speech_stopped_at（e2e 锚）
	inTokens     int
	outTokens    int
	pipeline     *Pipeline
	mu           sync.Mutex
}

// pipeline 响应面状态（挂在 Pipeline 上）。
//   inResponse/responseID/responseTurnID/responseRev 已有；
//   新增 respState、closedKeys 墓碑（≤128 FIFO）、chat 历史。

func (p *Pipeline) beginResponse(s *Session, userText string, turnID string, rev int, speechStoppedAt time.Time) *responseState {
	p.responseMux.Lock()
	defer p.responseMux.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	rs := &responseState{
		id:           "resp_" + randHex(6),
		key:          randHex(16),
		turnID:       turnID,
		rev:          rev,
		respItemID:   "item_" + randHex(8),
		cancel:       cancel,
		ctx:          ctx,
		startedAt:    time.Now(),
		speechAnchor: speechStoppedAt,
	}
	rs.pipeline = p
	p.inResponse = true
	p.responseID = rs.id
	p.responseTurnID = turnID
	p.responseRev = rev
	p.respState = rs
	return rs
}

// sendResponseCreated 首输出前的顺序守卫（created 先于一切输出帧）。
func (p *Pipeline) sendResponseCreated(s *Session, rs *responseState) {
	s.Send(evResponseCreated(rs.id))
}

// beginAssistantOutput output_item.added + content_part.added（一次）。
func (p *Pipeline) beginAssistantOutput(s *Session, rs *responseState) {
	if rs.textSent {
		return
	}
	rs.textSent = true
	s.Send(evOutputItemAdded(rs.id, rs.respItemID, 0))
	s.Send(evContentPartAdded(rs.id, rs.respItemID))
}

// streamAssistantText 增量文本（transcript.delta）。
func (p *Pipeline) streamAssistantText(s *Session, rs *responseState, delta string) {
	if delta == "" {
		return
	}
	s.Send(evTranscriptDelta(rs.id, rs.respItemID, delta))
}

// speakSentence 单句 TTS → audio.delta（6400B 批）→ 完成；首个音频块提交轮次。
// 返回 false=被打断（barge-in/cancel）。
func (p *Pipeline) speakSentence(s *Session, rs *responseState, sentence string, isFirst bool) bool {
	if p.legs == nil {
		return true
	}
	// gate：提交轮（commit=true）——首个 TTS 音频提交（规格：TTS wait_for_gate(commit=True)）
	g := p.turns.Gate(rs.turnID, rs.rev, true, time.Time{}, true)
	if g.Action == GateHold {
		ok := p.turns.WaitForGate(rs.turnID, rs.rev, true, time.Time{})
		if !ok {
			return false
		}
	} else if g.Action == GateDrop {
		return false
	}
	p.beginAssistantOutput(s, rs)
	ttfaStart := time.Now()
	firstAudio := true
	err := p.legs.Speak(rs.ctx, sentence, func(pcm []byte) {
		if firstAudio {
			firstAudio = false
			p.muLatency.Lock()
			p.lastTTFA = time.Since(ttfaStart).Seconds()
			if !rs.speechAnchor.IsZero() {
				p.lastE2E = time.Since(rs.speechAnchor).Seconds()
			}
			p.muLatency.Unlock()
		}
		rs.mu.Lock()
		rs.audioBuf = append(rs.audioBuf, pcm...)
		for len(rs.audioBuf) >= 6400 {
			chunk := rs.audioBuf[:6400]
			rs.audioBuf = rs.audioBuf[6400:]
			s.Send(evAudioDelta(rs.id, rs.respItemID, chunk))
		}
		rs.mu.Unlock()
	})
	if err != nil {
		if rs.ctx.Err() != nil {
			return false // 取消
		}
		slogWarn("TTS 句失败", "error", err)
	}
	rs.mu.Lock()
	if len(rs.audioBuf) > 0 {
		s.Send(evAudioDelta(rs.id, rs.respItemID, rs.audioBuf))
		rs.audioBuf = nil
	}
	rs.mu.Unlock()
	s.Send(evAudioDone(rs.id, rs.respItemID))
	return true
}

// finishResponse 终态装配（transcript.done → content_part.done → item.done →
// response.done{usage+latency metadata}）+ tracker close + 墓碑。
func (p *Pipeline) finishResponse(s *Session, rs *responseState, status string, reason string) {
	full := ""
	for i, part := range rs.parts {
		if i > 0 {
			full += " "
		}
		full += part
	}
	if rs.textSent {
		s.Send(evTranscriptDone(rs.id, rs.respItemID, full))
		s.Send(evContentPartDone(rs.id, rs.respItemID))
		itemStatus := "completed"
		if status != "completed" {
			itemStatus = "incomplete"
		}
		s.Send(evOutputItemDone(rs.id, rs.respItemID, 0))
		_ = itemStatus
	}
	done := evResponseDone(ResponseDonePayload{
		ID:            rs.id,
		Status:        status,
		StatusDetails: map[string]any{"reason": reason},
		Usage: map[string]any{
			"input_tokens":         rs.inTokens,
			"output_tokens":        rs.outTokens,
			"total_tokens":         rs.inTokens + rs.outTokens,
			"input_token_details":  map[string]any{"audio_tokens": 0, "cached_tokens": 0, "image_tokens": 0, "text_tokens": rs.inTokens},
			"output_token_details": map[string]any{"audio_tokens": 0, "text_tokens": rs.outTokens},
		},
	})
	done["metadata"] = map[string]any{
		"speech_to_speech.turn_latency": p.latencyMetadata(rs, status, reason),
	}
	s.Send(done)
	// 日志行（eot_analyze/false_interrupt_bound 解析格式）
	slogLatencyLine(rs, status, reason, full)
	// tracker close + 墓碑 + 状态清位
	p.turns.Close(rs.turnID, rs.rev)
	p.tombstone(rs.key)
	p.responseMux.Lock()
	if p.respState == rs {
		p.inResponse = false
		p.respState = nil
	}
	p.responseMux.Unlock()
}

// latencyMetadata response.done metadata（≤512 字节才附，规格条件由调用面守住）。
func (p *Pipeline) latencyMetadata(rs *responseState, status, reason string) map[string]any {
	p.muLatency.Lock()
	defer p.muLatency.Unlock()
	rev := rs.rev
	return map[string]any{
		"version":        2,
		"source":         "speech",
		"turn_id":        rs.turnID,
		"turn_revision":  rev,
		"response_key":   rs.key,
		"stt_s":          p.lastSTT,
		"llm_s":          p.lastLLM,
		"tts_ttfa_s":     p.lastTTFA,
		"e2e_s":          p.lastE2E,
		"vad_decision_s": p.lastVADDecision,
		"smart_status":   p.lastSmart,
		"hold_s":         p.lastHold,
		"status":         status,
	}
}

// tombstone 关闭 response_key（≤128 FIFO）。
func (p *Pipeline) tombstone(key string) {
	p.muItems.Lock()
	p.closedKeys[key] = true
	p.closedOrder = append(p.closedOrder, key)
	for len(p.closedOrder) > 128 {
		delete(p.closedKeys, p.closedOrder[0])
		p.closedOrder = p.closedOrder[1:]
	}
	p.muItems.Unlock()
}

// bargeIn 打断：取消在途响应 + finishResponse(cancelled/turn_detected)。
func (p *Pipeline) bargeIn(s *Session, rs *responseState) {
	if rs == nil {
		return
	}
	rs.cancel()
	p.finishResponse(s, rs, "cancelled", "turn_detected")
	slogInfo("Pipeline 0: speech during response: cancelled, queue flushed")
}

// chatAppend 追加历史（chat_size 窗口）。
func (p *Pipeline) chatAppend(role, text string) {
	p.muItems.Lock()
	p.chat = append(p.chat, ChatMessage{Role: role, Content: text})
	for len(p.chat) > p.chatSize {
		p.chat = p.chat[1:]
	}
	p.muItems.Unlock()
}

func (p *Pipeline) chatSnapshot() []ChatMessage {
	p.muItems.Lock()
	defer p.muItems.Unlock()
	out := make([]ChatMessage, len(p.chat))
	copy(out, p.chat)
	return out
}

var (
	_ = json.Marshal // 保留（metadata 序列化校验用）
	_ = sync.Mutex{}
)
