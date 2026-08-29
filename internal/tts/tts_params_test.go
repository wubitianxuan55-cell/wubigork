// Package tts — TTS 风格/情绪参数扩展测试（v4.3d）。
//
// 固化 SynthesizeWithParams 契约：
//   - edge：SSML 由参数生成（Speed→rate、Pitch→Hz），零值保持默认
//   - herdsman：buildBody 在 Speed>0/Emotion/Style 下携带对应字段；
//     cosyvoice 工厂不再丢弃 voiceDescription（请求体含 voice_description）
//   - xai/sapi：忽略参数不报错（能力外）
//   - Synthesize(text) 兼容路径（默认参数）行为不变
package tts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 编译期断言：四个提供者均实现完整 TTSProvider 接口（含 SynthesizeWithParams）。
var (
	_ TTSProvider = (*EdgeTTS)(nil)
	_ TTSProvider = (*HerdsmanTTS)(nil)
	_ TTSProvider = (*XaiTTS)(nil)
	_ TTSProvider = (*WinTTS)(nil)
)

// ─── edge：SSML 参数化 ────────────────────────────────────────

func TestEdgeBuildSSML_Params(t *testing.T) {
	e := NewEdgeTTS()

	// 零值 → 保持现默认（pitch='+0Hz' rate='+0%'）
	ssml := e.buildSSML("你好", TTSParams{})
	if !strings.Contains(ssml, "pitch='+0Hz' rate='+0%' volume='+0%'") {
		t.Errorf("零值应保持默认 prosody: %s", ssml)
	}

	// Speed 1.2 → rate='+20%'
	if got := e.buildSSML("你好", TTSParams{Speed: 1.2}); !strings.Contains(got, "rate='+20%'") {
		t.Errorf("Speed 1.2 应为 rate='+20%%': %s", got)
	}
	// Speed 0.9 → rate='-10%'
	if got := e.buildSSML("你好", TTSParams{Speed: 0.9}); !strings.Contains(got, "rate='-10%'") {
		t.Errorf("Speed 0.9 应为 rate='-10%%': %s", got)
	}
	// Pitch 2 → pitch='+2Hz'（1 半音 ≈ 1Hz 简化近似）
	if got := e.buildSSML("你好", TTSParams{Pitch: 2}); !strings.Contains(got, "pitch='+2Hz'") {
		t.Errorf("Pitch 2 应为 pitch='+2Hz': %s", got)
	}
	// Pitch -3 → pitch='-3Hz'
	if got := e.buildSSML("你好", TTSParams{Pitch: -3}); !strings.Contains(got, "pitch='-3Hz'") {
		t.Errorf("Pitch -3 应为 pitch='-3Hz': %s", got)
	}
	// 组合参数 + 文本转义仍生效
	got := e.buildSSML("a < b", TTSParams{Speed: 1.5, Pitch: 2})
	for _, want := range []string{"rate='+50%'", "pitch='+2Hz'", "a &lt; b"} {
		if !strings.Contains(got, want) {
			t.Errorf("组合参数缺少 %q: %s", want, got)
		}
	}
}

func TestEdgeSynthesize_EmptyTextCompat(t *testing.T) {
	e := NewEdgeTTS()
	// 兼容路径：空文本直接报错（不联网），与历史行为一致
	if _, err := e.Synthesize(""); err == nil {
		t.Error("Synthesize 空文本应报错（兼容路径行为不变）")
	}
	if _, _, err := e.SynthesizeWithMime(""); err == nil {
		t.Error("SynthesizeWithMime 空文本应报错（兼容路径行为不变）")
	}
	// 带参数路径同样校验空文本
	if _, _, err := e.SynthesizeWithParams("", TTSParams{Speed: 1.2, Pitch: 2, Style: "s", Emotion: "HAPPY"}); err == nil {
		t.Error("SynthesizeWithParams 空文本应报错")
	}
}

// ─── herdsman：buildBody 参数化 ───────────────────────────────

func TestHerdsmanBuildBody_Params(t *testing.T) {
	h := NewHerdsmanTTS("http://localhost:8080/v1", "qwen3-tts-customvoice", "serena")

	// 零值（兼容路径 buildBody）→ 无 speed/emotion/style
	body := h.buildBody("文本", "serena")
	if _, ok := body["speed"]; ok {
		t.Errorf("零值不应携带 speed: %+v", body)
	}
	if _, ok := body["emotion"]; ok {
		t.Errorf("零值不应携带 emotion: %+v", body)
	}
	if _, ok := body["style"]; ok {
		t.Errorf("零值不应携带 style: %+v", body)
	}
	if body["voice"] != "serena" {
		t.Errorf("零值应保留 voice: %+v", body)
	}

	// Speed>0 / Emotion / Style 均携带
	body = h.buildBodyParams("文本", "serena", TTSParams{Speed: 1.2, Emotion: "HAPPY", Style: "narration"})
	if body["speed"] != 1.2 {
		t.Errorf("speed 应为 1.2, got %v", body["speed"])
	}
	if body["emotion"] != "HAPPY" {
		t.Errorf("emotion 应为 HAPPY, got %v", body["emotion"])
	}
	if body["style"] != "narration" {
		t.Errorf("style 应为 narration, got %v", body["style"])
	}
	if body["voice"] != "serena" {
		t.Errorf("参数化不应影响 voice: %+v", body)
	}

	// Speed=0（不指定）但 Emotion/Style 非空 → 只带 emotion/style
	body = h.buildBodyParams("文本", "serena", TTSParams{Emotion: "CALM"})
	if _, ok := body["speed"]; ok {
		t.Errorf("Speed 0 不应携带 speed: %+v", body)
	}
	if body["emotion"] != "CALM" {
		t.Errorf("emotion 应为 CALM: %+v", body)
	}
}

// TestHerdsmanSynthesizeWithParams_RequestBody 用 httptest server 捕获请求体 JSON，
// 断言 Speed/Emotion/Style 进入请求体（参照既有 herdsman 测试模式）。
func TestHerdsmanSynthesizeWithParams_RequestBody(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/audio/info") {
			_, _ = w.Write([]byte(`{"supported_speakers":["serena"]}`))
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("解析请求体失败: %v", err)
		}
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("wav-bytes"))
	}))
	defer srv.Close()

	h := NewHerdsmanTTS(srv.URL+"/v1", "qwen3-tts-customvoice", "serena")
	audio, mime, err := h.SynthesizeWithParams("要合成的文本", TTSParams{Speed: 1.2, Emotion: "HAPPY", Style: "narration"})
	if err != nil {
		t.Fatalf("SynthesizeWithParams: %v", err)
	}
	if string(audio) != "wav-bytes" || mime != "audio/wav" {
		t.Errorf("音频/MIME 不符: %q/%q", audio, mime)
	}
	if got["speed"] != 1.2 || got["emotion"] != "HAPPY" || got["style"] != "narration" {
		t.Errorf("请求体应含 speed/emotion/style: %+v", got)
	}
	if got["input"] != "要合成的文本" || got["voice"] != "serena" {
		t.Errorf("input/voice 不符: %+v", got)
	}

	// 兼容路径 Synthesize：默认参数 → 请求体无 speed/emotion/style
	got = nil
	if _, err := h.Synthesize("第二段文本"); err != nil {
		t.Fatalf("Synthesize 兼容路径: %v", err)
	}
	if _, ok := got["speed"]; ok {
		t.Errorf("默认参数不应携带 speed: %+v", got)
	}
	if _, ok := got["emotion"]; ok {
		t.Errorf("默认参数不应携带 emotion: %+v", got)
	}
	if _, ok := got["style"]; ok {
		t.Errorf("默认参数不应携带 style: %+v", got)
	}
}

// ─── herdsman：cosyvoice 保留 voiceDescription（v4.3d 修复） ──

func TestHerdsmanFactory_CosyVoiceKeepsDescription(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("解析请求体失败: %v", err)
		}
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("wav-bytes"))
	}))
	defer srv.Close()

	// cosyvoice 模型名 + 描述 → 工厂应走 WithDesc 构造（不再落 default 丢弃）
	desc := "温柔亲切的女声，语速平缓"
	p, err := NewTTSProvider("herdsman", TTSConfig{
		BaseURL:          srv.URL + "/v1",
		Model:            "cosyvoice-300m",
		Voice:            "中文女",
		VoiceDescription: desc,
	})
	if err != nil {
		t.Fatalf("NewTTSProvider(herdsman cosyvoice): %v", err)
	}
	h, ok := p.(*HerdsmanTTS)
	if !ok {
		t.Fatalf("类型应为 *HerdsmanTTS, got %T", p)
	}
	if h.voiceDescription != desc {
		t.Errorf("cosyvoice 构造应保留 voiceDescription=%q, got %q", desc, h.voiceDescription)
	}

	audio, mime, err := h.SynthesizeWithMime("测试文本")
	if err != nil {
		t.Fatalf("SynthesizeWithMime: %v", err)
	}
	if len(audio) == 0 || mime != "audio/wav" {
		t.Errorf("音频/MIME 不符: %d/%q", len(audio), mime)
	}
	if got["voice_description"] != desc {
		t.Errorf("cosyvoice 请求体应含 voice_description=%q: %+v", desc, got)
	}
	if _, has := got["voice"]; has {
		t.Errorf("cosyvoice voicedesign 模式不应携带 voice: %+v", got)
	}
	if _, has := got["speed"]; has {
		t.Errorf("默认参数不应携带 speed: %+v", got)
	}
}

// ─── xai / sapi：参数忽略不报错 ──────────────────────────────

func TestXaiSynthesizeWithParams_IgnoresParams(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("解析请求体失败: %v", err)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte{0xFF, 0xF3, 0x01, 0x02})
	}))
	defer srv.Close()

	xtts := NewXaiTTS(srv.URL, "eve", func() (string, error) { return "tok", nil }, nil)
	audio, mime, err := xtts.SynthesizeWithParams("好的，晚安。", TTSParams{Speed: 1.5, Pitch: 3, Style: "s", Emotion: "ANGRY"})
	if err != nil {
		t.Fatalf("SynthesizeWithParams 带参数不应报错: %v", err)
	}
	if len(audio) != 4 || mime != "audio/mpeg" {
		t.Errorf("音频/MIME 不符: %d/%q", len(audio), mime)
	}
	// 请求体保持 text/voice_id/language，不出现参数字段
	if gotBody["text"] != "好的，晚安。" || gotBody["voice_id"] != "eve" || gotBody["language"] != "zh" {
		t.Errorf("xAI 请求体不符: %+v", gotBody)
	}
	for _, k := range []string{"speed", "pitch", "style", "emotion"} {
		if _, has := gotBody[k]; has {
			t.Errorf("xAI 不应携带参数字段 %q: %+v", k, gotBody)
		}
	}
}

// TestSapiSynthesizeWithParams_Interface sapi 忽略参数（能力外，不报错）。
// 不实际调用（会拉起 PowerShell 子进程，与既有测试约定一致），
// 以编译期断言方法存在与签名匹配。
func TestSapiSynthesizeWithParams_Interface(t *testing.T) {
	w := NewWinTTS()
	var _ func(string, TTSParams) ([]byte, string, error) = w.SynthesizeWithParams
}
