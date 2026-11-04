// pipeline.go — 语音管线（VAD→EOT→STT→LLM→TTS）的会话/协议胶水层。
//
// 输入项生命周期/持发规则/响应装配语义逐字对齐 sidecar api/openai_realtime
// （规格 §5.1/5.2）。LLM 腿与 TTS 腿在刀5 接入（当前 final 之后只发布转写，
// 不产生响应——unanswered 路径直接 commit+close）。
package realtime

import (
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EOTEngine Smart Turn 适配（native.Engine.EndOfTurnTiming 直配）。
type EOTEngine interface {
	Predict(pcm []float32) (probability float32, took time.Duration, err error)
}

// STTEngine 转写引擎（native.Engine.Transcribe 直配）。
type STTEngine interface {
	Transcribe(pcm []float32, sampleRate int) (string, error)
}

// PipelineConfig 管线配置。
type PipelineConfig struct {
	VAD VADHandlerConfig
	// Legs LLM/TTS 腿（nil=无响应能力，转写后持发等待——刀3/4 态）。
	Legs ResponseLegs
	// ChatSize LLM 对话历史窗口（sidecar chat_size=12）。
	ChatSize int
	// CreateResponse 转写后是否产生响应（Legs!=nil 时恒 true）。
	CreateResponse bool
}

// Pipeline 单会话语音管线（实现 Handler；单会话池由 Server 保证）。
type Pipeline struct {
	cfg   PipelineConfig
	turns *SpeculativeTurns
	vad   *VADHandler
	eot   EOTEngine
	stt   STTEngine

	// —— 输入项状态（§5.1）——
	muItems             sync.Mutex
	inputItemByTurnRev  map[turnKey]string
	pendingTerminals    map[string]*pendingInputTerminal // itemID → terminal
	lastItemID          string
	completedFinalRevs  map[turnKey]bool // completed-final 集合（FIFO 上限 2048）
	completedFinalOrder []turnKey
	instructions        string // session.update 指令（app 层 LLM 适配读取）
	voiceID             string // session.update 输出音色（breeze 克隆 id）
	// —— 响应状态（刀5）——
	legs           ResponseLegs
	chatSize       int
	chat           []ChatMessage
	respState      *responseState
	inResponse     bool
	responseID     string
	responseTurnID string
	responseRev    int
	responseMux    sync.Mutex
	// 延迟账 + 墓碑
	muLatency       sync.Mutex
	lat             latencyRec
	lastSTT         float64
	lastLLM         float64
	lastTTFA        float64
	lastE2E         float64
	lastVADDecision float64
	lastSmart       string
	lastHold        float64
	closedKeys      map[string]bool
	closedOrder     []string
	// 音频时钟
	totalSamples   int
	pendingSamples []float32 // 512 块化余数（跨帧累积）

	// 持发重试泵（sidecar send-loop HOLD retry 语义：被 candidate/grace 挂住的
	// terminal 到期后要有东西再 resolve——这里用 50ms 泵替代其发送循环重试）
	pumpOnce sync.Once
	stopPump chan struct{}
	pumpSess *Session
	pumpMu   sync.Mutex
}

func (p *Pipeline) startPump(s *Session) {
	p.pumpOnce.Do(func() {
		p.pumpMu.Lock()
		p.pumpSess = s
		p.stopPump = make(chan struct{})
		stop := p.stopPump
		p.pumpMu.Unlock()
		go func() {
			t := time.NewTicker(50 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-t.C:
					p.muItems.Lock()
					empty := len(p.pendingTerminals) == 0
					p.muItems.Unlock()
					if empty {
						continue
					}
					for _, ev := range p.resolveInputTerminals(p.pumpSess, "") {
						p.pumpSess.Send(ev)
					}
				case <-stop:
					return
				}
			}
		}()
	})
}

type pendingInputTerminal struct {
	itemID        string
	turnID        string
	rev           int
	speechStopped map[string]any
	transcription map[string]any
	inputClosed   bool
}

const maxCompletedFinals = 2048

// NewPipeline 构造（eng 可为 nil→VAD/EOT/STT 缺席，仅协议壳）。
func NewPipeline(cfg PipelineConfig, scorer VadScorer, eot EOTEngine, stt STTEngine) *Pipeline {
	turns := NewSpeculativeTurns()
	p := &Pipeline{
		cfg:                cfg,
		turns:              turns,
		eot:                eot,
		stt:                stt,
		inputItemByTurnRev: map[turnKey]string{},
		pendingTerminals:   map[string]*pendingInputTerminal{},
		completedFinalRevs: map[turnKey]bool{},
	}
	var eotAn EOTAnalyzer
	if eot != nil {
		eotAn = eotEngineAdapter{eot}
	}
	p.vad = NewVADHandler(cfg.VAD, scorer, eotAn, turns)
	return p
}

type eotEngineAdapter struct{ e EOTEngine }

func (a eotEngineAdapter) Predict(pcm []float32) (float32, time.Duration, error) {
	return a.e.Predict(pcm)
}

// —— Handler 接口 ——

// OnAudio 上行音频：PCM16 字节 → float32 → 512 块喂 VAD。
// 不足 512 的余数跨帧累积（前端 320 样本/帧粒度；sidecar service 同语义）。
func (p *Pipeline) OnAudio(s *Session, pcm []byte) {
	n := len(pcm) / 2
	if n == 0 {
		return
	}
	p.startPump(s)
	p.muItems.Lock()
	p.totalSamples += n
	p.muItems.Unlock()

	samples := make([]float32, 0, len(p.pendingSamples)+n)
	samples = append(samples, p.pendingSamples...)
	for i := 0; i < n; i++ {
		samples = append(samples, float32(int16(uint16(pcm[2*i])|uint16(pcm[2*i+1])<<8))/32768.0)
	}
	const chunk = 512
	off := 0
	for off+chunk <= len(samples) {
		for _, ev := range p.vad.Process(samples[off : off+chunk]) {
			p.dispatchVADEvent(s, ev)
		}
		off += chunk
	}
	p.muItems.Lock()
	p.pendingSamples = append([]float32(nil), samples[off:]...)
	p.muItems.Unlock()
}

// OnSessionConfig session.update：阈值运行时生效。
func (p *Pipeline) OnSessionConfig(s *Session, cfg SessionConfig) {
	if cfg.VAD.Threshold != nil {
		p.vad.SetThreshold(*cfg.VAD.Threshold)
	}
	p.muItems.Lock()
	p.voiceID = cfg.Voice
	p.muItems.Unlock()
}

// Voice 当前输出音色。
func (p *Pipeline) Voice() string {
	p.muItems.Lock()
	defer p.muItems.Unlock()
	return p.voiceID
}

// OnSessionEnd 会话结束复位。
func (p *Pipeline) OnSessionEnd(s *Session) {
	p.muItems.Lock()
	p.inputItemByTurnRev = map[turnKey]string{}
	p.pendingTerminals = map[string]*pendingInputTerminal{}
	p.lastItemID = ""
	p.completedFinalRevs = map[turnKey]bool{}
	p.completedFinalOrder = nil
	p.totalSamples = 0
	p.pendingSamples = nil
	p.muItems.Unlock()
	p.pumpMu.Lock()
	if p.stopPump != nil {
		close(p.stopPump)
		p.stopPump = nil
	}
	p.pumpOnce = sync.Once{}
	p.pumpSess = nil
	p.pumpMu.Unlock()
	p.turns.Reset()
	p.vad.OnSessionEnd()
}

// OnResponseCancel 客户端取消（刀5 实装完整取消链）。
func (p *Pipeline) OnResponseCancel(s *Session) {
	p.responseMux.Lock()
	had := p.inResponse
	p.responseMux.Unlock()
	if had {
		s.Send(evResponseDone(ResponseDonePayload{
			ID:            p.responseID,
			Status:        "cancelled",
			StatusDetails: map[string]any{"reason": "client_cancelled"},
		}))
	}
	p.responseMux.Lock()
	p.inResponse = false
	p.responseMux.Unlock()
}

// OnResponseCreate 客户端显式请求（刀5 LLM 腿接入；当前回声占位由 Server 层兜）。
func (p *Pipeline) OnResponseCreate(s *Session) {
	// 刀5：LLM 生成链。当前无 LLM 腿——静默忽略（前端可再发起）。
}

// —— VAD 事件 → 协议发射（§5.1） ——

func (p *Pipeline) dispatchVADEvent(s *Session, ev VADEvent) {
	switch ev.Kind {
	case VADEventStarted:
		p.onSpeechStarted(s, ev.Started)
	case VADEventStopped:
		// 被持：pending terminal（不发）
		p.holdSpeechStopped(ev.Stopped)
	case VADEventFinal:
		p.onFinal(s, ev.Final)
	}
}

func (p *Pipeline) onSpeechStarted(s *Session, st *VADStarted) {
	events := p.resolveInputTerminals(s, st.TurnID)

	// barge-in：响应中 + 允许打断 → cancelled/turn_detected
	p.responseMux.Lock()
	interrupt := p.inResponse
	if st.InterruptResponse {
		_ = interrupt // 刀5 在此 finish_response(cancelled, turn_detected) + 队列 flush
	}
	p.responseMux.Unlock()

	isReopen := st.Reopened && st.TurnID != ""
	itemID, existed := p.inputItemForTurn(st.TurnID, st.Rev, isReopen)
	if existed && isReopen {
		// 复用同一协议 item，不重发 speech_started
		for _, e := range events {
			s.Send(e)
		}
		return
	}
	s.Send(evSpeechStarted(itemID, st.AudioStartMs))
	for _, e := range events {
		s.Send(e)
	}
}

func (p *Pipeline) holdSpeechStopped(st *VADStopped) {
	p.muItems.Lock()
	defer p.muItems.Unlock()
	key := turnKey{id: st.TurnID, rev: st.Rev}
	itemID, ok := p.inputItemByTurnRev[key]
	if !ok {
		slog.Debug("speech_stopped 无输入项（丢弃）", "turn", st.TurnID, "rev", st.Rev)
		return
	}
	term := p.pendingTerminals[itemID]
	if term == nil {
		term = &pendingInputTerminal{itemID: itemID, turnID: st.TurnID, rev: st.Rev}
		p.pendingTerminals[itemID] = term
	}
	term.speechStopped = evSpeechStopped(itemID, st.AudioEndMs)
	if st.DurationS > 0 {
		term.speechStopped["duration_s"] = st.DurationS
	}
}

// onFinal mode=final：STT 输入门 → 转写 → chat/响应（刀5）→ 持 terminal。
func (p *Pipeline) onFinal(s *Session, f *VADFinal) {
	// STT 输入门（§4）：final 稳定等待 = max(0, processing_delay − age)；
	// current 轮有 tracker processing_deadline 时用剩余时间替代。
	holdUntil := time.Time{}
	deadline, ok := p.turns.ProcessingDeadline(f.TurnID, f.Rev)
	if ok && deadline.After(time.Now()) {
		holdUntil = deadline
	}
	if p.turns.WaitForGate(f.TurnID, f.Rev, false, holdUntil) {
		p.transcribe(s, f)
	}
}

// transcribe STT + 转写完成事件 + 持 terminal。
func (p *Pipeline) transcribe(s *Session, f *VADFinal) {
	key := turnKey{id: f.TurnID, rev: f.Rev}
	if p.completedFinalRevs[key] {
		return // input-after-final
	}
	text := ""
	sttStart := time.Now()
	if p.stt != nil && len(f.Audio) > 0 {
		if t, err := p.stt.Transcribe(f.Audio, p.cfg.VAD.SampleRate); err == nil {
			text = t
		} else {
			slog.Warn("STT 失败", "turn", f.TurnID, "rev", f.Rev, "error", err)
		}
	}
	p.muLatency.Lock()
	p.lastSTT = time.Since(sttStart).Seconds()
	p.lastVADDecision = f.VADDecisionS
	p.lastSmart = f.SmartStatus
	p.muLatency.Unlock()
	itemID, ok := p.inputItemForTurn(f.TurnID, f.Rev, false)
	if !ok {
		return
	}
	p.muItems.Lock()
	p.completedFinalRevs[key] = true
	p.completedFinalOrder = append(p.completedFinalOrder, key)
	for len(p.completedFinalOrder) > maxCompletedFinals {
		delete(p.completedFinalRevs, p.completedFinalOrder[0])
		p.completedFinalOrder = p.completedFinalOrder[1:]
	}
	term := p.pendingTerminals[itemID]
	if term == nil {
		term = &pendingInputTerminal{itemID: itemID, turnID: f.TurnID, rev: f.Rev}
		p.pendingTerminals[itemID] = term
	}
	term.transcription = evTranscriptionCompleted(itemID, text)
	// chat 写入（刀5 LLM 腿时同批）。terminal 保持持发（sidecar 语义）：
	// 提前 commit+close 会杀死 speculative reopen（后续语音必须能 rev+1 重开
	// 同轮，被超龄 revision 的 terminal 由「下一轮开始」或「响应提交」处置）。
	p.muItems.Unlock()

	slog.Info("Transcription completed (language=zh): chars=" + strconv.Itoa(len([]rune(text))) +
		" turn=" + f.TurnID + " rev=" + strconv.Itoa(f.Rev))

	// 响应腿（刀5）：转写非空 → chat 历史 → LLM 流 → 分句 TTS → response 链。
	// 空转写：移除 chat user 消息等价=不入历史，直接返回（terminal 继续持发）。
	if text != "" && p.legs != nil && p.cfg.CreateResponse {
		p.chatAppend("user", text)
		p.generateResponse(s, text, f)
	}
	p.resolveInputTerminals(s, "")
}

// generateResponse 响应全链（LLM 流 + 分句 TTS + finish）。
func (p *Pipeline) generateResponse(s *Session, userText string, f *VADFinal) {
	history := p.chatSnapshot()
	// 历史含刚追加的 user 消息——LLM 侧剔除末条（adapter 自行组装 persona 注入）
	if len(history) > 0 {
		history = history[:len(history)-1]
	}
	rs := p.beginResponse(s, userText, f.TurnID, f.Rev, f.SpeechEndAt)
	// 提交轮（commit=true）：首个输出前提交（解除 hold、进入 ANSWERING）
	if g := p.turns.Gate(f.TurnID, f.Rev, false, time.Time{}, true); g.Action != GateAccept {
		// DROP=轮已过期；静默放弃
		return
	}
	p.sendResponseCreated(s, rs)

	llmStart := time.Now()
	sentenceCh := make(chan string, 64)
	done := make(chan error, 1)
	go func() {
		full, inT, outT, err := p.legs.Reply(rs.ctx, userText, history, f.TurnID, f.Rev, p.currentItemID(f.TurnID, f.Rev), func(delta string) {
			// 增量：开输出 + transcript.delta（规格：文本先于音频可见）
			p.beginAssistantOutput(s, rs)
			p.streamAssistantText(s, rs, delta)
			// 分句批切（stream_batch_sentences=1）
			p.pendSentence(rs, delta, sentenceCh)
		})
		rs.inTokens, rs.outTokens = inT, outT
		p.muLatency.Lock()
		p.lastLLM = time.Since(llmStart).Seconds()
		p.muLatency.Unlock()
		p.flushSentence(rs, sentenceCh)
		close(sentenceCh)
		done <- err
		_ = full
	}()

	speakErr := error(nil)
	first := true
	for sentence := range sentenceCh {
		if !p.speakSentence(s, rs, sentence, first) {
			speakErr = errInterrupted
			first = false
			break
		}
		first = false
	}
	if speakErr == errInterrupted {
		// barge-in 已在 onSpeechStarted 里 finishResponse(cancelled/turn_detected)
		<-done
		return
	}
	if err := <-done; err != nil {
		if rs.ctx.Err() != nil {
			return // 已取消路径
		}
		slog.Warn("LLM 生成失败", "error", err)
		// 兜底语（sidecar PROVIDER_FAILURE_FALLBACK 语义）
		p.beginAssistantOutput(s, rs)
		p.streamAssistantText(s, rs, "我现在有点跟不上，请再说一遍。")
		if !p.speakSentence(s, rs, "我现在有点跟不上，请再说一遍。", false) {
			return
		}
		p.finishResponse(s, rs, "failed", "response_failed")
		return
	}
	p.finishResponse(s, rs, "completed", "")
}

// currentItemID 当前轮协议 item id（LLM 关联头用）。
func (p *Pipeline) currentItemID(turnID string, rev int) string {
	p.muItems.Lock()
	defer p.muItems.Unlock()
	if id, ok := p.inputItemByTurnRev[turnKey{id: turnID, rev: rev}]; ok {
		return id
	}
	for k, id := range p.inputItemByTurnRev {
		if k.id == turnID {
			return id
		}
	}
	return ""
}

var errInterrupted = errorString("interrupted")

type errorString string

func (e errorString) Error() string { return string(e) }

// pendSentence 流式分句累积（跨 delta）。
func (p *Pipeline) pendSentence(rs *responseState, delta string, ch chan<- string) {
	rs.mu.Lock()
	rs.sb += delta
	text := rs.sb
	rs.mu.Unlock()
	sentences := spokenSplit(text)
	if len(sentences) > 1 {
		// 除最后一段（可能未完）外全部下发
		complete := ""
		for i := 0; i < len(sentences)-1; i++ {
			complete += sentences[i]
		}
		rs.mu.Lock()
		rs.sb = sentences[len(sentences)-1]
		rs.parts = append(rs.parts, sentences[:len(sentences)-1]...)
		rs.mu.Unlock()
		ch <- complete
	}
}

// flushSentence 响应结束前冲掉残余半句。
func (p *Pipeline) flushSentence(rs *responseState, ch chan<- string) {
	rs.mu.Lock()
	rest := strings.TrimSpace(rs.sb)
	rs.sb = ""
	if rest != "" {
		rs.parts = append(rs.parts, rest)
	}
	rs.mu.Unlock()
	if rest != "" {
		ch <- rest
	}
}

// inputItemForTurn 取/建输入协议 item id。isReopen 时复用同 turn 既有 revision 的
// item（sidecar _reuse_input_item：重开不重发 speech_started）。
func (p *Pipeline) inputItemForTurn(turnID string, rev int, isReopen bool) (string, bool) {
	p.muItems.Lock()
	defer p.muItems.Unlock()
	if id, ok := p.inputItemByTurnRev[turnKey{id: turnID, rev: rev}]; ok {
		return id, true
	}
	if isReopen {
		for k, id := range p.inputItemByTurnRev {
			if k.id == turnID {
				p.inputItemByTurnRev[turnKey{id: turnID, rev: rev}] = id
				return id, true
			}
		}
	}
	id := "item_" + randHex(8)
	p.inputItemByTurnRev[turnKey{id: turnID, rev: rev}] = id
	return id, false
}

// resolveInputTerminals 持发处置（§5.1 规则表）。
func (p *Pipeline) resolveInputTerminals(s *Session, startedTurnID string) []map[string]any {
	p.muItems.Lock()
	defer p.muItems.Unlock()
	var out []map[string]any
	for itemID, term := range p.pendingTerminals {
		var disposition string // publish | hold | discard
		if term.turnID == "" || !p.turns.IsLatest(term.turnID, term.rev) {
			if p.turns.IsCurrentTurn(term.turnID) {
				disposition = "discard" // 同 live turn 的超龄 revision
			} else {
				term.inputClosed = true
				disposition = "publish"
			}
		} else if startedTurnID != "" && term.turnID != startedTurnID {
			disposition = "publish"
		} else {
			hasResponse := (p.inResponse && p.responseTurnID == term.turnID) ||
				p.turns.IsCommitted(term.turnID, term.rev)
			unanswered := term.inputClosed && !hasResponse
			failed := term.transcription != nil && term.transcription["type"] == "conversation.item.input_audio_transcription.failed"
			if unanswered || failed {
				g := p.turns.Gate(term.turnID, term.rev, false, time.Time{}, true)
				if g.Action == GateHold {
					disposition = "hold"
				} else if g.Action == GateAccept {
					if !hasResponse {
						p.turns.Close(term.turnID, term.rev)
					}
					disposition = "publish"
				} else {
					disposition = "discard"
				}
			} else if p.turns.IsCommitted(term.turnID, term.rev) {
				disposition = "publish"
			} else {
				disposition = "hold"
			}
		}
		switch disposition {
		case "publish":
			out = append(out, p.publishTerminalLocked(term)...)
		case "discard":
			delete(p.pendingTerminals, itemID)
		case "hold":
			// 继续持
		}
	}
	return out
}

// publishTerminalLocked 发射顺序：speech_stopped → committed → item.created → transcription。
func (p *Pipeline) publishTerminalLocked(term *pendingInputTerminal) []map[string]any {
	var out []map[string]any
	if term.speechStopped != nil {
		out = append(out, term.speechStopped)
		term.speechStopped = nil
	}
	out = append(out, map[string]any{
		"type":             "input_audio_buffer.committed",
		"item_id":          term.itemID,
		"previous_item_id": p.lastItemID,
	})
	out = append(out, evItemCreated(map[string]any{
		"id":               term.itemID,
		"object":           "realtime.item",
		"type":             "message",
		"role":             "user",
		"status":           "completed",
		"content":          []map[string]any{{"type": "input_audio"}},
		"previous_item_id": p.lastItemID,
	}))
	p.lastItemID = term.itemID
	if term.transcription != nil {
		out = append(out, term.transcription)
		term.transcription = nil
	}
	if term.inputClosed {
		delete(p.pendingTerminals, term.itemID)
	}
	return out
}
