package app

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	appconfig "github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/secure"
	"github.com/gaea/gaea/internal/realtime"
)

// S2 Realtime 注入与就绪门测试（initVoice 构造注入 + VoiceStart realtime 门）：
// provider 配置 → 真实 session 注入 Manager；构造失败/未配置 → 不注入、拼接
// 管线零变化；realtime 会话在位时 VoiceStart 不再强制 ASRReady。

// stubRealtimeSession app 层测试桩（实现 RealtimeSession + TurnControl）。
type stubRealtimeSession struct {
	events chan realtime.Event
	closed int32
}

func newStubRealtimeSession() *stubRealtimeSession {
	return &stubRealtimeSession{events: make(chan realtime.Event, 8)}
}

func (s *stubRealtimeSession) Dial(context.Context) error    { return nil }
func (s *stubRealtimeSession) SendAudio([]byte) error        { return nil }
func (s *stubRealtimeSession) Events() <-chan realtime.Event { return s.events }
func (s *stubRealtimeSession) Close() error {
	if atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		close(s.events)
	}
	return nil
}
func (s *stubRealtimeSession) Commit() error         { return nil }
func (s *stubRealtimeSession) ClearBuffer() error    { return nil }
func (s *stubRealtimeSession) CreateResponse() error { return nil }
func (s *stubRealtimeSession) CancelResponse() error { return nil }

// TestInitVoice_InjectsRealtimeSession provider 配置（合法 DPAPI 密文 Key）→
// initVoice 构造真实会话注入 Manager（RealtimeReady=true）。
func TestInitVoice_InjectsRealtimeSession(t *testing.T) {
	enc, err := secure.EncryptString("sk-realtime-inject-sample")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	cfg := &appconfig.Config{
		RealtimeProvider: "openai",
		RealtimeModel:    "gpt-4o-realtime-preview",
		RealtimeAPIKey:   enc,
	}
	a := &mediaState{core: &core{cfg: cfg}}
	a.initVoice()

	if !a.voiceManager.RealtimeReady() {
		t.Fatal("provider 配置时应注入 realtime 会话（RealtimeReady=true）")
	}
	if health := a.VoiceHealth(); health["realtimeReady"] != true {
		t.Errorf("realtimeReady = %v, want true", health["realtimeReady"])
	}
}

// TestInitVoice_RealtimeConstructFailureNotInjected 构造失败（解密失败 → Key
// 置空 → 注册表构造因缺 Key 报错）→ 不注入、不崩、拼接管线零变化。
func TestInitVoice_RealtimeConstructFailureNotInjected(t *testing.T) {
	cfg := &appconfig.Config{
		RealtimeProvider: "openai",
		RealtimeModel:    "gpt-4o-realtime-preview",
		RealtimeAPIKey:   "dpapi:bm90LWEtcmVhbC1ibG9i", // 非合法密文
	}
	a := &mediaState{core: &core{cfg: cfg}}
	a.initVoice() // 不崩即通过（内部 slog.Warn）

	if a.voiceManager.RealtimeReady() {
		t.Fatal("构造失败时不应注入（RealtimeReady=false）")
	}
	if health := a.VoiceHealth(); health["realtimeReady"] != false {
		t.Errorf("realtimeReady = %v, want false（构造失败降级）", health["realtimeReady"])
	}
}

// TestInitVoice_RealtimeUnconfiguredNotInjected 守护测试：未配置（Provider 空）
// → 不注入（RealtimeReady=false），现拼接管线零变化。
func TestInitVoice_RealtimeUnconfiguredNotInjected(t *testing.T) {
	a := &mediaState{core: &core{cfg: &appconfig.Config{}}}
	a.initVoice()

	if a.voiceManager.RealtimeReady() {
		t.Fatal("未配置时不应注入（未注入 = 走老路）")
	}
}

// TestVoiceStart_RealtimeGateSkipsASR realtime 会话在位时，VoiceStart(false)
// 以 realtimeReady 为门：无本地 ASR 提供者也能启动（不再强制 ASRReady）。
func TestVoiceStart_RealtimeGateSkipsASR(t *testing.T) {
	cfg := &appconfig.Config{}
	app := &App{core: &core{cfg: cfg}} // 事件发射经 core.emit（无 Wails ctx 时跳过）
	a := &mediaState{core: &core{cfg: cfg}, app: app}
	a.initVoice() // engineMgr=nil → 无 ASR 提供者
	a.voiceManager.SetRealtimeSession(newStubRealtimeSession())

	if err := a.VoiceStart(false); err != nil {
		t.Fatalf("realtime 在位时 VoiceStart(false) 不应要求 ASRReady: %v", err)
	}
	defer a.voiceManager.Stop() // 收尾：关会话 → 事件泵退出
}

// TestVoiceStart_LegacyASRGateUnchanged 会话不在位时维持原 ASRReady 门：
// 无 ASR 提供者且未注入 realtime → 报错（拼接管线行为不变）。
func TestVoiceStart_LegacyASRGateUnchanged(t *testing.T) {
	cfg := &appconfig.Config{}
	app := &App{core: &core{cfg: cfg}}
	a := &mediaState{core: &core{cfg: cfg}, app: app}
	a.initVoice()

	err := a.VoiceStart(false)
	if err == nil {
		t.Fatal("无 ASR 且未注入 realtime 时 VoiceStart(false) 应报错（原门保留）")
	}
	if want := "语音识别未就绪"; !strings.Contains(err.Error(), want) {
		t.Errorf("错误信息 = %q, want 含 %q", err.Error(), want)
	}
}
