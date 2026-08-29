// Package tts — TTS 提供者 Seam 测试（Step 3c）。
//
// 固化 seam 行为：
//   - 注册表：edge/herdsman/xai/sapi 自注册、互斥注册 panic、未知 kind fail-closed
//   - TTSChain：首个成功即赢 / 失败回退 / 全败报错（与历史 SynthesizerChain 同语义）
//   - herdsman 工厂：customvoice / voicedesign / voiceclone / voxcpm 构造路由
//     （收敛 app 层 tryEngineTTS 的按模型分支）
package tts

import (
	"testing"
)

// ─── 注册表 ─────────────────────────────────────────────────

func TestTTSProviderRegistry_Kinds(t *testing.T) {
	kinds := TTSProviderKinds()
	got := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		got[k] = true
	}
	for _, want := range []string{"edge", "herdsman", "xai", "sapi"} {
		if !got[want] {
			t.Errorf("注册表缺少 kind %q（已注册: %v）", want, kinds)
		}
	}
}

func TestTTSProviderRegistry_Construct(t *testing.T) {
	// edge / sapi：无需配置
	for _, kind := range []string{"edge", "sapi"} {
		p, err := NewTTSProvider(kind, TTSConfig{})
		if err != nil {
			t.Fatalf("NewTTSProvider(%q): %v", kind, err)
		}
		if p == nil || p.Name() != kind {
			t.Errorf("%q 提供者 Name = %v, want %q", kind, p, kind)
		}
	}

	// herdsman：BaseURL + Model
	h, err := NewTTSProvider("herdsman", TTSConfig{BaseURL: "http://localhost:8080/v1", Model: "qwen3-tts-customvoice", Voice: "serena"})
	if err != nil {
		t.Fatalf("NewTTSProvider(herdsman): %v", err)
	}
	if h.Name() != "herdsman" {
		t.Errorf("herdsman Name = %q", h.Name())
	}

	// xai：缺少 GetToken → fail-closed
	if _, err := NewTTSProvider("xai", TTSConfig{BaseURL: "http://x", Voice: "eve"}); err == nil {
		t.Fatal("xai 缺少 GetToken 应报错（fail-closed）")
	}
	x, err := NewTTSProvider("xai", TTSConfig{BaseURL: "http://x", Voice: "eve", GetToken: func() (string, error) { return "tok", nil }})
	if err != nil {
		t.Fatalf("NewTTSProvider(xai): %v", err)
	}
	if x.Name() != "xai" {
		t.Errorf("xai Name = %q", x.Name())
	}
}

func TestTTSProviderRegistry_UnknownKindFailsClosed(t *testing.T) {
	if _, err := NewTTSProvider("no-such-engine", TTSConfig{}); err == nil {
		t.Fatal("未知 kind 应报错（fail-closed，不静默降级）")
	}
}

func TestTTSProviderRegistry_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic（互斥注册纪律）")
		}
	}()
	RegisterTTSProvider("edge", func(cfg TTSConfig) (TTSProvider, error) { return nil, nil }) // 已注册 → panic
}

// ─── TTSChain（回退链，与 SynthesizerChain 同语义） ─────────

type fakeTTSProvider struct {
	name string
	out  []byte
	mime string
	err  error
}

func (f *fakeTTSProvider) Name() string { return f.name }
func (f *fakeTTSProvider) Synthesize(text string) ([]byte, error) {
	return f.out, f.err
}
func (f *fakeTTSProvider) SynthesizeWithMime(text string) ([]byte, string, error) {
	return f.out, f.mime, f.err
}
func (f *fakeTTSProvider) SynthesizeWithParams(text string, p TTSParams) ([]byte, string, error) {
	return f.out, f.mime, f.err
}

func TestTTSChain_FirstSuccessWins(t *testing.T) {
	chain := NewTTSChain(
		&fakeTTSProvider{name: "herdsman", out: []byte("first"), mime: "audio/wav"},
		&fakeTTSProvider{name: "edge", out: []byte("second"), mime: "audio/mp3"},
	)
	audio, mime, name, err := chain.Synthesize("测试")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(audio) != "first" || mime != "audio/wav" || name != "herdsman" {
		t.Errorf("应取第一个成功提供者: %q/%q/%q", audio, mime, name)
	}
}

func TestTTSChain_FallbackOnFailure(t *testing.T) {
	chain := NewTTSChain(
		&fakeTTSProvider{name: "a", err: errEngineDown},
		&fakeTTSProvider{name: "b", out: []byte("backup"), mime: "audio/mpeg"},
	)
	audio, _, name, err := chain.Synthesize("测试")
	if err != nil {
		t.Fatalf("第二个成功不应报错: %v", err)
	}
	if string(audio) != "backup" || name != "b" {
		t.Errorf("应回退到第二个提供者: %q/%q", audio, name)
	}
}

func TestTTSChain_AllFail(t *testing.T) {
	chain := NewTTSChain(
		&fakeTTSProvider{name: "a", err: errEngineDown},
		&fakeTTSProvider{name: "b", err: errEngineDown},
	)
	if _, _, _, err := chain.Synthesize("测试"); err == nil {
		t.Fatal("全部失败应返回错误")
	}
}

func TestTTSChain_NoProviders(t *testing.T) {
	if _, _, _, err := NewTTSChain().Synthesize("测试"); err == nil {
		t.Fatal("无提供者应返回错误")
	}
}

func TestTTSChain_EmptyAudioIsFailure(t *testing.T) {
	// 空音频视为失败（继续尝试下一个），与历史「len(audio)>0 才算成功」一致
	chain := NewTTSChain(
		&fakeTTSProvider{name: "a", out: nil, mime: "audio/mp3"},
		&fakeTTSProvider{name: "b", out: []byte("ok"), mime: "audio/wav"},
	)
	audio, _, name, err := chain.Synthesize("测试")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(audio) != "ok" || name != "b" {
		t.Errorf("空音频应视为失败并回退: %q/%q", audio, name)
	}
}

func TestTTSChain_ProviderByName(t *testing.T) {
	edge := &fakeTTSProvider{name: "edge"}
	chain := NewTTSChain(&fakeTTSProvider{name: "herdsman"}, edge)
	if got := chain.ProviderByName("edge"); got != edge {
		t.Errorf("ProviderByName(edge) 应返回同一实例")
	}
	if got := chain.ProviderByName("nope"); got != nil {
		t.Errorf("ProviderByName(未知) 应返回 nil")
	}
}

// ─── herdsman 工厂构造路由（收敛 tryEngineTTS 分支） ────────

// TestHerdsmanFactory_CustomVoice 默认分支：customvoice 携带音色。
func TestHerdsmanFactory_CustomVoice(t *testing.T) {
	p, err := NewTTSProvider("herdsman", TTSConfig{BaseURL: "http://x/v1", Model: "qwen3-tts-customvoice", Voice: "serena"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	h, ok := p.(*HerdsmanTTS)
	if !ok {
		t.Fatalf("类型应为 *HerdsmanTTS, got %T", p)
	}
	if h.voice != "serena" {
		t.Errorf("customvoice 应设置 voice=serena, got %q", h.voice)
	}
	body := h.buildBody("文本", h.resolveVoice())
	if body["voice"] != "serena" {
		t.Errorf("customvoice 应携带 voice=serena: %+v", body)
	}
	if _, has := body["voice_description"]; has {
		t.Errorf("customvoice 不应携带 voice_description: %+v", body)
	}
}

// TestHerdsmanFactory_VoiceDesign 有描述 → WithDesc（voice_description 进请求体）。
func TestHerdsmanFactory_VoiceDesign(t *testing.T) {
	p, err := NewTTSProvider("herdsman", TTSConfig{BaseURL: "http://x/v1", Model: "qwen3-tts-voicedesign", VoiceDescription: "用温柔的语气说"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	h, ok := p.(*HerdsmanTTS)
	if !ok {
		t.Fatalf("类型应为 *HerdsmanTTS, got %T", p)
	}
	body := h.buildBody("文本", "")
	if body["voice_description"] != "用温柔的语气说" {
		t.Errorf("voicedesign 应携带 voice_description: %+v", body)
	}
	if _, has := body["voice"]; has {
		t.Errorf("voicedesign 不应携带 voice: %+v", body)
	}
}

// TestHerdsmanFactory_VoiceDesignNoDesc voxcpm/voicedesign 无描述 → 空音色构造
// （与历史 tryEngineTTS 的 NewHerdsmanTTS(model, "") 一致）。
func TestHerdsmanFactory_VoiceDesignNoDesc(t *testing.T) {
	p, err := NewTTSProvider("herdsman", TTSConfig{BaseURL: "http://x/v1", Model: "voxcpm2"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	h, ok := p.(*HerdsmanTTS)
	if !ok {
		t.Fatalf("类型应为 *HerdsmanTTS, got %T", p)
	}
	body := h.buildBody("文本", "")
	if _, has := body["voice_description"]; has {
		t.Errorf("无描述不应携带 voice_description: %+v", body)
	}
	if _, has := body["voice"]; has {
		t.Errorf("voxcpm 无描述不应携带 voice: %+v", body)
	}
}

// TestHerdsmanFactory_VoiceClone voiceclone → WithClone（ref_audio/ref_text 进请求体）。
func TestHerdsmanFactory_VoiceClone(t *testing.T) {
	p, err := NewTTSProvider("herdsman", TTSConfig{
		BaseURL: "http://x/v1", Model: "qwen3-tts-voiceclone",
		RefAudio: "data:audio/wav;base64,AAAA", RefText: "参考文本",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	h, ok := p.(*HerdsmanTTS)
	if !ok {
		t.Fatalf("类型应为 *HerdsmanTTS, got %T", p)
	}
	body := h.buildBody("文本", "")
	if body["ref_audio"] != "data:audio/wav;base64,AAAA" || body["ref_text"] != "参考文本" {
		t.Errorf("voiceclone 应携带 ref_audio/ref_text: %+v", body)
	}
	if _, has := body["voice"]; has {
		t.Errorf("voiceclone 不应携带 voice: %+v", body)
	}
}

// ─── 提供者 Name / 已知格式 ────────────────────────────────

func TestEdgeProvider_Format(t *testing.T) {
	e := NewEdgeTTS()
	if e.Name() != "edge" {
		t.Errorf("edge Name = %q", e.Name())
	}
	// 空文本直接报错（不联网），验证 SynthesizeWithMime 透传
	if _, _, err := e.SynthesizeWithMime(""); err == nil {
		t.Error("空文本应报错")
	}
}

func TestWinProvider_Format(t *testing.T) {
	w := NewWinTTS()
	if w.Name() != "sapi" {
		t.Errorf("sapi Name = %q", w.Name())
	}
	// 不调用 SynthesizeWithMime（会拉起 PowerShell 子进程）；编译期断言方法存在，
	// 格式由 SAPI 输出 WAV 固化（SynthesizeWithMime 返回 audio/wav）。
	var _ func(string) ([]byte, string, error) = w.SynthesizeWithMime
}
