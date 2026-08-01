package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
	"io"
	"net/http"
	"time"
)

// HerdsmanTTS 通过 Herdsman 的 OpenAI 兼容 TTS 端点合成语音
type HerdsmanTTS struct {
	baseURL          string
	model            string
	voice            string
	voiceDescription string // voicedesign 模式的声音描述
	client           *http.Client
}

// NewHerdsmanTTS 创建 Herdsman TTS 客户端（customvoice 模式）
func NewHerdsmanTTS(baseURL, model, voice string) *HerdsmanTTS {
	return &HerdsmanTTS{
		baseURL: baseURL,
		model:   model,
		voice:   voice,
		client:  netclient.NewSimpleClient(30 * time.Second),
	}
}

// NewHerdsmanTTSWithDesc 创建 Herdsman TTS 客户端（voicedesign 模式）
// 对齐 Ackem CosyVoice 的情感指令注入：voiceDescription 为中文自然语言描述
func NewHerdsmanTTSWithDesc(baseURL, model, voiceDescription string) *HerdsmanTTS {
	return &HerdsmanTTS{
		baseURL:          baseURL,
		model:            model,
		voiceDescription: voiceDescription,
		client:           netclient.NewSimpleClient(30 * time.Second),
	}
}

// ttsResponse /v1/audio/speech 非流式响应
type ttsResponse struct {
	AudioURL   string  `json:"audio_url"`
	SampleRate int     `json:"sample_rate"`
	Duration   float64 `json:"duration"`
}

// Synthesize 合成语音（POST /audio/speech → 解析 JSON audio_url → GET 下载音频）
func (h *HerdsmanTTS) Synthesize(text string) ([]byte, error) {
	body := map[string]interface{}{
		"model": h.model,
		"input": text,
	}
	if h.voiceDescription != "" {
		body["voice_description"] = h.voiceDescription
	} else if h.voice != "" {
		body["voice"] = h.voice
	}
	jsonBody, _ := json.Marshal(body)

	url := h.baseURL + "/audio/speech"
	resp, err := h.client.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("herdsman TTS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("herdsman TTS: HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var result ttsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("herdsman TTS: parse: %w", err)
	}
	if result.AudioURL == "" {
		return nil, fmt.Errorf("herdsman TTS: 响应中无 audio_url")
	}

	audioResp, err := h.client.Get(h.baseURL + result.AudioURL)
	if err != nil {
		return nil, fmt.Errorf("herdsman TTS: download: %w", err)
	}
	defer audioResp.Body.Close()

	if audioResp.StatusCode != 200 {
		return nil, fmt.Errorf("herdsman TTS: download HTTP %d", audioResp.StatusCode)
	}

	data, err := io.ReadAll(audioResp.Body)
	if err != nil {
		return nil, fmt.Errorf("herdsman TTS: read: %w", err)
	}
	return data, nil
}
