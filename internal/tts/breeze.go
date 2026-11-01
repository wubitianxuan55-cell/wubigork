package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gaea/gaea/internal/netclient"
)

// BreezeTTS 通过 breeze-server（Breeze-TTS-2.cpp）的 OpenAI 兼容 JSON 路径合成语音
// （POST /v1/audio/speech，Content-Type: application/json）。
//
// 与 herdsman kind 的字段差异（breeze-server docs/server.md「OpenAI compatible
// requests」节）：音色描述走 instructions（不是 voice_description）；voice 只接受
// 已保存/缓存的音色 id，未知名 404——留空即 voice design 模式（instructions 即
// 音色描述）；speed 被服务端忽略（模型无语速控制）；响应恒为二进制音频流，
// response_format=wav → audio/wav（24kHz 16bit 单声道，浏览器可播）。
type BreezeTTS struct {
	baseURL      string
	model        string
	voice        string
	instructions string // 音色描述（design 模式）或语气执导（克隆模式）
	client       *http.Client
}

// NewBreezeTTS 创建 Breeze TTS 客户端。
func NewBreezeTTS(baseURL, model, voice, instructions string) *BreezeTTS {
	return &BreezeTTS{
		baseURL:      baseURL,
		model:        model,
		voice:        voice,
		instructions: instructions,
		client:       netclient.NewSimpleClient(ttsTimeoutForModel(model)),
	}
}

// Name 返回提供者 kind（seam 提供者自注册用）。
func (b *BreezeTTS) Name() string { return "breeze" }

func init() {
	// breeze kind 工厂：VoiceDescription → instructions（voice design / 语气执导
	// 双语义由服务端按是否带 voice 自行判定），voice 非空才透传（未知名 404）。
	RegisterTTSProvider("breeze", func(cfg TTSConfig) (TTSProvider, error) {
		return NewBreezeTTS(cfg.BaseURL, cfg.Model, cfg.Voice, cfg.VoiceDescription), nil
	})
}

// Synthesize 合成语音（默认参数）。
func (b *BreezeTTS) Synthesize(text string) ([]byte, error) {
	audio, _, err := b.SynthesizeWithMime(text)
	return audio, err
}

// SynthesizeWithMime 合成并返回音频和实际 MIME 类型（默认参数）。
func (b *BreezeTTS) SynthesizeWithMime(text string) ([]byte, string, error) {
	return b.SynthesizeWithParams(text, TTSParams{})
}

// SynthesizeWithParams 携带单次合成参数合成并返回音频和实际 MIME 类型。
// Emotion/Style 非空时并入 instructions（breeze 无结构化情绪字段，按自然语言
// 语气执导消费），并带 cfg_scale=2.5——服务端口径 cfg=1 时情感标签只是建议
// 倾向被念白跳过，2~3 才真正触发（实弹：代价仅 +11% 耗时）；Speed/Pitch 能力
// 外静默忽略（与服务端口径一致）。
func (b *BreezeTTS) SynthesizeWithParams(text string, p TTSParams) ([]byte, string, error) {
	body := map[string]interface{}{
		"model":           b.model, // 服务端忽略，恒发以兼容必带 model 的调用方
		"input":           text,
		"response_format": "wav",
	}
	if strings.TrimSpace(p.Emotion) != "" || strings.TrimSpace(p.Style) != "" {
		body["cfg_scale"] = 2.5
	}
	if v := strings.TrimSpace(b.voice); v != "" {
		body["voice"] = v
	}
	if ins := b.buildInstructions(p); ins != "" {
		body["instructions"] = ins
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, "", fmt.Errorf("breeze TTS: %w", err)
	}

	resp, err := b.client.Post(b.speechEndpoint(), "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, "", fmt.Errorf("breeze TTS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, "", fmt.Errorf("breeze TTS: HTTP %d (voice=%s): %s", resp.StatusCode, b.voice, string(errBody))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "audio/") {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, "", fmt.Errorf("breeze TTS: 非音频响应 Content-Type=%s: %s", contentType, string(errBody))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("breeze TTS: read: %w", err)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("breeze TTS: audio 为空")
	}
	return data, contentType, nil
}

// buildInstructions 由音色描述与单次合成参数拼语气执导文本。
// design 模式下 instructions 即音色描述；克隆模式下为语气执导——两种语义
// 服务端自行判定，这里只做拼接。Emotion/Style 原词透传（调用方给的是
// 中文情绪词，breeze 按自然语言理解）。
func (b *BreezeTTS) buildInstructions(p TTSParams) string {
	parts := make([]string, 0, 3)
	if s := strings.TrimSpace(b.instructions); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(p.Emotion); s != "" {
		parts = append(parts, "用"+s+"的语气")
	}
	if s := strings.TrimSpace(p.Style); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "，")
}

// speechEndpoint 处理 /v1 前缀后的合成端点。
func (b *BreezeTTS) speechEndpoint() string {
	return openAIEndpoint(b.baseURL, "/audio/speech")
}
