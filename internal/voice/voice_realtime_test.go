// Package voice — S2 Realtime 事件环测试（全离线，fake session）。
//
// 覆盖设计文档 §6 离线清单：
//   - PushAudioChunk realtime 分支（旁路本地 VAD/状态门、16k→24k 重采样直发）
//   - 「未注入 = 走老路」守护测试
//   - 事件泵状态机映射（thinking/speaking/idle）+ 聚合器 done 冲洗 WAV（24k 头）
//   - barge-in 三联（停播放 + response.cancel + buffer.clear）
//   - 输入转写 → EmitVoiceTranscript、转写回复 → EmitVoiceReply
//   - 协议 error → 降级（关会话回拼接管线）；Dial 失败 → 回拼接管线
//   - TurnControl：PTT 释放 → commit；空缓冲不 commit
package voice

import (
	"context"
	"encoding/binary"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/realtime"
)

// ── fake Realtime 会话 ──

// fakeRealtimeSession 内存 fake（实现 RealtimeSession + TurnControl）：
// SendAudio 记录重采样后的载荷，TurnControl 记录协议帧类型，事件由测试经
// Emit 下发。Close 后 Dial 拒绝（与真实会话"已关闭不可重拨"语义一致）。
type fakeRealtimeSession struct {
	mu     sync.Mutex
	sent   [][]byte
	turns  []string
	dialed bool
	closed bool

	events chan realtime.Event
}

func newFakeRealtimeSession() *fakeRealtimeSession {
	return &fakeRealtimeSession{events: make(chan realtime.Event, 64)}
}

func (f *fakeRealtimeSession) Dial(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return context.Canceled // 已关闭 → 拒绝重拨（真实语义）
	}
	f.dialed = true
	return nil
}

func (f *fakeRealtimeSession) SendAudio(pcm []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, append([]byte(nil), pcm...))
	return nil
}

func (f *fakeRealtimeSession) Events() <-chan realtime.Event { return f.events }

func (f *fakeRealtimeSession) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	close(f.events)
	return nil
}

func (f *fakeRealtimeSession) Commit() error         { return f.turn("input_audio_buffer.commit") }
func (f *fakeRealtimeSession) ClearBuffer() error    { return f.turn("input_audio_buffer.clear") }
func (f *fakeRealtimeSession) CreateResponse() error { return f.turn("response.create") }
func (f *fakeRealtimeSession) CancelResponse() error { return f.turn("response.cancel") }

func (f *fakeRealtimeSession) turn(t string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.turns = append(f.turns, t)
	return nil
}

// closeEventsLocked 在持锁状态下关闭事件通道（模拟死会话，测试辅助）。
func (f *fakeRealtimeSession) closeEventsLocked() {
	close(f.events)
}

// Emit 由测试下发服务端事件（Close 后调用会 panic，测试须守序）。
func (f *fakeRealtimeSession) Emit(ev realtime.Event) {
	f.events <- ev
}

func (f *fakeRealtimeSession) sentPayloads() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.sent...)
}

func (f *fakeRealtimeSession) turnFrames() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.turns...)
}

func (f *fakeRealtimeSession) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// activateRealtimeForTest 注入 fake 会话并经 Start 激活（Dial + 事件泵）。
func activateRealtimeForTest(t *testing.T, m *Manager, f *fakeRealtimeSession) {
	t.Helper()
	m.SetRealtimeSession(f)
	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(m.Stop)
}

// realtimeSpeechChunk 构造一段带语音能量的 16k PCM 帧（6400 字节）。
func realtimeSpeechChunk() []byte {
	chunk := make([]byte, ChunkBytes)
	for i := 0; i < len(chunk); i += 2 {
		v := int16(1500)
		chunk[i] = byte(v)
		chunk[i+1] = byte(v >> 8)
	}
	return chunk
}

// ── 推送分支 ──

// TestManager_RealtimePushBypassesVAD realtime 分支：重采样后直发
//（16k 6400 字节 → 24k 9600 字节），本地 VAD 缓冲零累积（旁路）。
func TestManager_RealtimePushBypassesVAD(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	chunk := realtimeSpeechChunk()
	if err := m.PushAudioChunk(chunk); err != nil {
		t.Fatalf("PushAudioChunk: %v", err)
	}

	sent := f.sentPayloads()
	if len(sent) != 1 {
		t.Fatalf("应直发 1 帧, got %d", len(sent))
	}
	if len(sent[0]) != 9600 {
		t.Errorf("重采样后 = %d 字节, want 9600（16k→24k ×3/2）", len(sent[0]))
	}
	want := realtime.Resample16kTo24k(chunk)
	if string(sent[0]) != string(want) {
		t.Error("直发载荷应等于 Resample16kTo24k(chunk)")
	}

	m.mu.Lock()
	bufLen := len(m.vadBuffer)
	m.mu.Unlock()
	if bufLen != 0 {
		t.Errorf("realtime 分支应旁路本地 VAD 缓冲, got %d 字节", bufLen)
	}
	if len(em.errors) != 0 {
		t.Errorf("不应有错误事件: %v", em.errors)
	}
}

// TestManager_RealtimeNotInjectedLegacyGuard 守护测试：未注入会话 = 走老路
//（VAD 缓冲照常累积，无任何 realtime 帧下发）。
func TestManager_RealtimeNotInjectedLegacyGuard(t *testing.T) {
	m, _ := newTestManager()
	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop()

	if m.RealtimeReady() {
		t.Fatal("未注入时 RealtimeReady 应为 false")
	}
	if err := m.PushAudioChunk(realtimeSpeechChunk()); err != nil {
		t.Fatalf("PushAudioChunk: %v", err)
	}
	m.mu.Lock()
	bufLen := len(m.vadBuffer)
	m.mu.Unlock()
	if bufLen != ChunkBytes {
		t.Errorf("未注入应走老路（VAD 累积 %d 字节）, got %d", ChunkBytes, bufLen)
	}
}

// TestManager_RealtimeThinkingStateStillPushes realtime 分支旁路状态门：
// thinking 阶段音频照发（服务端 interrupt_response 负责打断）。
func TestManager_RealtimeThinkingStateStillPushes(t *testing.T) {
	m, _ := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	m.mu.Lock()
	m.state = StateThinking
	m.mu.Unlock()

	if err := m.PushAudioChunk(realtimeSpeechChunk()); err != nil {
		t.Fatalf("PushAudioChunk: %v", err)
	}
	if got := len(f.sentPayloads()); got != 1 {
		t.Errorf("thinking 阶段应照发音频, got %d 帧", got)
	}
}

// ── 事件泵 ──

// TestManager_RealtimeEventPumpMapping 事件泵状态机映射 + 聚合器冲洗：
// created→thinking，delta→speaking，转写→Reply，输入转写→Transcript，
// done→冲洗 24k WAV→idle→自动续听。
func TestManager_RealtimeEventPumpMapping(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	f.Emit(realtime.Event{Type: realtime.EventResponseCreated, DataJSON: `{"type":"response.created"}`})
	waitFor(t, 2*time.Second, func() bool { return m.GetState() == StateThinking })

	audio := []byte{0x01, 0x02, 0x03, 0x04}
	f.Emit(realtime.Event{Type: realtime.EventResponseAudioDelta, DataJSON: `{"type":"response.audio.delta"}`, AudioPCM: audio})
	waitFor(t, 2*time.Second, func() bool { return m.GetState() == StateSpeaking })

	f.Emit(realtime.Event{Type: realtime.EventResponseAudioTranscriptDone, DataJSON: `{"type":"response.audio_transcript.done","transcript":"你好呀"}`})
	f.Emit(realtime.Event{Type: realtime.EventInputAudioTranscriptionCompleted, DataJSON: `{"type":"conversation.item.input_audio_transcription.completed","transcript":"用户问好"}`})
	f.Emit(realtime.Event{Type: realtime.EventResponseDone, DataJSON: `{"type":"response.done","response":{"status":"completed"}}`})

	// 冲洗：一次 EmitVoiceTTSAudio（24k WAV 头）+ 回复/转写文本 + 自动续听
	waitFor(t, 2*time.Second, func() bool {
		em.mu.Lock()
		defer em.mu.Unlock()
		return em.ttsAudioCnt == 1 && len(em.replies) == 1 && len(em.transcripts) == 1
	})
	if s := m.GetState(); s != StateIdle && s != StateListening {
		t.Errorf("done 后应 idle→自动续听, got %s", s)
	}
	em.mu.Lock()
	defer em.mu.Unlock()
	if em.replies[0] != "你好呀" {
		t.Errorf("EmitVoiceReply = %q, want 你好呀", em.replies[0])
	}
	if em.transcripts[0] != "用户问好" {
		t.Errorf("EmitVoiceTranscript = %q, want 用户问好", em.transcripts[0])
	}
	if em.ttsMimeLast != "audio/wav" {
		t.Errorf("mimeType = %q, want audio/wav", em.ttsMimeLast)
	}
	wav := em.ttsAudioLast
	if len(wav) != 44+len(audio) || string(wav[0:4]) != "RIFF" {
		t.Fatalf("冲洗载荷应为 WAV, len=%d", len(wav))
	}
	if sr := binary.LittleEndian.Uint32(wav[24:28]); sr != RealtimeSampleRate {
		t.Errorf("WAV 采样率 = %d, want %d（24kHz 头）", sr, RealtimeSampleRate)
	}
	if !bytesEqual(wav[44:], audio) {
		t.Error("WAV payload 应等于聚合的 audio.delta PCM")
	}
}

// TestManager_RealtimeResponseDoneCancelledDiscardsAudio 被打断的 response
//（status=cancelled）不冲洗半截音频，聚合器复位。
func TestManager_RealtimeResponseDoneCancelledDiscardsAudio(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	f.Emit(realtime.Event{Type: realtime.EventResponseAudioDelta, DataJSON: `{"type":"response.audio.delta"}`, AudioPCM: []byte{0x01, 0x02}})
	f.Emit(realtime.Event{Type: realtime.EventResponseDone, DataJSON: `{"type":"response.done","response":{"status":"cancelled"}}`})

	time.Sleep(150 * time.Millisecond)
	em.mu.Lock()
	defer em.mu.Unlock()
	if em.ttsAudioCnt != 0 {
		t.Errorf("cancelled response 不应冲洗音频, got %d 次", em.ttsAudioCnt)
	}
}

// TestManager_RealtimeCancelledDoneKeepsListening barge-in 后（已 listening）
// 到达的 cancelled response.done 不复位状态（避免续听空档截断用户语音）。
func TestManager_RealtimeCancelledDoneKeepsListening(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	// barge-in：speaking → listening（服务端随后对被打断的 response 发 done）
	m.mu.Lock()
	m.state = StateSpeaking
	m.mu.Unlock()
	f.Emit(realtime.Event{Type: realtime.EventInputAudioBufferSpeechStarted, DataJSON: `{"type":"input_audio_buffer.speech_started"}`})
	waitFor(t, 2*time.Second, func() bool { return m.GetState() == StateListening })

	f.Emit(realtime.Event{Type: realtime.EventResponseDone, DataJSON: `{"type":"response.done","response":{"status":"cancelled"}}`})
	time.Sleep(350 * time.Millisecond) // 越过自动续听窗口

	if s := m.GetState(); s != StateListening {
		t.Errorf("barge-in 后 cancelled done 不应改状态, got %s", s)
	}
	em.mu.Lock()
	defer em.mu.Unlock()
	if em.ttsAudioCnt != 0 {
		t.Errorf("cancelled response 不应冲洗音频, got %d 次", em.ttsAudioCnt)
	}
}

// TestManager_RealtimeBargeInTriple barge-in 三联：speaking 中 speech_started
// → EmitVoiceTTSCancel（停播放）+ response.cancel + input_audio_buffer.clear
// → listening。realtime 真实形态下 speak() 未运行（ttsActive=false），
// 停播放事件必须仍能发出。
func TestManager_RealtimeBargeInTriple(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	// AI 正在说话（realtime 模式：仅状态机进入 speaking，本地 speak() 未运行）
	m.mu.Lock()
	m.state = StateSpeaking
	m.mu.Unlock()

	f.Emit(realtime.Event{Type: realtime.EventInputAudioBufferSpeechStarted, DataJSON: `{"type":"input_audio_buffer.speech_started"}`})

	waitFor(t, 2*time.Second, func() bool { return m.GetState() == StateListening })
	frames := f.turnFrames()
	hasCancel, hasClear := false, false
	for _, fr := range frames {
		switch fr {
		case "response.cancel":
			hasCancel = true
		case "input_audio_buffer.clear":
			hasClear = true
		}
	}
	if !hasCancel || !hasClear {
		t.Errorf("打断三联缺帧: %v（want response.cancel + input_audio_buffer.clear）", frames)
	}
	em.mu.Lock()
	cancelCnt := em.ttsCancelCnt
	em.mu.Unlock()
	if cancelCnt == 0 {
		t.Error("打断应触发 EmitVoiceTTSCancel（停播放，与 ttsActive 无关）")
	}
}

// TestManager_RealtimeBargeInOnlyWhenSpeaking listening/thinking 阶段的
// speech_started 不触发打断（交由服务端 interrupt_response 处理）。
func TestManager_RealtimeBargeInOnlyWhenSpeaking(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	f.Emit(realtime.Event{Type: realtime.EventInputAudioBufferSpeechStarted, DataJSON: `{"type":"input_audio_buffer.speech_started"}`})
	time.Sleep(100 * time.Millisecond)

	if frames := f.turnFrames(); len(frames) != 0 {
		t.Errorf("listening 阶段不应下发打断帧, got %v", frames)
	}
	em.mu.Lock()
	cancelCnt := em.ttsCancelCnt
	em.mu.Unlock()
	if cancelCnt != 0 {
		t.Error("listening 阶段不应触发停播放")
	}
}

// ── 降级护栏 ──

// TestManager_RealtimeProtocolErrorDegrades 协议 error → 关会话降级：
// fake 会话被关闭 + voice:error 事件 + 后续推送走拼接管线（VAD 累积）。
func TestManager_RealtimeProtocolErrorDegrades(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	f.Emit(realtime.Event{Type: realtime.EventError, DataJSON: `{"type":"error","error":{"type":"invalid_request_error","message":"boom"}}`})

	waitFor(t, 2*time.Second, func() bool { return f.isClosed() })
	waitFor(t, 2*time.Second, func() bool {
		em.mu.Lock()
		defer em.mu.Unlock()
		return len(em.errors) == 1
	})

	// 降级后再 Start：fake 拒绝重拨（真实语义）→ 本轮拼接管线接管
	if err := m.Start(); err != nil {
		t.Fatalf("降级后 Start: %v", err)
	}
	if err := m.PushAudioChunk(realtimeSpeechChunk()); err != nil {
		t.Fatalf("PushAudioChunk: %v", err)
	}
	m.mu.Lock()
	bufLen := len(m.vadBuffer)
	m.mu.Unlock()
	if bufLen != ChunkBytes {
		t.Errorf("降级后应走拼接管线（VAD 累积 %d 字节）, got %d", ChunkBytes, bufLen)
	}
}

// TestManager_RealtimeBufferErrorNotFatal 缓冲类协议错误（空 buffer commit）
// 可自愈：仅告警，不关会话不降级。
func TestManager_RealtimeBufferErrorNotFatal(t *testing.T) {
	m, _ := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	f.Emit(realtime.Event{Type: realtime.EventError, DataJSON: `{"type":"error","error":{"type":"invalid_request_error","param":"input_audio_buffer.commit","message":"buffer empty"}}`})

	time.Sleep(150 * time.Millisecond)
	if f.isClosed() {
		t.Error("缓冲类错误不应关闭会话")
	}
	// 会话仍在位：音频照常直发
	if err := m.PushAudioChunk(realtimeSpeechChunk()); err != nil {
		t.Fatalf("PushAudioChunk: %v", err)
	}
	if got := len(f.sentPayloads()); got != 1 {
		t.Errorf("缓冲类错误后音频应照发, got %d 帧", got)
	}
}

// TestManager_RealtimeDialFailureFallsBack Dial 失败 = 本轮回拼接管线：
// Start 不报错、状态机照常、音频走 VAD。
func TestManager_RealtimeDialFailureFallsBack(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	f.mu.Lock()
	f.closed = true // Dial 将拒绝
	f.closeEventsLocked()
	f.mu.Unlock()
	m.SetRealtimeSession(f)

	if err := m.Start(); err != nil {
		t.Fatalf("Dial 失败不应阻断 Start: %v", err)
	}
	defer m.Stop()

	if m.GetState() != StateListening {
		t.Errorf("降级后应照常进入 listening, got %s", m.GetState())
	}
	if err := m.PushAudioChunk(realtimeSpeechChunk()); err != nil {
		t.Fatalf("PushAudioChunk: %v", err)
	}
	m.mu.Lock()
	bufLen := len(m.vadBuffer)
	m.mu.Unlock()
	if bufLen != ChunkBytes {
		t.Errorf("Dial 失败应走拼接管线（VAD 累积）, got %d 字节", bufLen)
	}
	if len(em.errors) != 0 {
		t.Errorf("降级不应产生错误事件: %v", em.errors)
	}
}

// ── TurnControl / PTT ──

// TestManager_RealtimePTTReleaseCommits PTT 释放 → input_audio_buffer.commit
//（有音频流入时），旁路本地 VAD/ASR 识别路径。
func TestManager_RealtimePTTReleaseCommits(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	cfg := DefaultVoiceConfig()
	cfg.VoiceMode = VoiceModePTT
	m.ApplyConfig(cfg)
	activateRealtimeForTest(t, m, f)

	if err := m.PushAudioChunk(realtimeSpeechChunk()); err != nil {
		t.Fatalf("PushAudioChunk: %v", err)
	}
	m.SetPTTActive(false) // 释放

	waitFor(t, 2*time.Second, func() bool { return m.GetState() == StateIdle })
	frames := f.turnFrames()
	if len(frames) != 1 || frames[0] != "input_audio_buffer.commit" {
		t.Errorf("PTT 释放应下发 commit, got %v", frames)
	}
	em.mu.Lock()
	defer em.mu.Unlock()
	if len(em.errors) != 0 {
		t.Errorf("realtime PTT 释放不应走本地 ASR（无错误事件）: %v", em.errors)
	}
}

// TestManager_RealtimePTTReleaseEmptyNoCommit 无音频流入时 PTT 释放不 commit
//（防服务端空缓冲协议错误）。
func TestManager_RealtimePTTReleaseEmptyNoCommit(t *testing.T) {
	m, _ := newTestManager()
	f := newFakeRealtimeSession()
	cfg := DefaultVoiceConfig()
	cfg.VoiceMode = VoiceModePTT
	m.ApplyConfig(cfg)
	activateRealtimeForTest(t, m, f)

	m.SetPTTActive(false)
	if frames := f.turnFrames(); len(frames) != 0 {
		t.Errorf("空缓冲不应 commit, got %v", frames)
	}
}

// TestManager_CancelTTSOverlayRealtime CancelTTS 路径叠加下发：会话在位且
// speaking 中时补发 response.cancel + input_audio_buffer.clear（UI 打断按钮；
// realtime 形态下本地 speak() 未运行，不重复发停播放事件）。
func TestManager_CancelTTSOverlayRealtime(t *testing.T) {
	m, em := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	m.mu.Lock()
	m.state = StateSpeaking
	m.mu.Unlock()

	m.CancelTTS()

	frames := f.turnFrames()
	joined := strings.Join(frames, ",")
	if !strings.Contains(joined, "response.cancel") || !strings.Contains(joined, "input_audio_buffer.clear") {
		t.Errorf("CancelTTS 应叠加 response.cancel + clear, got %v", frames)
	}
	em.mu.Lock()
	cancelCnt := em.ttsCancelCnt
	em.mu.Unlock()
	if cancelCnt != 0 {
		t.Errorf("本地 speak 未运行时不应重复发停播放事件, got %d 次", cancelCnt)
	}
}

// TestManager_CancelTTSOverlayIdleSkipsFrames idle/listening 状态下 CancelTTS
// 不下发 cancel/clear 帧（前端回打 VoiceCancelTTS 的去重守卫）。
func TestManager_CancelTTSOverlayIdleSkipsFrames(t *testing.T) {
	m, _ := newTestManager()
	f := newFakeRealtimeSession()
	activateRealtimeForTest(t, m, f)

	m.CancelTTS() // 状态 listening → 无进行中 response

	if frames := f.turnFrames(); len(frames) != 0 {
		t.Errorf("listening 状态不应下发 cancel/clear 帧, got %v", frames)
	}
}

// ── 工具 ──

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
