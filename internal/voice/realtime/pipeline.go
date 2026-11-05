// pipeline.go — 语音管线：5 阶段 goroutine + 信号驱动发送循环。
//
// 拓扑逐字对齐 sidecar s2s_pipeline.py build_configured_server 的队列图（8 队列 5 线程）
// 与 websocket_router._send_loop_for（output_signal 驱动、文本侧先于音频、
// 停靠/HOLD 重试、barge-in 三队列 flush、每轮 resolve_input_terminals）：
//
//	WS read ─► audioCh ─► [VAD 线程] ─► sttCh（utterance）
//	                              └─► textCh（started/stopped）
//	          [STT 线程] sttCh ─► textCh（transcription）─► promptCh（GenerateResponse）
//	          [LLM 线程] promptCh ─► textCh（assistant delta）+ ttsCh（句子）
//	          [TTS 线程] ttsCh ─► audioOutCh（PCM 块 + done 哨兵）
//	发送循环：signal 唤醒 → 文本侧（停靠/HOLD）→ 协议翻译全在此 → 音频批 6400B
//
// 协议翻译（resolve terminals/barge-in/response 装配）只在发送循环单线程发生
// ——sidecar 的 service.dispatch_pipeline_event 同位。规格：进度计划/gaea-voice-sidecar-port-spec-20261104.md §5。
package realtime

import (
	"context"
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

type eotEngineAdapter struct{ e EOTEngine }

func (a eotEngineAdapter) Predict(pcm []float32) (float32, time.Duration, error) {
	return a.e.Predict(pcm)
}

// STTEngine 转写引擎。
type STTEngine interface {
	Transcribe(pcm []float32, sampleRate int) (string, error)
}

// ResponseLegs LLM/TTS 腿（app 层适配注入）。
type ResponseLegs interface {
	Reply(ctx context.Context, userText string, history []ChatMessage, turnID string, rev int, itemID string, onDelta func(string)) (fullText string, inTokens, outTokens int, err error)
	Speak(ctx context.Context, sentence string, onAudio func([]byte)) error
}

// cancelScope 统一取消信号（pipeline/cancel_scope.py 同位）：
// 代际计数让管线线程检测取消；discarding 让发送循环丢弃陈旧输出。
type cancelScope struct {
	mu         sync.Mutex
	gen        int
	discarding bool
}

func (c *cancelScope) current() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gen
}

// cancel 递增代际并进入丢弃窗口（barge-in/客户端取消）。
func (c *cancelScope) discardable(gen int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if gen < c.gen {
		return true
	}
	return c.discarding && gen != c.gen
}

// clearDiscarding 终态处理完毕清除丢弃窗口（response_done 同位）。
func (c *cancelScope) clearDiscarding() {
	c.mu.Lock()
	c.discarding = false
	c.mu.Unlock()
}

// PipelineConfig 管线配置。
type PipelineConfig struct {
	VAD            VADHandlerConfig
	Legs           ResponseLegs
	ChatSize       int
	CreateResponse bool
}

// ─────────── 管线消息（pipeline/messages.py 同位） ───────────

// textEvent 文本侧通道事件（text_output_queue 载荷）。
type textEvent struct {
	kind       int
	started    *VADStarted
	stopped    *VADStopped
	transcript string
	resp       *responseState
}

const (
	evKindSpeechStarted = iota
	evKindSpeechStopped
	evKindTranscription
	evKindAssistantDelta
)

// promptRequest GenerateResponseRequest（text_prompt_queue 载荷）。
type promptRequest struct {
	resp            *responseState
	userText        string
	history         []ChatMessage
	speechStoppedAt time.Time
}

// ttsRequest 单句合成（lm_processed_queue 载荷）。
type ttsRequest struct {
	resp     *responseState
	sentence string
	last     bool // 响应最后一句（合成完发 done 哨兵）
}

// audioOut send_audio_chunks 载荷：音频块或 done 哨兵。
type audioOut struct {
	resp *responseState
	pcm  []byte
	done bool
	gen  int
}

// Pipeline 单会话语音管线（实现 Handler）。
type Pipeline struct {
	cfg   PipelineConfig
	turns *SpeculativeTurns
	vad   *VADHandler
	eot   EOTEngine
	stt   STTEngine
	legs  ResponseLegs

	// —— 队列（s2s_pipeline 拓扑同位）——
	audioCh  chan []float32      // recv_audio_chunks_queue
	sttCh    chan *VADFinal      // spoken_prompt_queue
	promptCh chan *promptRequest // text_prompt_queue
	ttsCh    chan *ttsRequest    // lm_processed_queue
	audioOut chan *audioOut      // send_audio_chunks_queue
	textCh   chan *textEvent     // text_output_queue
	cancelCh chan string         // 客户端取消请求（发送循环消费）
	signal   chan struct{}       // output_signal（NotifyingQueue 聚合）

	// —— 发送循环服务状态（单线程访问；sidecar SessionState 同位）——
	inputItemByTurnRev map[turnKey]string
	pendingTerminals   map[string]*pendingInputTerminal
	lastItemID         string
	completedFinalRevs map[turnKey]bool
	completedOrder     []turnKey
	inResponse         bool
	respState          *responseState
	responseTurnID     string
	closedKeys         map[string]bool
	closedOrder        []string
	chat               []ChatMessage
	chatSize           int
	voiceID            string
	pendingText        []*textEvent // 停靠文本事件（pending_text_output_items）
	parkedAudio        *audioOut    // pending_output_item（单停靠位）
	audioBatch         []byte

	scope cancelScope // 统一取消信号（代际）
	// 聊天历史互斥（STT/LLM 线程写，发送循环读）
	muHist sync.Mutex
	// muTerm 终态段锁：closedKeys/closedOrder/respState 翻转的写写互斥。
	// 生产路径 finishResponse 只在 sendLoop 单线程；测试（F3 回归钉）与
	// 潜在新调用方直调 cancelActiveResponse 时兜住并发写（2026-11-05
	// 全量套件实弹 panic：closedKeys 并发写）。
	muTerm sync.Mutex
	// 延迟账（多 worker 写，发送循环读）
	muLatency       sync.Mutex
	lastSTT         float64
	lastLLM         float64
	lastLLMTTFT     float64
	lastTTFA        float64
	lastE2E         float64
	lastVADDecision float64
	lastSmart       string
	lastHold        float64

	session        *Session
	totalSamples   int
	pendingSamples []float32
	stopped        chan struct{}
	startOnce      sync.Once
}

// pendingInputTerminal 持发的输入项终态。
type pendingInputTerminal struct {
	itemID        string
	turnID        string
	rev           int
	speechStopped map[string]any
	transcription map[string]any
	inputClosed   bool
}

const (
	maxCompletedFinals = 2048
	maxClosedKeys      = 128
)

// NewPipeline 构造。
func NewPipeline(cfg PipelineConfig, scorer VadScorer, eot EOTEngine, stt STTEngine) *Pipeline {
	if cfg.ChatSize <= 0 {
		cfg.ChatSize = 12
	}
	turns := NewSpeculativeTurns()
	p := &Pipeline{
		cfg:                cfg,
		turns:              turns,
		eot:                eot,
		stt:                stt,
		legs:               cfg.Legs,
		chatSize:           cfg.ChatSize,
		audioCh:            make(chan []float32, 8192),
		sttCh:              make(chan *VADFinal, 16),
		promptCh:           make(chan *promptRequest, 16),
		ttsCh:              make(chan *ttsRequest, 64),
		audioOut:           make(chan *audioOut, 4096),
		textCh:             make(chan *textEvent, 256),
		cancelCh:           make(chan string, 4),
		signal:             make(chan struct{}, 1),
		inputItemByTurnRev: map[turnKey]string{},
		pendingTerminals:   map[string]*pendingInputTerminal{},
		completedFinalRevs: map[turnKey]bool{},
		closedKeys:         map[string]bool{},
		stopped:            make(chan struct{}),
	}
	var eotAn EOTAnalyzer
	if eot != nil {
		eotAn = eotEngineAdapter{eot}
	}
	p.vad = NewVADHandler(cfg.VAD, scorer, eotAn, turns)
	return p
}

func (p *Pipeline) notify() {
	select {
	case p.signal <- struct{}{}:
	default:
	}
}

// ─────────── Handler 接口（WS 读线程；绝不阻塞） ───────────

func (p *Pipeline) OnAudio(s *Session, pcm []byte) {
	p.session = s
	n := len(pcm) / 2
	if n == 0 {
		return
	}
	p.totalSamples += n
	samples := make([]float32, 0, len(p.pendingSamples)+n)
	samples = append(samples, p.pendingSamples...)
	for i := 0; i < n; i++ {
		samples = append(samples, float32(int16(uint16(pcm[2*i])|uint16(pcm[2*i+1])<<8))/32768.0)
	}
	const chunk = 512
	off := 0
	for off+chunk <= len(samples) {
		buf := make([]float32, chunk)
		copy(buf, samples[off:off+chunk])
		off += chunk
		select {
		case p.audioCh <- buf:
		case <-p.stopped:
			return
		}
	}
	p.pendingSamples = append([]float32(nil), samples[off:]...)
	p.startOnce.Do(p.runStages)
	p.notify()
}

func (p *Pipeline) OnSessionConfig(_ *Session, cfg SessionConfig) {
	if cfg.VAD.Threshold != nil {
		p.vad.SetThreshold(*cfg.VAD.Threshold)
	}
	p.voiceID = cfg.Voice
}

func (p *Pipeline) Voice() string { return p.voiceID }

// OnSessionEnd 会话断开：清队列+会话态，常驻阶段线程不清（管线=应用生命周期，
// 第二次进语音必须照常工作——stopped 通道只在应用退出时关闭）。
func (p *Pipeline) OnSessionEnd(_ *Session) {
	if p.respState != nil {
		p.respState.cancel()
		p.respState = nil
	}
	p.inResponse = false
	p.drainQueues()
	p.pendingText = nil
	p.parkedAudio = nil
	p.audioBatch = nil
	p.inputItemByTurnRev = map[turnKey]string{}
	p.pendingTerminals = map[string]*pendingInputTerminal{}
	p.lastItemID = ""
	p.completedFinalRevs = map[turnKey]bool{}
	p.completedOrder = nil
	p.totalSamples = 0
	p.pendingSamples = nil
	p.session = nil
	// S3：每连接新建 Chat（service.py:413 register 同位）——历史不跨 WS 会话残留
	p.muHist.Lock()
	p.chat = nil
	p.muHist.Unlock()
	p.closedKeys = map[string]bool{}
	p.closedOrder = nil
	p.turns.Reset()
	p.vad.OnSessionEnd()
}

// drainQueues 非阻塞清空全部管线队列（会话断开时丢弃在途工作）。
func (p *Pipeline) drainQueues() {
	for {
		select {
		case <-p.audioCh:
		case <-p.sttCh:
		case <-p.promptCh:
		case <-p.ttsCh:
		case <-p.audioOut:
		case <-p.textCh:
		default:
			return
		}
	}
}

// OnResponseCancel 客户端取消：cancelled/client_cancelled（规格 §5.5）。
// 发送循环线程安全：经 textCh 序列化（避免与发送循环状态竞争）。
func (p *Pipeline) OnResponseCancel(s *Session) {
	p.cancelCh <- "client_cancelled"
}

func (p *Pipeline) OnResponseCreate(_ *Session) {}

// cancelActiveResponse barge-in/客户端取消共用（规格 §5.4/5.5）：
// cancel scope（resp.ctx）+ flush prompt/tts 队列 + finish_response(cancelled)。
func (p *Pipeline) cancelActiveResponse(reason string) {
	rs := p.respState
	inResp := p.inResponse
	if !inResp && rs == nil {
		return
	}
	// flush text_prompt_queue：逐个取出标记 dead（上游 close_pending_responses 同位：
	// 排队响应逐 key 收口）。单接收——旧双接收写法首载荷静默丢弃且单请求时会阻塞发送循环。
	for {
		select {
		case pr := <-p.promptCh:
			if pr != nil && pr.resp != nil {
				pr.resp.dead = true
			}
			continue
		default:
		}
		break
	}
	if rs != nil {
		rs.cancel()
		p.finishResponse(p.session, rs, "cancelled", reason)
	}
}

// ─────────── 阶段线程 ───────────

// runStages 启动全部阶段线程（首次音频时）。
func (p *Pipeline) runStages() {
	// VAD 线程：audioCh → VAD → started/stopped→textCh；final→sttCh
	go func() {
		for {
			select {
			case <-p.stopped:
				return
			case x := <-p.audioCh:
				for _, ev := range p.vad.Process(x) {
					switch ev.Kind {
					case VADEventStarted:
						p.textCh <- &textEvent{kind: evKindSpeechStarted, started: ev.Started}
					case VADEventStopped:
						p.textCh <- &textEvent{kind: evKindSpeechStopped, stopped: ev.Stopped}
					case VADEventFinal:
						p.sttCh <- ev.Final
					}
					p.notify()
				}
			}
		}
	}()
	// STT 线程：utterance → 输入门 → 转写 → textCh + promptCh
	go func() {
		for {
			select {
			case <-p.stopped:
				return
			case f := <-p.sttCh:
				p.sttStage(f)
			}
		}
	}()
	// LLM 线程：prompt → 流式 delta → textCh；分句 → ttsCh
	go func() {
		for {
			select {
			case <-p.stopped:
				return
			case req := <-p.promptCh:
				p.llmStage(req)
			}
		}
	}()
	// TTS 线程：句子 → gate 提交（首音频提交轮）→ 合成 → audioOut + done 哨兵
	go func() {
		for {
			select {
			case <-p.stopped:
				return
			case t := <-p.ttsCh:
				p.ttsStage(t)
			}
		}
	}()
	// 发送循环
	go p.sendLoop()
}

// sttStage STT 输入门 + 转写（规格 §4）。
func (p *Pipeline) sttStage(f *VADFinal) {
	key := turnKey{id: f.TurnID, rev: f.Rev}
	if p.completedFinalRevs[key] {
		return
	}
	// 输入门：final 稳定等待（processing deadline 或 delay−age）
	wait := time.Time{}
	if d, ok := p.turns.ProcessingDeadline(f.TurnID, f.Rev); ok && d.After(time.Now()) {
		wait = d
	} else if f.ProcessingDelayS > 0 {
		wait = time.Now().Add(time.Duration(f.ProcessingDelayS * float64(time.Second)))
	}
	if !p.turns.WaitForGate(f.TurnID, f.Rev, false, wait) {
		return
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

	itemID, _ := p.inputItemForTurn(f.TurnID, f.Rev, false)
	p.completedFinalRevs[key] = true
	p.completedOrder = append(p.completedOrder, key)
	for len(p.completedOrder) > maxCompletedFinals {
		delete(p.completedFinalRevs, p.completedOrder[0])
		p.completedOrder = p.completedOrder[1:]
	}
	term := p.pendingTerminals[itemID]
	if term == nil {
		term = &pendingInputTerminal{itemID: itemID, turnID: f.TurnID, rev: f.Rev}
		p.pendingTerminals[itemID] = term
	}
	term.transcription = evTranscriptionCompleted(itemID, text)
	slog.Info("Transcription completed (language=zh): chars=" + strconv.Itoa(len([]rune(text))) +
		" turn=" + f.TurnID + " rev=" + strconv.Itoa(f.Rev))

	// GenerateResponseRequest（text_prompt_queue）——转写 terminal 持发，随响应提交发布
	if text != "" && p.legs != nil && p.cfg.CreateResponse {
		p.muHist.Lock()
		// O4：同轮重开替换 user 文本（replace_user_message_text 同位），不新增条目
		tk := f.TurnID
		replaced := false
		for i := range p.chat {
			if p.chat[i].Role == "user" && p.chat[i].turnKey == tk {
				p.chat[i].Content = text
				replaced = true
				break
			}
		}
		if !replaced {
			p.chat = append(p.chat, ChatMessage{Role: "user", Content: text, turnKey: tk})
		}
		// O7：按轮裁剪（chat.py trim_if_needed 同位）
		p.trimChatTurnsLocked()
		snap := p.chatSnapshot()
		p.muHist.Unlock()
		rs := p.newResponseState(f.TurnID, f.Rev, f.SpeechEndAt)
		p.promptCh <- &promptRequest{resp: rs, userText: text, history: snap, speechStoppedAt: f.SpeechEndAt}
		p.notify()
	}
	p.notify()
}

// llmStage LLM 流 → delta 事件 + 分句入 TTS 队列。
func (p *Pipeline) llmStage(req *promptRequest) {
	rs := req.resp
	// dead 短路：barge-in flush 标记的排队请求不再生成（上游 LM 阶段查 cancel scope 同位）
	if rs.dead || p.closedKeys[rs.key] {
		return
	}
	llmStart := time.Now()
	history := req.history
	if len(history) > 0 {
		history = history[:len(history)-1] // 末条 user 由本次请求携带
	}
	sentenceCh := make(chan string, 64)
	done := make(chan error, 1)
	go func() {
		full, inT, outT, err := p.legs.Reply(rs.ctx, req.userText, history, rs.turnID, rs.rev, p.currentItemID(rs.turnID, rs.rev), func(delta string) {
			p.textCh <- &textEvent{kind: evKindAssistantDelta, resp: rs, transcript: delta}
			p.pendSentence(rs, delta, sentenceCh)
			p.notify()
		})
		// O1：完整回复作为暂存代际写入历史（add_provisional_generation_items 同位）。
		// 仅成功且未被取消时写入——上游失败响应 reject=全撤（S16），半截不进历史；
		// 取消回滚/完成 finalize 由 finishResponse 终态处置负责（O2/O3）
		if err == nil && rs.ctx.Err() == nil {
			p.muHist.Lock()
			p.writeAssistantAnchor(rs, full)
			p.trimChatTurnsLocked()
			p.muHist.Unlock()
		}
		rs.inTokens, rs.outTokens = inT, outT
		p.muLatency.Lock()
		p.lastLLM = time.Since(llmStart).Seconds()
		p.muLatency.Unlock()
		p.flushSentence(rs, sentenceCh)
		close(sentenceCh)
		done <- err
		_ = full
	}()
	first := true
	for sentence := range sentenceCh {
		p.ttsCh <- &ttsRequest{resp: rs, sentence: sentence, last: false}
		first = false
	}
	_ = first
	if err := <-done; err != nil && rs.ctx.Err() == nil && !rs.dead {
		slog.Warn("LLM 生成失败", "error", err)
		if !rs.dead {
			// 兜底语进转写面（F1，2026-11-05 ws_probe 实弹）：直推 TTS 会绕过
			// delta/parts——用户听到的话在转写面板空白、done 文本为空。
			p.textCh <- &textEvent{kind: evKindAssistantDelta, resp: rs, transcript: voiceLLMFallbackText}
			p.notify()
			p.ttsCh <- &ttsRequest{resp: rs, sentence: voiceLLMFallbackText, last: true}
			return
		}
	}
	// 正常结束：给 TTS 线程发最后一句标记（done 哨兵跟在其合成后）
	p.ttsCh <- &ttsRequest{resp: rs, sentence: "\x00mark", last: true}
}

// ttsStage gate 提交（首音频提交轮）→ 合成 → 音频块 + done 哨兵。
func (p *Pipeline) ttsStage(t *ttsRequest) {
	rs := t.resp
	if t.sentence == "\x00mark" {
		p.audioOut <- &audioOut{resp: rs, done: true}
		return
	}
	g := p.turns.Gate(rs.turnID, rs.rev, true, time.Time{}, true)
	if g.Action == GateHold {
		if !p.turns.WaitForGate(rs.turnID, rs.rev, true, time.Time{}) {
			return
		}
	} else if g.Action == GateDrop {
		return
	}
	ttfaStart := time.Now()
	firstAudio := true
	err := p.legs.Speak(rs.ctx, t.sentence, func(pcm []byte) {
		if firstAudio {
			firstAudio = false
			// 播放期回声门开（vadhandler EchoGuardOn 注释：监听 0.5/播放 0.75）
			p.vad.EchoGuardOn()
			p.muLatency.Lock()
			p.lastTTFA = time.Since(ttfaStart).Seconds()
			if !rs.speechAnchor.IsZero() {
				p.lastE2E = time.Since(rs.speechAnchor).Seconds()
			}
			p.muLatency.Unlock()
		}
		p.audioOut <- &audioOut{resp: rs, pcm: pcm, gen: rs.generation}
		p.notify()
	})
	if err != nil && rs.ctx.Err() == nil {
		slog.Warn("TTS 句失败", "error", err)
	}
	if t.last {
		p.audioOut <- &audioOut{resp: rs, done: true, gen: rs.generation}
		p.notify()
	}
}

// voiceLLMFallbackText LLM 失败兜底语（单一来源：转写面与 TTS 面同文本）。
const voiceLLMFallbackText = "我现在有点跟不上，请再说一遍。"

// ─────────── 发送循环（websocket_router._send_loop_for 同位） ───────────

func (p *Pipeline) sendLoop() {
	for {
		// HOLD 重试语义（SEND_LOOP_HOLD_RETRY_S=10ms）：有停靠项时短周期复查；
		// 空闲 50ms 安全网兜底（notify 是快路径，定时器防信号链断裂）
		wait := 50 * time.Millisecond
		if len(p.pendingText) > 0 || p.parkedAudio != nil {
			wait = 10 * time.Millisecond
		}
		select {
		case <-p.stopped:
			return
		case reason := <-p.cancelCh:
			p.cancelActiveResponse(reason)
			continue
		case <-p.signal:
		case <-time.After(wait):
		}
		for i := 0; i < 4096; i++ {
			if !p.drainOnce() {
				break
			}
		}
	}
}

// drainOnce 单次：文本侧优先 → resolve terminals → 音频；返回是否还有活干。
func (p *Pipeline) drainOnce() bool {
	// ① 停靠文本复查（HOLD 重试语义：发送循环每轮复查停靠列表）
	for i, ev := range p.pendingText {
		if !p.outputIsHeld(ev) {
			p.pendingText = append(p.pendingText[:i], p.pendingText[i+1:]...)
			p.dispatchText(ev)
			return true
		}
	}
	select {
	case ev := <-p.textCh:
		if p.outputIsHeld(ev) {
			p.pendingText = append(p.pendingText, ev)
			return false
		}
		p.dispatchText(ev)
		return true
	default:
	}
	// ② resolve terminals（每轮）
	p.resolveInputTerminals("")
	// ③ 停靠音频
	if p.parkedAudio != nil && !p.audioHeld(p.parkedAudio) {
		a := p.parkedAudio
		p.parkedAudio = nil
		p.dispatchAudio(a)
		return true
	}
	// ④ 音频队列
	select {
	case a := <-p.audioOut:
		// router:1134 同位——代际丢弃（含 done 哨兵）先于一切处理
		if p.scope.discardable(a.gen) {
			return true
		}
		if a.done {
			if p.closedKeys[a.resp.key] {
				return true // router:1138 墓葬同位
			}
			p.scope.clearDiscarding()
			// R3（audit 复核池）：零输出响应不得谎报 completed——空回复/丢弃
			// 残留按 cancelled 收口，O3 回滚撤销空暂存，前端不再误报完成。
			if a.resp.audioEmitted == 0 {
				p.finishResponse(p.session, a.resp, "cancelled", "empty_output")
				return true
			}
			p.finishResponse(p.session, a.resp, "completed", "")
			return true
		}
		if p.audioHeld(a) {
			if p.parkedAudio == nil {
				p.parkedAudio = a
			}
			return false
		}
		p.dispatchAudio(a)
		return true
	default:
	}
	return false
}

// outputIsHeld 文本事件停靠判定（宽限+候选持有期间 assistant 输出被持）。
func (p *Pipeline) outputIsHeld(ev *textEvent) bool {
	if ev.kind == evKindAssistantDelta {
		rs := ev.resp
		if rs.dead || p.closedKeys[rs.key] {
			return false
		}
		g := p.turns.Gate(rs.turnID, rs.rev, true, time.Time{}, false)
		return g.Action == GateHold
	}
	return false
}

func (p *Pipeline) audioHeld(a *audioOut) bool {
	if a.done {
		return false
	}
	g := p.turns.Gate(a.resp.turnID, a.resp.rev, true, time.Time{}, false)
	return g.Action == GateHold
}

// dispatchText 协议翻译（service.dispatch_pipeline_event 同位；发送循环单线程）。
func (p *Pipeline) dispatchText(ev *textEvent) {
	switch ev.kind {
	case evKindSpeechStarted:
		// 规格顺序：先处置旧 terminal（started_turn_id 绑定）→ barge-in → started 帧
		p.resolveInputTerminals(ev.started.TurnID)
		rs := p.respState
		if p.inResponse && rs != nil && ev.started.InterruptResponse {
			p.cancelActiveResponse("turn_detected")
			slog.Info("Pipeline 0: speech during response: cancelled, queue flushed")
		}
		isReopen := ev.started.Reopened && ev.started.TurnID != ""
		itemID, existed := p.inputItemForTurn(ev.started.TurnID, ev.started.Rev, isReopen)
		if existed && isReopen {
			return // 复用 item，不重发 started（规格 §5.1）
		}
		if p.session != nil {
			p.session.Send(evSpeechStarted(itemID, ev.started.AudioStartMs))
		}
	case evKindSpeechStopped:
		p.holdSpeechStopped(ev.stopped)
	case evKindTranscription:
		// terminal 持发，随 resolve/提交发布
	case evKindAssistantDelta:
		rs := ev.resp
		if rs.dead || p.closedKeys[rs.key] {
			return
		}
		// 首增量：提交轮 → resolve（user terminal 先行）→ created → 输出帧（规格 §5.3）
		if !rs.committedSent {
			g := p.turns.Gate(rs.turnID, rs.rev, false, time.Time{}, true)
			if g.Action != GateAccept {
				rs.dead = true
				return
			}
			rs.committedSent = true
			p.muLatency.Lock()
			p.lastLLMTTFT = time.Since(rs.startedAt).Seconds()
			p.muLatency.Unlock()
			p.resolveInputTerminals("")
		}
		p.sendResponseCreated(rs)
		p.beginAssistantOutput(rs)
		p.streamAssistantText(rs, ev.transcript)
		rs.parts = append(rs.parts, ev.transcript)
	}
}

// dispatchAudio 音频批 6400B（MAX_AUDIO_BATCH_BYTES 同位）。
func (p *Pipeline) dispatchAudio(a *audioOut) {
	if p.closedKeys[a.resp.key] {
		return
	}
	a.resp.audioEmitted++
	p.sendResponseCreated(a.resp)
	p.beginAssistantOutput(a.resp)
	p.audioBatch = append(p.audioBatch, a.pcm...)
	for len(p.audioBatch) >= 6400 {
		chunk := make([]byte, 6400)
		copy(chunk, p.audioBatch[:6400])
		p.audioBatch = p.audioBatch[6400:]
		if p.session != nil {
			p.session.Send(evAudioDelta(a.resp.id, a.resp.itemID, chunk))
		}
	}
}

// holdSpeechStopped speech_stopped 挂 terminal（不立即下行）。
func (p *Pipeline) holdSpeechStopped(st *VADStopped) {
	key := turnKey{id: st.TurnID, rev: st.Rev}
	itemID, ok := p.inputItemByTurnRev[key]
	if !ok {
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

// resolveInputTerminals 持发处置（规格 §5.1 规则表；发送循环单线程）。
// startedTurnID：speech_started 触发的处置轮（其余调用传空）。
func (p *Pipeline) resolveInputTerminals(startedTurnID string) {
	for itemID, term := range p.pendingTerminals {
		var disposition string
		if term.turnID == "" || !p.turns.IsLatest(term.turnID, term.rev) {
			if p.turns.IsCurrentTurn(term.turnID) {
				disposition = "discard" // 同 live turn 的超龄 revision（客户端从未见过）
			} else {
				term.inputClosed = true
				disposition = "publish"
			}
		} else if startedTurnID != "" && term.turnID != startedTurnID {
			disposition = "publish" // 新轮开始时处置旧轮 terminal（sidecar 同守卫）
		} else {
			// 规格规则表原序：unanswered/failed → gate 提交；is_committed → publish；else hold
			hasResponse := (p.inResponse && (p.responseTurnID == "" || p.responseTurnID == term.turnID)) ||
				p.turns.IsCommitted(term.turnID, term.rev)
			unanswered := term.inputClosed && !hasResponse
			failed := term.transcription != nil &&
				term.transcription["type"] == "conversation.item.input_audio_transcription.failed"
			switch {
			case unanswered || failed:
				g := p.turns.Gate(term.turnID, term.rev, false, time.Time{}, true)
				switch g.Action {
				case GateHold:
					disposition = "hold"
				case GateAccept:
					if !hasResponse {
						p.turns.Close(term.turnID, term.rev)
					}
					disposition = "publish"
				default:
					disposition = "discard"
				}
			case hasResponse || p.turns.IsCommitted(term.turnID, term.rev):
				disposition = "publish"
			default:
				disposition = "hold"
			}
		}
		switch disposition {
		case "publish":
			p.publishTerminal(term)
		case "discard":
			delete(p.pendingTerminals, itemID)
		}
	}
}

// publishTerminal 发布顺序：speech_stopped → committed → item.created → transcription。
func (p *Pipeline) publishTerminal(term *pendingInputTerminal) {
	if p.session == nil {
		return
	}
	if term.speechStopped != nil {
		p.session.Send(term.speechStopped)
		term.speechStopped = nil
	}
	p.session.Send(map[string]any{
		"type":             "input_audio_buffer.committed",
		"item_id":          term.itemID,
		"previous_item_id": p.lastItemID,
	})
	p.session.Send(evItemCreated(map[string]any{
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
		p.session.Send(term.transcription)
		term.transcription = nil
	}
	if term.inputClosed {
		delete(p.pendingTerminals, term.itemID)
	}
}

// ─────────── response 装配（发送循环调用） ───────────

type responseState struct {
	id            string
	key           string
	turnID        string
	rev           int
	itemID        string
	generation    int
	cancel        context.CancelFunc
	ctx           context.Context
	startedAt     time.Time
	speechAnchor  time.Time
	createdSent   bool
	committedSent bool
	dead          bool
	textSent      bool
	// audioEmitted 过闸下行的音频批次数（dispatchAudio 计数）。done 哨兵收口时
	// 为 0=整响应零输出（LLM 空回复/代际丢弃残留）——按 cancelled 收口走 O3
	// 回滚，不再以 completed+空输出谎报完成（audit R3 刀6 复核池销项）。
	audioEmitted int
	inTokens     int
	outTokens    int
	parts        []string
	sb           string
}

func (p *Pipeline) newResponseState(turnID string, rev int, anchor time.Time) *responseState {
	ctx, cancel := context.WithCancel(context.Background())
	rs := &responseState{
		id:           "resp_" + randHex(6),
		key:          randHex(16),
		turnID:       turnID,
		rev:          rev,
		itemID:       "item_" + randHex(8),
		generation:   p.scope.current(),
		cancel:       cancel,
		ctx:          ctx,
		startedAt:    time.Now(),
		speechAnchor: anchor,
	}
	p.inResponse = true
	p.respState = rs
	p.responseTurnID = turnID
	return rs
}

func (p *Pipeline) sendResponseCreated(rs *responseState) {
	if rs.createdSent {
		return
	}
	rs.createdSent = true
	if p.session != nil {
		p.session.Send(evResponseCreated(rs.id))
	}
}

func (p *Pipeline) beginAssistantOutput(rs *responseState) {
	if rs.textSent {
		return
	}
	rs.textSent = true
	if p.session != nil {
		p.session.Send(evOutputItemAdded(rs.id, rs.itemID, 0))
		p.session.Send(evContentPartAdded(rs.id, rs.itemID))
	}
}

func (p *Pipeline) streamAssistantText(rs *responseState, delta string) {
	if delta == "" || p.session == nil {
		return
	}
	p.session.Send(evTranscriptDelta(rs.id, rs.itemID, delta))
}

// finishResponse 终态装配 + tracker close + 墓碑（发送循环）。
// 终态守卫：cancelled 后 TTS 队列残余的 done 哨兵不得重复装配（规格：一响应一终态）。
func (p *Pipeline) finishResponse(s *Session, rs *responseState, status, reason string) {
	// response.py:991 同位：key 与当前响应不符（含已终结）→ 空操作。
	// 双终态防护由代际丢弃（哨兵过不了闸）+ 此守卫共同承担，不用手搓标志。
	if p.respState != nil && p.respState.key != rs.key {
		return
	}
	if p.respState == nil && p.closedKeys[rs.key] {
		return
	}
	// 播放期回声门关（含 barge-in cancelled 路径：本函数是全部终态唯一收口）。
	p.vad.EchoGuardOff()
	// 终态前冲刷本响应的停靠 delta（grace/候选窗口内入队的兜底语等）：
	// 终态帧之后不允许再有本响应文本帧，且 parts 必须含全部已播报文本——
	// 否则 done 转写空白、面板无声（F1 配套，ws_probe 实弹钉）。
	remaining := p.pendingText[:0]
	for _, ev := range p.pendingText {
		if ev.resp.key == rs.key {
			p.dispatchText(ev)
			continue
		}
		remaining = append(remaining, ev)
	}
	p.pendingText = remaining
	full := strings.Join(rs.parts, " ")
	if rs.textSent && s != nil {
		s.Send(evTranscriptDone(rs.id, rs.itemID, full))
		s.Send(evContentPartDone(rs.id, rs.itemID))
		s.Send(evOutputItemDone(rs.id, rs.itemID, 0))
	}
	// 尾批 flush：6400B 批的 remainder 必须在响应收口时下行（上游 audio 输出
	// flush-remainder 语义；漏 flush=每轮尾部 ≤200ms 音频被静默吞掉）。仅
	// completed 下发——cancelled 的尾部残余随播放冲刷语义一并丢弃。
	if status == "completed" && s != nil && len(p.audioBatch) > 0 {
		tail := p.audioBatch
		p.audioBatch = nil
		s.Send(evAudioDelta(rs.id, rs.itemID, tail))
	}
	if s != nil {
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
			"speech_to_speech.turn_latency": p.latencyMetadata(rs, status),
		}
		s.Send(done)
	}
	p.muLatency.Lock()
	lat := latencyRec{sttS: p.lastSTT, llmS: p.lastLLM, ttsTTFA: p.lastTTFA, e2eS: p.lastE2E,
		vadDecideS: p.lastVADDecision, holdS: p.lastHold, smart: p.lastSmart}
	p.muLatency.Unlock()
	slogLatencyLine(rs, lat, status)
	// O2/O3 终态处置暂存代际（handlers/history.py close()/finalize 同位）：
	// completed → finalize 转永久；cancelled/failed → 回滚撤销该响应写入的内容
	p.muHist.Lock()
	if status == "completed" {
		for i := range p.chat {
			if p.chat[i].Role == "assistant" && p.chat[i].respKey == rs.key {
				p.chat[i].provisional = false
			}
		}
	} else {
		out := p.chat[:0]
		for _, m := range p.chat {
			if m.Role == "assistant" && m.respKey == rs.key {
				continue
			}
			out = append(out, m)
		}
		p.chat = out
	}
	p.muHist.Unlock()
	p.muTerm.Lock()
	p.turns.Close(rs.turnID, rs.rev)
	p.closedKeys[rs.key] = true
	p.closedOrder = append(p.closedOrder, rs.key)
	for len(p.closedOrder) > maxClosedKeys {
		delete(p.closedKeys, p.closedOrder[0])
		p.closedOrder = p.closedOrder[1:]
	}
	if p.respState == rs {
		p.inResponse = false
		p.respState = nil
	}
	p.muTerm.Unlock()
	p.resolveInputTerminals("")
	p.notify()
}

// latencyMetadata response.done metadata（§5.8）。
func (p *Pipeline) latencyMetadata(rs *responseState, status string) map[string]any {
	p.muLatency.Lock()
	defer p.muLatency.Unlock()
	return map[string]any{
		"version":        2,
		"source":         "speech",
		"turn_id":        rs.turnID,
		"turn_revision":  rs.rev,
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

// ─────────── 服务态辅助 ───────────

func (p *Pipeline) inputItemForTurn(turnID string, rev int, isReopen bool) (string, bool) {
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

func (p *Pipeline) currentItemID(turnID string, rev int) string {
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

// writeAssistantAnchor O1 暂存写入 + R2 anchor 写回（language_model.py:631/679 同位）：
// assistant 落在本轮 user 条目之后——生成期 soft-end 非打断语音（vadhandler 补发路径
// InterruptResponse=false）追加的更新 user 不被本轮回复越过。须持 muHist。
func (p *Pipeline) writeAssistantAnchor(rs *responseState, full string) {
	insertAt := len(p.chat)
	for i, m := range p.chat {
		if m.Role == "user" && m.turnKey == rs.turnID {
			insertAt = i + 1
			// 越过该轮已有的 assistant（同轮重开 replace 场景写入位置仍在其后）
			for insertAt < len(p.chat) && p.chat[insertAt].Role == "assistant" {
				insertAt++
			}
			break
		}
	}
	p.chat = append(p.chat, ChatMessage{})
	copy(p.chat[insertAt+1:], p.chat[insertAt:])
	p.chat[insertAt] = ChatMessage{Role: "assistant", Content: full, respKey: rs.key, provisional: true}
}

// trimChatTurnsLocked O7 同位：user_turn_count > size 时同步逐出最旧完整轮
// （user 及其紧邻 assistant）；须持 muHist 调用。
func (p *Pipeline) trimChatTurnsLocked() {
	userTurns := 0
	for _, m := range p.chat {
		if m.Role == "user" {
			userTurns++
		}
	}
	for userTurns > p.chatSize && len(p.chat) > 0 {
		drop := 1
		for drop < len(p.chat) && p.chat[drop].Role == "assistant" {
			drop++
		}
		p.chat = p.chat[drop:]
		userTurns--
	}
}

// chatSnapshot 快照（须持 muHist 调用）。
func (p *Pipeline) chatSnapshot() []ChatMessage {
	out := make([]ChatMessage, len(p.chat))
	copy(out, p.chat)
	return out
}

// pendSentence/flushSentence 分句批切（stream_batch_sentences=1）。
func (p *Pipeline) pendSentence(rs *responseState, delta string, ch chan<- string) {
	rs.sb += delta
	sentences := spokenSplit(rs.sb)
	if len(sentences) > 1 {
		complete := strings.Join(sentences[:len(sentences)-1], "")
		rs.sb = sentences[len(sentences)-1]
		ch <- complete
	}
}

func (p *Pipeline) flushSentence(rs *responseState, ch chan<- string) {
	rest := strings.TrimSpace(rs.sb)
	rs.sb = ""
	if rest != "" {
		ch <- rest
	}
}
