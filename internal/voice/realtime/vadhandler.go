// vadhandler.go — VAD 管线处理器移植（sidecar VAD/vad_handler.py gaea 生效面，规格 §2）。
//
// 覆盖：延迟 SpeechStarted 确认（384/192ms+候选先行）、幻影触发、discard、
// soft-end（Smart Turn 宽限表 + 前缀拼接 + final 事件）。不移植：progressive
// 快照、流式转写、diarization、short-segment merge（gaea 配置恒 0=禁用）。
// 日志行逐字保 sidecar 格式（eot_analyze.py/false_interrupt_bound.py 解析依赖）。
package realtime

import (
	"log/slog"
	"math"
	"strconv"
	"time"
)

// EOTAnalyzer Smart Turn 分析器接口。
type EOTAnalyzer interface {
	Predict(pcm []float32) (probability float32, took time.Duration, err error)
}

// VADHandlerConfig gaea 生效配置（gaea-spike.yaml + 2026-11-01 实弹提速值）。
type VADHandlerConfig struct {
	SampleRate                 int     // 16000
	Threshold                  float64 // 0.5（silero 官方默认）；session.update 可运行时改
	MinSilenceMs               int     // 64
	MinSpeechMs                float64 // 384
	MinSpeechContinuationMs    float64 // 192
	SpeechPadMs                int     // 500
	SpeculativeReopenMs        int     // 800
	UnansweredReopenMs         int     // 7000
	SmartTurnMaxWaitMs         int     // 1200
	SmartTurnIncompleteDelayMs int     // 400
	MaxSpeechMs                float64 // +Inf
}

// DefaultVADConfig gaea 生产配置。
func DefaultVADConfig() VADHandlerConfig {
	return VADHandlerConfig{
		SampleRate:                 16000,
		Threshold:                  0.5,
		MinSilenceMs:               64,
		MinSpeechMs:                384,
		MinSpeechContinuationMs:    192,
		SpeechPadMs:                500,
		SpeculativeReopenMs:        800,
		UnansweredReopenMs:         7000,
		SmartTurnMaxWaitMs:         1200,
		SmartTurnIncompleteDelayMs: 400,
		MaxSpeechMs:                math.Inf(1),
	}
}

// VADEventKind VAD 向管线发射的事件三态。
type VADEventKind int

const (
	VADEventStarted VADEventKind = iota
	VADEventStopped
	VADEventFinal
)

// VADStarted SpeechStartedEvent。
type VADStarted struct {
	AudioStartMs      int
	TurnID            string
	Rev               int
	Reopened          bool
	InterruptResponse bool // true=确认路径；false=soft-end 补发路径
}

// VADStopped SpeechStoppedEvent（accepted 路径 DurationS 为合并音频秒）。
type VADStopped struct {
	DurationS  float64
	AudioEndMs int
	TurnID     string
	Rev        int
}

// VADFinal mode=final 的 VADAudio 等价物。
type VADFinal struct {
	Audio            []float32
	TurnID           string
	Rev              int
	ProcessingDelayS float64
	SpeechEndAt      time.Time
	HasSpeechEnd     bool
	VADDecisionS     float64
	SmartStatus      string
	EndMs            int
	SegmentMs        float64
	ActiveMs         float64
}

// VADEvent 单块处理可能产生的事件（0~3 个：started/stopped/final）。
type VADEvent struct {
	Kind    VADEventKind
	Started *VADStarted
	Stopped *VADStopped
	Final   *VADFinal
}

// VADHandler 单会话 VAD 处理器（单音频 goroutine 持有，非并发安全）。
type VADHandler struct {
	cfg   VADHandlerConfig
	iter  *vadIterator
	turns *SpeculativeTurns
	eot   EOTAnalyzer

	totalSamples         int
	speechStartedEmitted bool
	audioPrefix          []float32 // 已定稿 revision 的输出音频前缀
	rawPrefix            []float32 // 已定稿 revision 的原始分析前缀（喂 Smart Turn）

	lastSmartStatus   string
	lastSmartAnalysis time.Duration

	receivedAt     time.Time
	baseThreshold  float64 // session.update 设定的监听阈值（回声门回落基准）
	echoGuard      bool    // 播放期回声门开关（ttsStage 首音频开/finishResponse 关）
}

// echoGuardThreshold 播放期回声门阈值（2026-11-01 实弹：0.6 被扬声器回声触发
// 假插话；0.75 是实弹定值，收敛进播放期专用）。
const echoGuardThreshold = 0.75

// NewVADHandler 构造（scorer/eot 由调用方注入；turns 为管线共享实例）。
func NewVADHandler(cfg VADHandlerConfig, scorer VadScorer, eot EOTAnalyzer, turns *SpeculativeTurns) *VADHandler {
	unanswered := maxInt(cfg.SpeculativeReopenMs, maxInt(cfg.UnansweredReopenMs, cfg.SmartTurnMaxWaitMs))
	turns.ConfigureReopen(unanswered)
	return &VADHandler{
		cfg:             cfg,
		iter:            newVADIterator(scorer, cfg.Threshold, cfg.SampleRate, cfg.MinSilenceMs, cfg.SpeechPadMs),
		turns:           turns,
		eot:             eot,
		lastSmartStatus: "disabled",
		baseThreshold:   cfg.Threshold,
	}
}

// SetThreshold session.update 运行时阈值（也是回声门关后的回落基准）。
func (h *VADHandler) SetThreshold(v float64) {
	h.baseThreshold = v
	h.iter.threshold = v
	slog.Info("VAD threshold updated to " + strconv.FormatFloat(v, 'g', -1, 64))
}

// EchoGuardOn 播放期回声门（2026-11-01 实弹口径：TTS 出声期间 threshold 抬到
// 0.75 防扬声器漏回的残余回声触发假插话；2026-11-05 实弹：恒 0.75 把弱电平真麦
// 全部 discard——segment 820ms 里 active 仅 64~192ms，过不了 384ms 门）。监听期
// 用 silero 官方默认 0.5 救弱麦，播放期回 0.75 保回声防线。与 SetThreshold 同款
// 跨 goroutine 直写（float64 单写竞态最坏一帧旧值）。
func (h *VADHandler) EchoGuardOn() {
	if h.echoGuard {
		return
	}
	h.echoGuard = true
	h.iter.threshold = echoGuardThreshold
}

// EchoGuardOff 播放结束（finishResponse 终态收口）回落监听阈值。
func (h *VADHandler) EchoGuardOff() {
	if !h.echoGuard {
		return
	}
	h.echoGuard = false
	h.iter.threshold = h.baseThreshold
}

// SetMinSilenceMs 运行时改静音断句窗（M14，vad_handler.py:237-239 同位；
// session.update silence_duration_ms——此前协议层解析即丢恒 64ms）。负值钳 0。
func (h *VADHandler) SetMinSilenceMs(ms int) {
	if ms < 0 {
		ms = 0
	}
	h.iter.minSilence = h.cfg.SampleRate * ms / 1000
}

// AudioMs 累计输入音频毫秒。
func (h *VADHandler) AudioMs() int {
	return int(float64(h.totalSamples) / float64(h.cfg.SampleRate) * 1000)
}

// Process 一块 512 样本（float32 /32768 已归一）。返回本块产生的有序事件。
func (h *VADHandler) Process(x []float32) []VADEvent {
	h.totalSamples += len(x)
	h.receivedAt = time.Now()
	utterance := h.iter.process(x)
	decisionAt := time.Now()

	if utterance != nil {
		return h.onUtterance(utterance, decisionAt)
	}
	if h.iter.triggered && !h.speechStartedEmitted {
		if ev := h.confirmSpeechStart(); ev != nil {
			return []VADEvent{*ev}
		}
	}
	return nil
}

// ensureTurnForSpeechStart speech_started 统一入口（上游 _ensure_turn_for_speech_start
// 同位，Go 刀3 移植漏抄的守卫）：开新轮（非重开、rev=0、本段 started 未发）即清
// audioPrefix/rawPrefix——前缀只服务同轮 reopen 的音频拼接；跨轮不清则每轮 final
// 音频=全会话历史语音，STT 逐轮重转全文（实弹 2026-11-06：转写字符数 2→6→…→54
// 滚雪球，角色答非所问+输入复读循环）。
func (h *VADHandler) ensureTurnForSpeechStart(audioMs int) (string, int, bool) {
	turnID, rev, reopened := h.turns.SpeechStarted(audioMs)
	if !reopened && rev == 0 && !h.speechStartedEmitted {
		h.audioPrefix = nil
		h.rawPrefix = nil
	}
	return turnID, rev, reopened
}

// confirmSpeechStart 延迟确认（384/192ms；候选先行）。
func (h *VADHandler) confirmSpeechStart() *VADEvent {
	segmentMs := float64(bufferSamples(h.iter.buffer)) / float64(h.cfg.SampleRate) * 1000
	activeMs := float64(h.iter.activeSpeech) / float64(h.cfg.SampleRate) * 1000
	speechBufferMs := float64(bufferSamples(h.iter.speechBuffer())) / float64(h.cfg.SampleRate) * 1000
	startMs := max0(h.AudioMs() - int(speechBufferMs))

	// 候选先行（HOLD ≤2s），再取 active_min
	h.turns.SpeechCandidateStarted(startMs)
	activeMin := h.cfg.MinSpeechMs
	if h.turns.HasSpeechCandidate() || h.turns.CanReopen(startMs) {
		activeMin = h.cfg.MinSpeechContinuationMs
	}
	if activeMs < activeMin {
		return nil
	}
	turnID, rev, reopened := h.ensureTurnForSpeechStart(startMs)
	h.speechStartedEmitted = true
	slog.Info("Speech started (confirmed, active=" + f0(activeMs) + "ms, min=" + f0(activeMin) +
		"ms, segment=" + f0(segmentMs) + "ms, turn=" + turnID + " rev=" + strconv.Itoa(rev) + ")")
	return &VADEvent{Kind: VADEventStarted, Started: &VADStarted{
		AudioStartMs:      startMs,
		TurnID:            turnID,
		Rev:               rev,
		Reopened:          reopened,
		InterruptResponse: true,
	}}
}

// onUtterance 段结束三分支（幻影/discard/soft-end）。
func (h *VADHandler) onUtterance(utterance [][]float32, decisionAt time.Time) []VADEvent {
	// 幻影触发：空 utterance
	if len(utterance) == 0 {
		slog.Info("VAD: phantom trigger (empty buffer), closing speech pair")
		var out []VADEvent
		curTurn, curRev := h.turns.CurrentTurn()
		if h.speechStartedEmitted {
			st := &VADStopped{AudioEndMs: h.AudioMs(), TurnID: curTurn, Rev: curRev}
			out = append(out, VADEvent{Kind: VADEventStopped, Stopped: st})
			h.turns.SegmentDiscarded()
		} else {
			h.turns.SpeechCandidateCancelled()
		}
		h.speechStartedEmitted = false
		return out
	}

	array := catF32(utterance)
	endMs := h.AudioMs()
	sr := float64(h.cfg.SampleRate)
	rawActiveMs := float64(h.iter.lastUtterActive) / sr * 1000
	activeMs := rawActiveMs
	startMs := max0(endMs - int(float64(len(array))/sr*1000))
	durationMs := float64(len(array)) / sr * 1000
	minActiveMs := 0.0
	if !h.speechStartedEmitted {
		minActiveMs = h.cfg.MinSpeechMs
		if h.turns.HasSpeechCandidate() || h.turns.CanReopen(startMs) {
			minActiveMs = h.cfg.MinSpeechContinuationMs
		}
	}

	// discard 路径
	if activeMs < minActiveMs || durationMs > h.cfg.MaxSpeechMs {
		slog.Info("VAD: discarding segment=" + f0(durationMs) + "ms active=" + f0(activeMs) +
			"ms (active_min=" + f0(minActiveMs) + "ms, segment_max=infms)")
		var out []VADEvent
		curTurn, curRev := h.turns.CurrentTurn()
		if h.speechStartedEmitted {
			out = append(out, VADEvent{Kind: VADEventStopped, Stopped: &VADStopped{
				AudioEndMs: endMs, TurnID: curTurn, Rev: curRev}})
			h.turns.SegmentDiscarded()
		} else {
			h.turns.SpeechCandidateCancelled()
		}
		h.speechStartedEmitted = false
		return out
	}

	// —— soft-end 接受路径 ——
	var turnID string
	var rev int
	var out []VADEvent
	if !h.speechStartedEmitted {
		var reopened bool
		turnID, rev, reopened = h.ensureTurnForSpeechStart(startMs)
		h.speechStartedEmitted = true
		// 补发路径 interrupt_response=False（§2）
		out = append(out, VADEvent{Kind: VADEventStarted, Started: &VADStarted{
			AudioStartMs:      startMs,
			TurnID:            turnID,
			Rev:               rev,
			Reopened:          reopened,
			InterruptResponse: false,
		}})
	} else {
		turnID, rev = h.turns.CurrentTurn()
	}
	slog.Info("Speech soft-ended (segment=" + f0(durationMs) + "ms, active=" + f0(activeMs) +
		"ms, turn=" + turnID + " rev=" + strconv.Itoa(rev) + ")")

	// Smart Turn：分析音频=前次 raw 前缀+本段（未增强）
	analysisAudio := make([]float32, 0, len(h.rawPrefix)+len(array))
	analysisAudio = append(analysisAudio, h.rawPrefix...)
	analysisAudio = append(analysisAudio, array...)
	reopenGraceMs, processingDelayMs := h.smartTurnTiming(analysisAudio)

	var speechEndAt time.Time
	hasSpeechEnd := false
	var vadDecisionS float64
	if h.iter.hasSpeechEnd && h.iter.lastSpeechEnd >= 0 {
		samplesAfter := h.totalSamples - h.iter.lastSpeechEnd
		if samplesAfter < 0 {
			samplesAfter = 0
		}
		speechEndAt = h.receivedAt.Add(-time.Duration(float64(samplesAfter) / sr * float64(time.Second)))
		hasSpeechEnd = true
		vadDecisionS = max0f(decisionAt.Sub(speechEndAt).Seconds())
	}

	// 输出音频=前次前缀+本段
	outputArray := make([]float32, 0, len(h.audioPrefix)+len(array))
	outputArray = append(outputArray, h.audioPrefix...)
	outputArray = append(outputArray, array...)
	combinedDurationS := float64(len(outputArray)) / sr

	out = append(out, VADEvent{Kind: VADEventStopped, Stopped: &VADStopped{
		DurationS: combinedDurationS, AudioEndMs: endMs, TurnID: turnID, Rev: rev}})
	h.audioPrefix = outputArray
	h.rawPrefix = analysisAudio
	h.turns.SegmentFinalized(endMs, reopenGraceMs, processingDelayMs)
	h.speechStartedEmitted = false

	out = append(out, VADEvent{Kind: VADEventFinal, Final: &VADFinal{
		Audio:            outputArray,
		TurnID:           turnID,
		Rev:              rev,
		ProcessingDelayS: float64(processingDelayMs) / 1000.0,
		SpeechEndAt:      speechEndAt,
		HasSpeechEnd:     hasSpeechEnd,
		VADDecisionS:     vadDecisionS,
		SmartStatus:      h.lastSmartStatus,
		EndMs:            endMs,
		SegmentMs:        durationMs,
		ActiveMs:         activeMs,
	}})
	return out
}

// smartTurnTiming 规格表：complete→(800,0)；incomplete→(1200,400)；failed/disabled→(800,0)。
func (h *VADHandler) smartTurnTiming(audio []float32) (reopenGraceMs int, processingDelayMs int) {
	if h.eot == nil {
		h.lastSmartStatus = "disabled"
		h.lastSmartAnalysis = 0
		return h.cfg.SpeculativeReopenMs, 0
	}
	p, took, err := h.eot.Predict(audio)
	if err != nil {
		h.lastSmartStatus = "failed"
		h.lastSmartAnalysis = took
		slog.Warn("Smart Turn inference failed; using the default speculative reopen grace", "error", err)
		return h.cfg.SpeculativeReopenMs, 0
	}
	h.lastSmartAnalysis = took
	if p > 0.5 {
		h.lastSmartStatus = "complete"
		slog.Info("Smart Turn: complete (p=" + f3(float64(p)) + ", " + f1(took.Seconds()*1000) + "ms); using " +
			strconv.Itoa(h.cfg.SpeculativeReopenMs) + "ms speculative reopen grace")
		return h.cfg.SpeculativeReopenMs, 0
	}
	delay := minInt(h.cfg.SmartTurnIncompleteDelayMs, h.cfg.SmartTurnMaxWaitMs)
	h.lastSmartStatus = "incomplete"
	slog.Info("Smart Turn: incomplete (p=" + f3(float64(p)) + ", " + f1(took.Seconds()*1000) + "ms); using " +
		strconv.Itoa(h.cfg.SmartTurnMaxWaitMs) + "ms speculative reopen grace and delaying processing by " +
		strconv.Itoa(delay) + "ms")
	return h.cfg.SmartTurnMaxWaitMs, delay
}

// OnSessionEnd 会话结束全量复位。
func (h *VADHandler) OnSessionEnd() {
	h.iter.resetStates()
	h.iter.threshold = h.cfg.Threshold
	h.iter.minSilence = h.cfg.SampleRate * h.cfg.MinSilenceMs / 1000 // M14：回落构造值
	h.totalSamples = 0
	h.speechStartedEmitted = false
	h.audioPrefix = nil
	h.rawPrefix = nil
	h.lastSmartStatus = "disabled"
	// M1 蒸馏（Go 新增态自欠清理；上游无此机制故无同位锚）：echoGuard/baseThreshold
	// 不复位则「播放中断线」后下会话 EchoGuardOn 早退空转——首轮播放回声门静默
	// 失效，扬声器回声触发假 barge-in（2026-11-01/05 根修症状复发窗口）。
	h.echoGuard = false
	h.baseThreshold = h.cfg.Threshold
}

// —— 辅助 ——

func bufferSamples(chunks [][]float32) int {
	n := 0
	for _, c := range chunks {
		n += len(c)
	}
	return n
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max0f(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

// f0/f1/f3 %.0f/%.1f/%.3f（log 行保 sidecar 格式）。
func f0(v float64) string { return strconv.FormatFloat(v, 'f', 0, 64) }
func f1(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }
func f3(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }
