package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/netclient"
)

// xaiVoices xAI Grok TTS 音色（大小写不敏感；经典 5 个 + 旗舰 21 个）
var xaiVoices = []string{
	"eve", "ara", "rex", "sal", "leo",
	"altair", "atlas", "carina", "castor", "celeste",
	"cosmo", "helios", "helix", "iris", "kepler",
	"lumen", "luna", "lux", "naksh", "orion",
	"perseus", "rigel", "sirius", "ursa", "zagan", "zenith",
}

// XaiSupportedVoices 返回 xAI Grok TTS 支持的音色列表
func XaiSupportedVoices() []string {
	out := make([]string, len(xaiVoices))
	copy(out, xaiVoices)
	return out
}

// IsXaiVoice 判断音色是否为 xAI 支持（大小写不敏感）
func IsXaiVoice(voice string) bool {
	v := strings.ToLower(strings.TrimSpace(voice))
	for _, s := range xaiVoices {
		if v == s {
			return true
		}
	}
	return false
}

// XaiTTS 使用 xAI TTS API 进行语音合成（云端在线，Grok 音色）
type XaiTTS struct {
	apiBase    string
	voiceID    string
	getToken   func() (string, error)
	httpClient *http.Client
}

// NewXaiTTS 创建 xAI TTS 合成器
//   - apiBase: xAI API 基础 URL，如 "https://api.x.ai/v1"
//   - voiceID: 音色 ID（eve/ara/rex/sal/leo 等），无效或空时回退 eve
//   - getToken: 返回 Bearer token 的函数，复用 ai.Client.GetToken
//   - httpClient: HTTP 客户端，30s 超时推荐
func NewXaiTTS(apiBase, voiceID string, getToken func() (string, error), httpClient *http.Client) *XaiTTS {
	if voiceID == "" || !IsXaiVoice(voiceID) {
		voiceID = "eve"
	}
	voiceID = strings.ToLower(voiceID)
	if httpClient == nil {
		httpClient = netclient.NewSimpleClient(30 * time.Second)
	}
	return &XaiTTS{apiBase: apiBase, voiceID: voiceID, getToken: getToken, httpClient: httpClient}
}

// Synthesize 实现 tts.Synthesizer 接口
func (x *XaiTTS) Synthesize(text string) ([]byte, error) {
	audio, _, err := x.SynthesizeWithMime(text)
	return audio, err
}

// Name 返回提供者 kind（seam 提供者自注册用）。
func (x *XaiTTS) Name() string { return "xai" }

func init() {
	RegisterTTSProvider("xai", func(cfg TTSConfig) (TTSProvider, error) {
		if cfg.GetToken == nil {
			return nil, fmt.Errorf("tts: xai provider 需要 GetToken（OAuth token 提供函数）")
		}
		return NewXaiTTS(cfg.BaseURL, cfg.Voice, cfg.GetToken, cfg.HTTPClient), nil
	})
}

// SynthesizeWithMime 合成语音并返回音频与 MIME（POST /v1/tts，返回 audio/mpeg）
func (x *XaiTTS) SynthesizeWithMime(text string) ([]byte, string, error) {
	token, err := x.getToken()
	if err != nil {
		return nil, "", fmt.Errorf("xAI TTS 认证失败: %w", err)
	}

	body, err := json.Marshal(map[string]string{
		"text": text, "voice_id": x.voiceID, "language": "zh",
	})
	if err != nil {
		return nil, "", fmt.Errorf("marshal xAI TTS 请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", x.apiBase+"/tts", bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("构造 xAI TTS 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := x.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("xAI TTS 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return nil, "", fmt.Errorf("xAI TTS HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取 xAI TTS 音频失败: %w", err)
	}

	slog.Info("xAI TTS 合成完成", "bytes", len(audio), "voice", x.voiceID, "text", truncStr(text, 40))
	return audio, "audio/mpeg", nil
}

// SynthesizeWithParams 合成语音并返回音频与 MIME。
// xAI 当前能力外忽略 Speed/Pitch/Style/Emotion（不报错，行为与 SynthesizeWithMime 一致）。
func (x *XaiTTS) SynthesizeWithParams(text string, p TTSParams) ([]byte, string, error) {
	return x.SynthesizeWithMime(text)
}

func truncStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
