package app

import (
	"errors"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/tts"
)

// fakeTTSProvider 测试用 TTS 提供者（不联网）。
type fakeTTSProvider struct {
	name string
	out  []byte
	mime string
	err  error
}

func (f *fakeTTSProvider) Name() string                      { return f.name }
func (f *fakeTTSProvider) Synthesize(string) ([]byte, error) { return f.out, f.err }
func (f *fakeTTSProvider) SynthesizeWithMime(string) ([]byte, string, error) {
	return f.out, f.mime, f.err
}

// TestSpeakFromTTSProviderSteps_Fallback 首个失败 → 回退下一个；首个成功即返回。
func TestSpeakFromTTSProviderSteps_Fallback(t *testing.T) {
	a := &mediaState{core: &core{}}
	steps := []ttsPipelineStep{
		{provider: &fakeTTSProvider{name: "herdsman", err: errors.New("down")}},
		{provider: &fakeTTSProvider{name: "edge", out: []byte("ok"), mime: "audio/mp3"}},
	}
	audio, mime, err := a.speakFromTTSProviderSteps("hi", steps)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(audio) != "ok" || mime != "audio/mp3" {
		t.Errorf("应回退到第二个步骤: %q/%q", audio, mime)
	}
}

// TestSpeakFromTTSProviderSteps_AllFail 全部失败 → 报错（不静默成功）。
func TestSpeakFromTTSProviderSteps_AllFail(t *testing.T) {
	a := &mediaState{core: &core{}}
	steps := []ttsPipelineStep{
		{provider: &fakeTTSProvider{name: "a", err: errors.New("down")}},
		{provider: &fakeTTSProvider{name: "b", err: errors.New("down")}},
	}
	if _, _, err := a.speakFromTTSProviderSteps("hi", steps); err == nil {
		t.Fatal("全部失败应报错")
	}
}

// TestSpeakFromTTSProviderSteps_EmptyMimeDefaults 空 MIME → audio/mp3（历史一致）。
func TestSpeakFromTTSProviderSteps_EmptyMimeDefaults(t *testing.T) {
	a := &mediaState{core: &core{}}
	steps := []ttsPipelineStep{{provider: &fakeTTSProvider{name: "b", out: []byte("ok")}}}
	audio, mime, err := a.speakFromTTSProviderSteps("hi", steps)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(audio) != "ok" || mime != "audio/mp3" {
		t.Errorf("空 MIME 应兜底 audio/mp3: %q/%q", audio, mime)
	}
}

// TestTTSPipeline_RegistryDrivenOrder 管线顺序（配置驱动）：
// 用户选中模型 → 扫描引擎 TTS 模型（非 TTS 模型不入链）→ Edge → SAPI。
func TestTTSPipeline_RegistryDrivenOrder(t *testing.T) {
	a := newVoiceSettingsTestState(t, []modelengine.ModelInfo{
		{ID: "qwen3-tts-customvoice", Status: "running"},
		{ID: "whisper-base", Status: "running"}, // 非 TTS，不应进管线
		{ID: "voxcpm2"},                         // Status 空 = 通过
	})
	a.activeTTSEngine = "herdsman"
	a.activeTTSModel = "qwen3-tts-customvoice"

	steps := a.ttsProviderPipeline(true)
	if len(steps) != 5 {
		t.Fatalf("应有 5 步（选中 1 + 扫描 2 + edge + sapi）, got %d: %+v", len(steps), steps)
	}
	// 1. 用户选中（herdsman）
	if steps[0].provider.Name() != "herdsman" || steps[0].engineID != "herdsman" {
		t.Errorf("步骤0 应为用户选中 herdsman: %+v", steps[0])
	}
	// 2-3. 扫描：qwen3-tts-customvoice（重复命中）+ voxcpm2
	herdCount := 0
	for _, st := range steps {
		if st.provider.Name() == "herdsman" {
			herdCount++
		}
	}
	if herdCount != 3 {
		t.Errorf("herdsman 步骤应为 3（选中 + qwen3 扫描 + voxcpm2 扫描，whisper-base 应被排除）, got %d", herdCount)
	}
	// 尾部：edge + sapi
	if steps[len(steps)-2].provider.Name() != "edge" || steps[len(steps)-1].provider.Name() != "sapi" {
		t.Errorf("尾部应为 edge+sapi: %+v / %+v", steps[len(steps)-2], steps[len(steps)-1])
	}
}

// TestTTSPipeline_NoScan 关闭扫描时管线为 选中 + edge + sapi。
func TestTTSPipeline_NoScan(t *testing.T) {
	a := newVoiceSettingsTestState(t, []modelengine.ModelInfo{{ID: "qwen3-tts-customvoice", Status: "running"}})
	a.activeTTSEngine = "herdsman"
	a.activeTTSModel = "qwen3-tts-customvoice"

	steps := a.ttsProviderPipeline(false)
	if len(steps) != 3 {
		t.Fatalf("无扫描应有 3 步（选中 + edge + sapi）, got %d", len(steps))
	}
	if steps[0].provider.Name() != "herdsman" || steps[1].provider.Name() != "edge" || steps[2].provider.Name() != "sapi" {
		t.Errorf("顺序应为 选中/edge/sapi: %v", steps)
	}
}

// TestTTSProviderForEngine_XaiRequiresClient xAI 缺 client（OAuth token）→ 不可用。
func TestTTSProviderForEngine_XaiRequiresClient(t *testing.T) {
	a := newVoiceSettingsTestState(t, nil)
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{ID: "xai", Enabled: true}); err != nil {
		t.Fatalf("SaveEngine xai: %v", err)
	}
	if _, ok := a.ttsProviderForEngine("xai", "grok-tts", ""); ok {
		t.Fatal("client 为 nil 时 xai 提供者应不可用")
	}
}

// TestTTSProviderForEngine_UnknownEngine 未启用引擎 → 不可用（fail-closed）。
func TestTTSProviderForEngine_UnknownEngine(t *testing.T) {
	a := &mediaState{core: &core{}} // engineMgr nil
	if _, ok := a.ttsProviderForEngine("herdsman", "qwen3-tts-customvoice", ""); ok {
		t.Fatal("engineMgr nil 时应不可用")
	}
}

// TestTTSPipeline_NoEngineDefaultsToEdgeSapi 无 engineMgr/无选中模型时，
// 管线退化为 Edge + SAPI（兜底链，与 TTSSpeakBase64 第 3/4 级一致）。
func TestTTSPipeline_NoEngineDefaultsToEdgeSapi(t *testing.T) {
	a := &mediaState{core: &core{}} // engineMgr nil
	steps := a.ttsProviderPipeline(true)
	if len(steps) != 2 {
		t.Fatalf("无引擎时应有 edge+sapi 两步, got %d: %+v", len(steps), steps)
	}
	if steps[0].provider.Name() != "edge" || steps[1].provider.Name() != "sapi" {
		t.Errorf("兜底链应为 edge+sapi: %v / %v", steps[0].provider.Name(), steps[1].provider.Name())
	}
}

// TestTTSStreamingProviders_HerdsmanFirst 流式合成器链（注册表驱动）：
// herdsman 模型（本地优先）→ edge → sapi；引擎未配置时纯 edge/sapi 兜底。
func TestTTSStreamingProviders_HerdsmanFirst(t *testing.T) {
	a := newVoiceSettingsTestState(t, []modelengine.ModelInfo{{ID: "edge-tts", Status: "running"}})
	providers := a.ttsStreamingProviders()
	names := make([]string, 0, len(providers))
	for _, p := range providers {
		names = append(names, p.Name())
	}
	// 期望：4 个 herdsman（edge-tts/customvoice/voicedesign/voxcpm2）+ edge + sapi
	if len(names) != 6 {
		t.Fatalf("流式链应有 6 个提供者（4 herdsman + edge + sapi）, got %d: %v", len(names), names)
	}
	if names[0] != "herdsman" || names[len(names)-2] != "edge" || names[len(names)-1] != "sapi" {
		t.Errorf("链顺序应为 herdsman...edge,sapi: %v", names)
	}
}

// TestIsTTSModelID 模型分类口径（与 TTSSpeakBase64 扫描一致）。
func TestIsTTSModelID(t *testing.T) {
	cases := map[string]bool{
		"qwen3-tts-customvoice":                  true,
		"edge-tts":                               true,
		"voxcpm2":                                true,
		"grok-tts":                               true,
		"whisper-base":                           false,
		"sherpa-onnx-streaming-zipformer-zh-14m": false,
	}
	for id, want := range cases {
		if got := isTTSModelID(id); got != want {
			t.Errorf("isTTSModelID(%q) = %v, want %v", id, got, want)
		}
	}
}

var _ tts.TTSProvider = (*fakeTTSProvider)(nil)
