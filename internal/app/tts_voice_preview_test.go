package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// tts_voice_preview_test.go — TTSVoicePreviewBase64（角色档案试听段）。
// 隔离口径与流播测试一致：breezeStreamEnsure/Probe 双桩（真装机跑单测绝不能
// ensure 拉起真实 GPU 服务），合成端点用 httptest 假 breeze 服务。

func stubBreezeStreamReady(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		breezeStreamEnsure = func(a *mediaState) { a.ensureLocalTTSService("breeze") }
		breezeStreamProbe = func(a *mediaState) bool { return a.ttsReady() }
	})
	breezeStreamEnsure = func(*mediaState) {}
	breezeStreamProbe = func(*mediaState) bool { return true }
}

// TestTTSVoicePreviewBase64_Guards 入参守卫：空文本 / 音色描述双空直接报错，
// 不进候选链（也绝不触达任何引擎）。
func TestTTSVoicePreviewBase64_Guards(t *testing.T) {
	a := &mediaState{core: &core{}}
	if _, err := a.TTSVoicePreviewBase64("  ", "v1", "desc"); err == nil || !strings.Contains(err.Error(), "试听文本为空") {
		t.Errorf("空文本应报错, got %v", err)
	}
	if _, err := a.TTSVoicePreviewBase64("你好", " ", " "); err == nil || !strings.Contains(err.Error(), "无可试听") {
		t.Errorf("音色/描述双空应报错, got %v", err)
	}
}

// TestTTSVoicePreviewBase64_NoCapableEngine 无任何可用引擎 → 诚实报错（不落
// Edge/SAPI 假试听；engineMgr=nil 时候选链只剩 breeze 步且必败）。
func TestTTSVoicePreviewBase64_NoCapableEngine(t *testing.T) {
	stubBreezeStreamReady(t)
	a := &mediaState{core: &core{}}
	_, err := a.TTSVoicePreviewBase64("你好", "", "温柔清晰的青年女性声线")
	if err == nil || !strings.Contains(err.Error(), "无支持角色音色预设的语音引擎") {
		t.Fatalf("应诚实报错, got %v", err)
	}
}

// TestVoicePreviewCapable 引擎/模型表达力口径：breeze 全模型可；herdsman 仅
// voicedesign/voxcpm；customvoice/edge/xai 是命名音色，带不进角色预设。
func TestVoicePreviewCapable(t *testing.T) {
	cases := []struct {
		engine, model string
		want          bool
	}{
		{"breeze", "breeze-tts-2", true},
		{"breeze", "", true},
		{"herdsman", "qwen3-tts-voicedesign", true},
		{"herdsman", "voxcpm2", true},
		{"herdsman", "qwen3-tts-customvoice", false},
		{"herdsman", "edge-tts", false},
		{"xai", "grok-tts", false},
		{"", "", false},
	}
	for _, c := range cases {
		if got := voicePreviewCapable(c.engine, c.model); got != c.want {
			t.Errorf("voicePreviewCapable(%q,%q) = %v, want %v", c.engine, c.model, got, c.want)
		}
	}
}

// TestTTSProviderForEngineWithVoice_Disabled 引擎禁用/缺管理器 → 不可构造。
func TestTTSProviderForEngineWithVoice_Disabled(t *testing.T) {
	a := &mediaState{core: &core{}}
	if _, ok := a.ttsProviderForEngineWithVoice("breeze", "m", "v", "d"); ok {
		t.Fatal("engineMgr 为 nil 应不可构造")
	}
	c := &core{engineMgr: modelengine.NewManager("", "")}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{ID: "breeze", Enabled: false}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	b := &mediaState{core: c}
	if _, ok := b.ttsProviderForEngineWithVoice("breeze", "m", "v", "d"); ok {
		t.Fatal("引擎禁用应不可构造")
	}
}

// TestTTSVoicePreviewBase64_BreezeSynth 全链正路：breeze 引擎（httptest 假服务）
// 显式音色/描述合成 → base64+mime 回传；双空描述守卫在链前已拦截。
func TestTTSVoicePreviewBase64_BreezeSynth(t *testing.T) {
	stubBreezeStreamReady(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v1/audio/speech") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("fakeaud"))
	}))
	t.Cleanup(srv.Close)

	c := &core{engineMgr: modelengine.NewManager("", "")}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{ID: "breeze", Enabled: true, BaseURL: srv.URL}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	a := &mediaState{core: &core{}}
	a.core = c

	res, err := a.TTSVoicePreviewBase64("你好，我是林晚。", "clone-voice-1", "用轻快的语气说")
	if err != nil {
		t.Fatalf("合成失败: %v", err)
	}
	if res["base64"] == "" || res["mimeType"] != "audio/wav" {
		t.Errorf("载荷不符: %v", res)
	}
}
