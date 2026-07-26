package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HerdsmanTTS 通过 Herdsman 的 OpenAI 兼容 TTS 端点合成语音
type HerdsmanTTS struct {
	baseURL string
	model   string
	voice   string
	client  *http.Client
}

// NewHerdsmanTTS 创建 Herdsman TTS 客户端
func NewHerdsmanTTS(baseURL, model, voice string) *HerdsmanTTS {
	if voice == "" {
		voice = "aiden"
	}
	return &HerdsmanTTS{
		baseURL: baseURL,
		model:   model,
		voice:   voice,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Synthesize 实现 Synthesizer 接口
func (h *HerdsmanTTS) Synthesize(text string) ([]byte, error) {
	body := map[string]interface{}{
		"model":          h.model,
		"input":          text,
		"voice":          h.voice,
		"response_format": "mp3",
	}
	jsonBody, _ := json.Marshal(body)

	url := h.baseURL + "/v1/audio/speech"
	resp, err := h.client.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("herdsman TTS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("herdsman TTS: HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("herdsman TTS: read: %w", err)
	}

	return data, nil
}
