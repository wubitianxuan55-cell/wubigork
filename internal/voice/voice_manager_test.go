// Package voice — 语音管道管理器状态机测试
package voice

import (
	"sync"
	"testing"
)

// mockEmitter 记录事件调用
type mockEmitter struct {
	mu         sync.Mutex
	states     []VoiceState
	listening  []bool
	thinking   []bool
	errors     []error
	listenCalls int
}

func (e *mockEmitter) EmitVoiceState(s VoiceState) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.states = append(e.states, s)
}
func (e *mockEmitter) EmitVoiceTranscript(text string, isFinal bool) {}
func (e *mockEmitter) EmitVoiceReply(text string)                     {}
func (e *mockEmitter) EmitVoiceTTSAudio(audio []byte, mimeType string) {}
func (e *mockEmitter) EmitVoiceTTSSpeakText(text string)              {}
func (e *mockEmitter) EmitVoiceTTSCancel()                            {}
func (e *mockEmitter) EmitVoiceListening(active bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.listening = append(e.listening, active)
	e.listenCalls++
}
func (e *mockEmitter) EmitVoiceThinking(active bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.thinking = append(e.thinking, active)
}
func (e *mockEmitter) EmitVoiceError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errors = append(e.errors, err)
}

func newTestManager() (*Manager, *mockEmitter) {
	em := &mockEmitter{}
	m := NewManager(em, DefaultVoiceConfig())
	return m, em
}

func TestManager_InitialState(t *testing.T) {
	m, _ := newTestManager()
	if got := m.GetState(); got != StateIdle {
		t.Errorf("初始状态应为 idle, got %q", got)
	}
}

func TestManager_StartTransitionsToListening(t *testing.T) {
	m, em := newTestManager()
	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if got := m.GetState(); got != StateListening {
		t.Errorf("Start 后应为 listening, got %q", got)
	}
	em.mu.Lock()
	defer em.mu.Unlock()
	if len(em.states) != 1 || em.states[0] != StateListening {
		t.Errorf("应 emit listening 状态, got %v", em.states)
	}
	if len(em.listening) != 1 || !em.listening[0] {
		t.Errorf("应 emit listening(true), got %v", em.listening)
	}
}

func TestManager_StartTwiceReturnsError(t *testing.T) {
	m, _ := newTestManager()
	_ = m.Start()
	if err := m.Start(); err == nil {
		t.Error("重复 Start 应返回错误")
	}
}

func TestManager_StopReturnsToIdle(t *testing.T) {
	m, em := newTestManager()
	_ = m.Start()
	m.Stop()
	if got := m.GetState(); got != StateIdle {
		t.Errorf("Stop 后应为 idle, got %q", got)
	}
	em.mu.Lock()
	defer em.mu.Unlock()
	if len(em.listening) < 2 || em.listening[len(em.listening)-1] {
		t.Errorf("Stop 应 emit listening(false), got %v", em.listening)
	}
}

func TestManager_ApplyAndGetConfig(t *testing.T) {
	m, _ := newTestManager()
	c := DefaultVoiceConfig()
	c.TTSEngine = TTSEngineHerdsman
	c.VoiceMode = VoiceModePTT
	m.ApplyConfig(c)
	got := m.GetConfig()
	if got.TTSEngine != TTSEngineHerdsman || got.VoiceMode != VoiceModePTT {
		t.Errorf("配置未生效: %+v", got)
	}
}

func TestManager_PushAudioChunk_IdleIgnored(t *testing.T) {
	m, em := newTestManager()
	// idle 状态推音频应被忽略（不 panic）
	if err := m.PushAudioChunk(make([]byte, 6400)); err != nil {
		t.Fatalf("idle 推音频: %v", err)
	}
	em.mu.Lock()
	defer em.mu.Unlock()
	if len(em.states) != 0 {
		t.Errorf("idle 推音频不应触发状态变化, got %v", em.states)
	}
}

func TestManager_PushAudioChunk_ListeningAccumulatesVAD(t *testing.T) {
	m, em := newTestManager()
	_ = m.Start()

	// 推入语音能量块（int16 1000 振幅 > 400 阈值）→ 检测到语音
	chunk := make([]byte, 6400)
	for i := 0; i < len(chunk); i += 2 {
		v := int16(1000)
		chunk[i] = byte(v)
		chunk[i+1] = byte(v >> 8)
	}
	if err := m.PushAudioChunk(chunk); err != nil {
		t.Fatalf("PushAudioChunk: %v", err)
	}
	m.mu.Lock()
	detected := m.vadSpeechDetected
	bufLen := len(m.vadBuffer)
	m.mu.Unlock()
	if !detected {
		t.Error("语音块应触发 vadSpeechDetected")
	}
	if bufLen != 6400 {
		t.Errorf("VAD 缓冲应累积 6400 字节, got %d", bufLen)
	}
	em.mu.Lock()
	defer em.mu.Unlock()
	if len(em.listening) != 2 || !em.listening[1] {
		t.Errorf("检测到语音应再次 emit listening(true), got %v", em.listening)
	}
}

func TestManager_PushAudioChunk_SilenceDiscardedBeforeSpeech(t *testing.T) {
	m, _ := newTestManager()
	_ = m.Start()
	// 静音块（全零）在未检测到语音前应被丢弃，不累积
	if err := m.PushAudioChunk(make([]byte, 6400)); err != nil {
		t.Fatalf("PushAudioChunk: %v", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.vadSpeechDetected {
		t.Error("静音不应触发语音检测")
	}
	if len(m.vadBuffer) != 0 {
		t.Errorf("语音前的静音应丢弃, bufLen=%d", len(m.vadBuffer))
	}
}

func TestManager_PushAudioChunk_SpeakingInterruptAccumulates(t *testing.T) {
	m, _ := newTestManager()
	// 手动置 speaking 状态
	m.mu.Lock()
	m.state = StateSpeaking
	m.config.InterruptThresholdMs = 500
	m.mu.Unlock()

	// 连续 3 块语音（200ms×3=600ms > 500ms 阈值）→ 触发 interruptCh
	chunk := make([]byte, 6400)
	for i := 0; i < len(chunk); i += 2 {
		v := int16(1000)
		chunk[i] = byte(v)
		chunk[i+1] = byte(v >> 8)
	}
	for i := 0; i < 3; i++ {
		if err := m.PushAudioChunk(chunk); err != nil {
			t.Fatalf("PushAudioChunk %d: %v", i, err)
		}
	}
	select {
	case <-m.InterruptCh():
		// 触发成功
	default:
		t.Error("累积超过打断阈值应触发打断信号")
	}
}

func TestManager_SpeakingSilenceResetsInterrupt(t *testing.T) {
	m, _ := newTestManager()
	m.mu.Lock()
	m.state = StateSpeaking
	m.config.InterruptThresholdMs = 500
	m.mu.Unlock()

	// 1 块语音 + 1 块静音 → 累积被重置
	voice := make([]byte, 6400)
	for i := 0; i < len(voice); i += 2 {
		v := int16(1000)
		voice[i] = byte(v)
		voice[i+1] = byte(v >> 8)
	}
	_ = m.PushAudioChunk(voice) // +200ms
	_ = m.PushAudioChunk(make([]byte, 6400)) // 静音重置
	_ = m.PushAudioChunk(voice) // +200ms（重置后重新累积）

	m.mu.Lock()
	acc := m.interruptSpeechMs
	m.mu.Unlock()
	if acc != 200 {
		t.Errorf("静音应重置打断累积, 现累积 %dms (want 200)", acc)
	}
}

func TestManager_HealthCheck(t *testing.T) {
	m, _ := newTestManager()
	hc := m.HealthCheck()
	if hc == nil {
		t.Fatal("HealthCheck 不应返回 nil")
	}
	if _, ok := hc["state"]; !ok {
		t.Error("HealthCheck 应含 state")
	}
}
