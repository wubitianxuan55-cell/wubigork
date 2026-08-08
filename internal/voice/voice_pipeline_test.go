package voice

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/asr"
)

// TestManager_VoiceRoundTrip drives the full pipeline through a fake Herdsman
// ASR server: VAD -> ASR (WAV payload) -> whisper chat -> TTS -> PlaybackDone
// -> back to idle.
func TestManager_VoiceRoundTrip(t *testing.T) {
	var mu sync.Mutex
	var gotAudio string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			http.NotFound(w, r)
			return
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		audio, _ := req["audio"].(string)
		mu.Lock()
		gotAudio = audio
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"text":"你好，世界"}`)
	}))
	defer srv.Close()

	m := NewManager(&mockEmitter{}, DefaultVoiceConfig())
	m.SetASRClient(asr.NewHerdsmanASR(srv.URL, "whisper-base"))
	m.SetWhisperChatFn(func(userMsg, personalityID string) (string, string, error) {
		return "收到", "CALM_RATIONAL", nil
	})

	var ttsCalls int32
	m.SetTTSSynthesizeFn(func(text, voiceDesc string) ([]byte, string, error) {
		atomic.AddInt32(&ttsCalls, 1)
		return []byte("fake-audio-bytes"), "audio/wav", nil
	})

	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop()

	// Push two speech-energy chunks to confirm the VAD (noise-spike filter).
	speech := make([]byte, ChunkBytes)
	for i := 0; i < len(speech); i += 2 {
		v := int16(1500)
		speech[i] = byte(v)
		speech[i+1] = byte(v >> 8)
	}
	for i := 0; i < 2; i++ {
		if err := m.PushAudioChunk(speech); err != nil {
			t.Fatalf("PushAudioChunk(speech %d): %v", i, err)
		}
	}

	// Push enough silence (5 x 200ms >= SilenceThresholdMs) to end the turn.
	for i := 0; i < 6; i++ {
		if err := m.PushAudioChunk(make([]byte, ChunkBytes)); err != nil {
			t.Fatalf("PushAudioChunk(silence %d): %v", i, err)
		}
	}

	// Wait until the pipeline reaches speaking (ASR + chat + TTS done).
	waitFor(t, 5*time.Second, func() bool {
		return m.GetState() == StateSpeaking && atomic.LoadInt32(&ttsCalls) == 1
	})

	// The ASR payload must be a real WAV (RIFF header), not raw PCM.
	mu.Lock()
	audio := gotAudio
	mu.Unlock()
	const prefix = "data:audio/wav;base64,"
	if !strings.HasPrefix(audio, prefix) {
		t.Fatalf("ASR audio prefix = %q, want %q", audio, prefix)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(audio, prefix))
	if err != nil {
		t.Fatalf("decode ASR audio: %v", err)
	}
	if len(raw) < 44 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		t.Errorf("ASR payload is not a WAV file (first bytes %q)", raw[:min(44, len(raw))])
	}

	// Playback finishes -> state leaves speaking and the pipeline re-arms.
	m.PlaybackDone()
	waitFor(t, 5*time.Second, func() bool {
		s := m.GetState()
		return s == StateIdle || s == StateListening
	})
	if got := m.GetState(); got == StateSpeaking {
		t.Errorf("PlaybackDone 后仍处于 speaking: %s", got)
	}
}

// TestManager_HandleUserText verifies the browser-recognition entry point:
// recognized text goes straight into the conversation pipeline, skipping
// backend ASR.
func TestManager_HandleUserText(t *testing.T) {
	m := NewManager(&mockEmitter{}, DefaultVoiceConfig())
	m.SetWhisperChatFn(func(userMsg, personalityID string) (string, string, error) {
		return "收到：" + userMsg, "CALM_RATIONAL", nil
	})
	var ttsCalls int32
	m.SetTTSSynthesizeFn(func(text, voiceDesc string) ([]byte, string, error) {
		atomic.AddInt32(&ttsCalls, 1)
		return []byte("fake-audio"), "audio/wav", nil
	})

	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop()

	// Empty text must be ignored and leave the state machine untouched.
	m.HandleUserText("   ")
	if got := m.GetState(); got != StateListening {
		t.Fatalf("空文本后状态应为 listening, got %s", got)
	}

	m.HandleUserText("你好")
	waitFor(t, 5*time.Second, func() bool {
		return m.GetState() == StateSpeaking && atomic.LoadInt32(&ttsCalls) == 1
	})

	m.PlaybackDone()
	waitFor(t, 5*time.Second, func() bool {
		s := m.GetState()
		return s == StateIdle || s == StateListening
	})
	if got := m.GetState(); got == StateSpeaking {
		t.Errorf("PlaybackDone 后仍处于 speaking: %s", got)
	}
}

// TestManager_StreamingTTS_SentenceBySentence verifies that a multi-sentence
// reply is synthesized sentence-by-sentence and played back one sentence per
// PlaybackDone, with the next sentence pre-synthesized while the current one
// is playing (Hermes-style streaming).
func TestManager_StreamingTTS_SentenceBySentence(t *testing.T) {
	em := &mockEmitter{}
	m := NewManager(em, DefaultVoiceConfig())
	m.SetWhisperChatFn(func(userMsg, personalityID string) (string, string, error) {
		return "第一句。第二句。第三句。", "CALM_RATIONAL", nil
	})
	var ttsCalls int32
	m.SetTTSSynthesizeFn(func(text, voiceDesc string) ([]byte, string, error) {
		atomic.AddInt32(&ttsCalls, 1)
		return []byte("fake-" + text), "audio/wav", nil
	})

	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop()

	m.HandleUserText("你好")

	// 第一句合成并开始播放
	waitFor(t, 5*time.Second, func() bool {
		return m.GetState() == StateSpeaking && atomic.LoadInt32(&ttsCalls) >= 1
	})
	// 播放期间预合成下一句
	waitFor(t, 5*time.Second, func() bool {
		return atomic.LoadInt32(&ttsCalls) >= 2
	})

	// 句1 播完 → 后端取句2 并发出音频
	m.PlaybackDone()
	waitFor(t, 5*time.Second, func() bool {
		em.mu.Lock()
		defer em.mu.Unlock()
		return em.ttsAudioCnt >= 2
	})

	// 句2 播完 → 发出句3
	m.PlaybackDone()
	waitFor(t, 5*time.Second, func() bool {
		em.mu.Lock()
		defer em.mu.Unlock()
		return em.ttsAudioCnt >= 3
	})

	// 三句都已合成
	if got := atomic.LoadInt32(&ttsCalls); got != 3 {
		t.Errorf("应合成 3 句, got %d", got)
	}

	// 句3 播完 → speak 退出 → idle/listening
	m.PlaybackDone()
	waitFor(t, 5*time.Second, func() bool {
		s := m.GetState()
		return s == StateIdle || s == StateListening
	})
}

// TestManager_BargeInStopsStreaming verifies the full barge-in chain: while
// the AI is speaking, user audio exceeding the interrupt threshold cancels
// the ongoing TTS, no further sentences are played, and the pipeline returns
// to listening.
func TestManager_BargeInStopsStreaming(t *testing.T) {
	em := &mockEmitter{}
	m := NewManager(em, DefaultVoiceConfig())
	m.SetWhisperChatFn(func(userMsg, personalityID string) (string, string, error) {
		return "第一句。第二句。第三句。", "CALM_RATIONAL", nil
	})
	var ttsCalls int32
	m.SetTTSSynthesizeFn(func(text, voiceDesc string) ([]byte, string, error) {
		atomic.AddInt32(&ttsCalls, 1)
		return []byte("fake"), "audio/wav", nil
	})

	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop()

	m.HandleUserText("你好")

	// 等第一句开始播放、第二句预合成完成
	waitFor(t, 5*time.Second, func() bool {
		return m.GetState() == StateSpeaking && atomic.LoadInt32(&ttsCalls) >= 2
	})
	em.mu.Lock()
	playedBefore := em.ttsAudioCnt
	em.mu.Unlock()

	// 用户插话：3 块语音（200ms×3 ≥ 500ms 打断阈值）
	chunk := make([]byte, ChunkBytes)
	for i := 0; i < len(chunk); i += 2 {
		v := int16(1500)
		chunk[i] = byte(v)
		chunk[i+1] = byte(v >> 8)
	}
	for i := 0; i < 3; i++ {
		if err := m.PushAudioChunk(chunk); err != nil {
			t.Fatalf("PushAudioChunk %d: %v", i, err)
		}
	}

	// speak 应退出，回到 idle / 自动恢复监听
	waitFor(t, 5*time.Second, func() bool {
		s := m.GetState()
		return s == StateIdle || s == StateListening
	})
	// 打断后不应再播放新句子
	time.Sleep(150 * time.Millisecond)
	em.mu.Lock()
	playedAfter := em.ttsAudioCnt
	cancelCnt := em.ttsCancelCnt
	em.mu.Unlock()
	if playedAfter > playedBefore {
		t.Errorf("打断后不应再播放新句子: before=%d after=%d", playedBefore, playedAfter)
	}
	if cancelCnt == 0 {
		t.Error("打断应触发 voice:tts-speak-cancel 事件")
	}
}

// TestManager_InterruptDuringGenerationSkipsPlayback verifies the full-duplex
// barge-in path: while the LLM is still generating (thinking), a new user
// input cancels the in-flight turn so its reply is never played; the queued
// new input then speaks normally.
func TestManager_InterruptDuringGenerationSkipsPlayback(t *testing.T) {
	em := &mockEmitter{}
	m := NewManager(em, DefaultVoiceConfig())

	var chatCalls int32
	releaseFirst := make(chan struct{})
	m.SetWhisperChatFn(func(userMsg, personalityID string) (string, string, error) {
		if atomic.AddInt32(&chatCalls, 1) == 1 {
			<-releaseFirst // 模拟 LLM 仍在生成
			return "旧回复。", "CALM_RATIONAL", nil
		}
		return "新回复。", "CALM_RATIONAL", nil
	})
	var ttsCalls int32
	m.SetTTSSynthesizeFn(func(text, voiceDesc string) ([]byte, string, error) {
		atomic.AddInt32(&ttsCalls, 1)
		return []byte("fake"), "audio/wav", nil
	})

	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop()

	// 第一轮：LLM 生成中（thinking）
	m.HandleUserText("问题一")
	waitFor(t, 5*time.Second, func() bool {
		return m.GetState() == StateThinking && atomic.LoadInt32(&chatCalls) == 1
	})

	// 用户插话：新一轮输入，打断生成中的轮次
	m.HandleUserText("问题二")
	close(releaseFirst)

	// 旧回复不应播放；新回复正常进入 speaking
	waitFor(t, 5*time.Second, func() bool {
		return m.GetState() == StateSpeaking && atomic.LoadInt32(&chatCalls) >= 2
	})
	em.mu.Lock()
	audioCnt := em.ttsAudioCnt
	em.mu.Unlock()
	if audioCnt != 1 {
		t.Errorf("应只播放新回复 1 句（旧回复被跳过）, 实际播放 %d 句", audioCnt)
	}
	// 清理：播放完成
	m.PlaybackDone()
	waitFor(t, 5*time.Second, func() bool {
		s := m.GetState()
		return s == StateIdle || s == StateListening
	})
}

// TestManager_InterruptNotifiesModel verifies the interrupted latch: after a
// barge-in, the next conversation turn tells the model the previous spoken
// reply was cut short (Hermes SPEECH_INTERRUPTED_NOTE equivalent).
func TestManager_InterruptNotifiesModel(t *testing.T) {
	m := NewManager(&mockEmitter{}, DefaultVoiceConfig())

	var mu sync.Mutex
	var gotMsgs []string
	m.SetWhisperChatFn(func(userMsg, personalityID string) (string, string, error) {
		mu.Lock()
		gotMsgs = append(gotMsgs, userMsg)
		mu.Unlock()
		return "收到。", "CALM_RATIONAL", nil
	})
	m.SetTTSSynthesizeFn(func(text, voiceDesc string) ([]byte, string, error) {
		return []byte("fake"), "audio/wav", nil
	})

	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop()

	// 第一轮正常对话并播完
	m.HandleUserText("你好")
	waitFor(t, 5*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(gotMsgs) == 1
	})
	m.PlaybackDone() // 第一轮播完
	waitFor(t, 5*time.Second, func() bool {
		s := m.GetState()
		return s == StateIdle || s == StateListening
	})

	// 第二轮开始播放后，用户语音打断（barge-in）
	m.HandleUserText("继续")
	waitFor(t, 5*time.Second, func() bool {
		return m.GetState() == StateSpeaking
	})
	chunk := make([]byte, ChunkBytes)
	for i := 0; i < len(chunk); i += 2 {
		v := int16(1500)
		chunk[i] = byte(v)
		chunk[i+1] = byte(v >> 8)
	}
	for i := 0; i < 3; i++ {
		_ = m.PushAudioChunk(chunk)
	}
	waitFor(t, 5*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(gotMsgs) == 2
	})
	waitFor(t, 5*time.Second, func() bool {
		s := m.GetState()
		return s == StateIdle || s == StateListening
	})

	// 第三轮：模型应收到上一轮被打断的提示
	m.HandleUserText("再问一个")
	waitFor(t, 5*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(gotMsgs) == 3
	})
	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(gotMsgs[2], "打断") {
		t.Errorf("第三轮输入应携带打断提示, got %q", gotMsgs[2])
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
