package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// XaiTTS 使用 xAI TTS API 进行语音合成（最高优先级，在线高质量）
type XaiTTS struct {
	apiBase    string
	voiceID    string
	getToken   func() (string, error)
	httpClient *http.Client
}

// NewXaiTTS 创建 xAI TTS 合成器
//   - apiBase: xAI API 基础 URL，如 "https://api.x.ai/v1"
//   - voiceID: 语音 ID，可选 eve/ara/rex/sal/leo，默认 eve
//   - getToken: 返回 Bearer token 的函数，复用 ai.Client.GetToken
//   - httpClient: HTTP 客户端，30s 超时推荐
func NewXaiTTS(apiBase, voiceID string, getToken func() (string, error), httpClient *http.Client) *XaiTTS {
	if voiceID == "" {
		voiceID = "eve"
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &XaiTTS{apiBase: apiBase, voiceID: voiceID, getToken: getToken, httpClient: httpClient}
}

// Synthesize 实现 tts.Synthesizer 接口
func (x *XaiTTS) Synthesize(text string) ([]byte, error) {
	token, err := x.getToken()
	if err != nil {
		return nil, fmt.Errorf("xAI TTS 认证失败: %w", err)
	}

	body, err := json.Marshal(map[string]string{
		"text": text, "voice_id": x.voiceID, "language": "zh",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal xAI TTS 请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", x.apiBase+"/tts", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造 xAI TTS 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := x.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xAI TTS 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return nil, fmt.Errorf("xAI TTS HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 xAI TTS 音频失败: %w", err)
	}

	slog.Info("xAI TTS 合成完成", "bytes", len(audio), "voice", x.voiceID, "text", truncStr(text, 40))
	return audio, nil
}

func truncStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
