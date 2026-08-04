// Package tts — TTS 纯函数与构造测试
package tts

import (
	"errors"
	"strings"
	"testing"
)

var errEngineDown = errors.New("engine down")

// ─── SplitSentences ──────────────────────────────────────────

func TestSplitSentences_BasicPunctuation(t *testing.T) {
	got := SplitSentences("你好。今天天气不错！我们去散步吧？")
	if len(got) != 3 {
		t.Fatalf("应拆 3 句, got %d: %v", len(got), got)
	}
	if got[0] != "你好" || got[1] != "今天天气不错" || got[2] != "我们去散步吧" {
		t.Errorf("拆分内容不符: %v", got)
	}
}

func TestSplitSentences_NewlineSplit(t *testing.T) {
	got := SplitSentences("第一行\n第二行")
	if len(got) != 2 || got[0] != "第一行" || got[1] != "第二行" {
		t.Errorf("换行应拆分: %v", got)
	}
}

func TestSplitSentences_EmptyPartsSkipped(t *testing.T) {
	got := SplitSentences("你好。。！？")
	if len(got) != 1 || got[0] != "你好" {
		t.Errorf("连续标点应合并跳过空段: %v", got)
	}
}

func TestSplitSentences_ShortSentenceKeptWhole(t *testing.T) {
	got := SplitSentences("这是一段不长的句子，虽然有逗号但不足百字。")
	if len(got) != 1 || got[0] != "这是一段不长的句子，虽然有逗号但不足百字" {
		t.Errorf("短句（含逗号）应整体保留: %v", got)
	}
}

func TestSplitSentences_LongSentenceSubSplit(t *testing.T) {
	long := strings.Repeat("长", 60) + "，" + strings.Repeat("短", 60) + "。"
	got := SplitSentences(long)
	if len(got) != 2 {
		t.Fatalf("超百字长句应按逗号拆 2 段, got %d", len(got))
	}
	if len([]rune(got[0])) != 60 || len([]rune(got[1])) != 60 {
		t.Errorf("子段长度不符: %d/%d", len([]rune(got[0])), len([]rune(got[1])))
	}
}

func TestSplitSentences_TrimsWhitespace(t *testing.T) {
	got := SplitSentences("  你好  。  ")
	if len(got) != 1 || got[0] != "你好" {
		t.Errorf("应去除首尾空白: %v", got)
	}
}

func TestSplitSentences_EmptyInput(t *testing.T) {
	got := SplitSentences("")
	if len(got) != 0 {
		t.Errorf("空输入应返回空切片, got %v", got)
	}
}

// ─── escapeSSML ──────────────────────────────────────────────

func TestEscapeSSML(t *testing.T) {
	cases := []struct{ in, want string }{
		{"AT&T", "AT&amp;T"},
		{"<b>", "&lt;b&gt;"},
		{"a < b & c > d", "a &lt; b &amp; c &gt; d"},
		{"plain", "plain"},
	}
	for _, c := range cases {
		if got := escapeSSML(c.in); got != c.want {
			t.Errorf("escapeSSML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ─── computeAcceptKey（WebSocket 握手，RFC 6455 标准向量） ──

func TestComputeAcceptKey_RFC6455Vector(t *testing.T) {
	// RFC 6455 §1.3 示例：dGhlIHNhbXBsZSBub25jZQ== → s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
	got := computeAcceptKey("dGhlIHNhbXBsZSBub25jZQ==")
	want := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	if got != want {
		t.Errorf("RFC6455 向量不符: got %q want %q", got, want)
	}
}

// ─── escapePath（SAPI） ──────────────────────────────────────

func TestEscapePath(t *testing.T) {
	if got := escapePath("C:\\voice\\a'b.wav"); got != "C:\\voice\\a''b.wav" {
		t.Errorf("单引号应翻倍: %q", got)
	}
	if got := escapePath("plain.wav"); got != "plain.wav" {
		t.Errorf("无引号应原样: %q", got)
	}
}

// ─── truncStr（rune 截断） ───────────────────────────────────

func TestTruncStr_Short(t *testing.T) {
	if got := truncStr("你好世界", 10); got != "你好世界" {
		t.Errorf("短文本不应截断: %q", got)
	}
}

func TestTruncStr_Long(t *testing.T) {
	got := truncStr("一二三四五六七八九十", 5)
	if got != "一二三四五..." {
		t.Errorf("超长应截断加省略号: %q", got)
	}
}

func TestTruncStr_ExactBoundary(t *testing.T) {
	if got := truncStr("你好", 2); got != "你好" {
		t.Errorf("恰好长度不应截断: %q", got)
	}
}

// ─── randomHex ───────────────────────────────────────────────

func TestRandomHex_LengthAndFormat(t *testing.T) {
	got := randomHex(8)
	if len(got) != 16 {
		t.Errorf("8 字节应得 16 个十六进制字符, got %q len=%d", got, len(got))
	}
	for _, c := range got {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("含非十六进制字符: %q", got)
			break
		}
	}
}

func TestRandomHex_NonDeterministic(t *testing.T) {
	if randomHex(16) == randomHex(16) {
		t.Error("两次随机应不同")
	}
}

// ─── EdgeVoices ──────────────────────────────────────────────

func TestEdgeVoices_ChineseNeural(t *testing.T) {
	vs := EdgeVoices()
	if len(vs) < 3 {
		t.Fatalf("至少应有 3 个中文音色, got %d", len(vs))
	}
	for _, v := range vs {
		if !strings.HasPrefix(v, "zh-CN-") {
			t.Errorf("应全为中文音色: %q", v)
		}
	}
}

// ─── generateSecMSGec（Edge token） ──────────────────────────

func TestGenerateSecMSGec_Format(t *testing.T) {
	got := generateSecMSGec()
	if len(got) != 64 {
		t.Errorf("应为 64 位 SHA256 十六进制, got len=%d", len(got))
	}
	if got != strings.ToUpper(got) {
		t.Error("应为大写十六进制")
	}
	for _, c := range got {
		if !strings.ContainsRune("0123456789ABCDEF", c) {
			t.Errorf("含非大写十六进制字符: %q", got)
			break
		}
	}
}

// ─── 构造函数（零依赖初始化） ───────────────────────────────

func TestNewHerdsmanTTS_SetsFields(t *testing.T) {
	h := NewHerdsmanTTS("http://localhost:8080", "qwen3-tts-customvoice", "zh-CN-YunxiNeural")
	if h.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL 未设置: %q", h.baseURL)
	}
	if h.model != "qwen3-tts-customvoice" || h.voice != "zh-CN-YunxiNeural" {
		t.Errorf("model/voice 未设置: %q/%q", h.model, h.voice)
	}
	if h.client == nil {
		t.Error("client 应为非 nil")
	}
}

func TestNewHerdsmanTTSWithDesc_SetsDescription(t *testing.T) {
	h := NewHerdsmanTTSWithDesc("http://localhost:8080", "qwen3-tts-voicedesign", "用温柔的语气说")
	if h.voiceDescription != "用温柔的语气说" {
		t.Errorf("voiceDescription 未设置: %q", h.voiceDescription)
	}
	if h.voice != "" {
		t.Errorf("voicedesign 模式不应设置 voice: %q", h.voice)
	}
	if h.client == nil {
		t.Error("client 应为非 nil")
	}
}

func TestNewEdgeTTS_NonNil(t *testing.T) {
	e := NewEdgeTTS()
	if e == nil {
		t.Fatal("NewEdgeTTS 不应返回 nil")
	}
}

func TestNewWinTTS_NonNil(t *testing.T) {
	w := NewWinTTS()
	if w == nil {
		t.Fatal("NewWinTTS 不应返回 nil")
	}
}

// ─── SynthesizerChain（引擎回退组合逻辑） ──────────────────

type fakeSynth struct {
	out []byte
	err error
}

func (f *fakeSynth) Synthesize(text string) ([]byte, error) {
	return f.out, f.err
}

func TestSynthesizerChain_FirstSuccessWins(t *testing.T) {
	chain := NewSynthesizerChain(
		&fakeSynth{out: []byte("first"), err: nil},
		&fakeSynth{out: []byte("second"), err: nil},
	)
	metas := []struct {
		Label  string
		Format string
	}{{Label: "herdsman", Format: "wav"}}
	data, format, label, err := chain.SynthesizeWithMeta("测试", metas)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if string(data) != "first" {
		t.Errorf("应取第一个引擎结果, got %q", data)
	}
	if format != "wav" || label != "herdsman" {
		t.Errorf("应使用对应 meta, got %q/%q", format, label)
	}
}

func TestSynthesizerChain_FallbackOnFailure(t *testing.T) {
	chain := NewSynthesizerChain(
		&fakeSynth{err: errEngineDown},
		&fakeSynth{out: []byte("backup"), err: nil},
	)
	data, _, _, err := chain.SynthesizeWithMeta("测试", nil)
	if err != nil {
		t.Fatalf("第二个引擎成功不应报错: %v", err)
	}
	if string(data) != "backup" {
		t.Errorf("应回退到第二个引擎, got %q", data)
	}
}

func TestSynthesizerChain_AllFail(t *testing.T) {
	chain := NewSynthesizerChain(
		&fakeSynth{err: errEngineDown},
		&fakeSynth{err: errEngineDown},
	)
	_, _, _, err := chain.SynthesizeWithMeta("测试", nil)
	if err == nil {
		t.Fatal("全部失败应返回错误")
	}
}

func TestSynthesizerChain_NoEngines(t *testing.T) {
	chain := NewSynthesizerChain()
	_, _, _, err := chain.SynthesizeWithMeta("测试", nil)
	if err == nil {
		t.Fatal("无引擎应返回错误")
	}
}

func TestSynthesizerChain_MetaFallbackDefaults(t *testing.T) {
	chain := NewSynthesizerChain(&fakeSynth{out: []byte("x"), err: nil})
	data, format, label, err := chain.SynthesizeWithMeta("测试", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(data) != "x" || format != "wav" || label != "unknown" {
		t.Errorf("无 meta 应回退默认 wav/unknown, got %q/%q/%q", data, format, label)
	}
}
