package tts

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/netclient"
)

// HerdsmanTTS 通过 Herdsman 的 OpenAI 兼容 TTS 端点合成语音
type HerdsmanTTS struct {
	baseURL          string
	model            string
	voice            string
	voiceDescription string // voicedesign 模式的音色描述
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

// audioInfoResponse GET /v1/audio/info 响应
type audioInfoResponse struct {
	SupportedSpeakers []string `json:"supported_speakers"`
}

// speakerCacheEntry 某个模型的支持音色缓存
type speakerCacheEntry struct {
	speakers []string
	fetched  time.Time
}

var (
	speakerCacheMu sync.Mutex
	speakerCache   = map[string]speakerCacheEntry{}
)

const speakerCacheTTL = 10 * time.Minute

// preferredSpeakers qwen3-tts 音色回退优先级（在服务端 supported_speakers 内选择）
var preferredSpeakers = []string{"serena", "vivian", "sohee", "aiden", "ryan", "eric"}

// Synthesize 合成语音（POST /audio/speech → 解析 audio_url → 获取音频）
func (h *HerdsmanTTS) Synthesize(text string) ([]byte, error) {
	audio, _, err := h.SynthesizeWithMime(text)
	return audio, err
}

// SynthesizeWithMime 合成并返回音频和实际 MIME 类型
func (h *HerdsmanTTS) SynthesizeWithMime(text string) ([]byte, string, error) {
	voice := h.resolveVoice()
	body := map[string]interface{}{
		"model": h.model,
		"input": text,
	}
	if h.voiceDescription != "" {
		body["voice_description"] = h.voiceDescription
	} else if voice != "" {
		body["voice"] = voice
	}
	jsonBody, _ := json.Marshal(body)

	audio, mime, err := h.requestAudio(jsonBody, voice)
	if err == nil {
		return audio, mime, nil
	}

	// 首选音色失败时（如配置了 Cherry 但服务端已不支持），用默认音色重试一次
	dv := defaultVoiceForModel(h.model)
	if voice != dv && dv != "" {
		defaultBody := map[string]interface{}{
			"model": h.model,
			"input": text,
		}
		if h.voiceDescription != "" {
			defaultBody["voice_description"] = h.voiceDescription
		} else {
			defaultBody["voice"] = dv
		}
		if db, mErr := json.Marshal(defaultBody); mErr == nil {
			if audio2, mime2, err2 := h.requestAudio(db, dv); err2 == nil {
				return audio2, mime2, nil
			}
		}
	}
	return nil, "", err
}

// requestAudio 发送 POST 并将响应解析为音频字节
func (h *HerdsmanTTS) requestAudio(jsonBody []byte, voice string) ([]byte, string, error) {
	resp, err := h.client.Post(h.speechEndpoint(), "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, "", fmt.Errorf("herdsman TTS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, "", fmt.Errorf("herdsman TTS: HTTP %d (voice=%s): %s", resp.StatusCode, voice, string(errBody))
	}

	contentType := resp.Header.Get("Content-Type")
	// 1. 服务端直接返回二进制音频（兼容旧实现）
	if strings.HasPrefix(strings.ToLower(contentType), "audio/") {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, "", fmt.Errorf("herdsman TTS: read: %w", err)
		}
		return data, contentType, nil
	}

	// 2. JSON 响应：audio_url 可能是 data URI / 相对路径 / 绝对 URL
	var result ttsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("herdsman TTS: parse: %w", err)
	}
	return h.fetchAudio(result.AudioURL)
}

// fetchAudio 解析 audio_url 并获取音频字节
func (h *HerdsmanTTS) fetchAudio(audioURL string) ([]byte, string, error) {
	audioURL = strings.TrimSpace(audioURL)
	if audioURL == "" {
		return nil, "", fmt.Errorf("herdsman TTS: 响应中无 audio_url")
	}

	// data:audio/wav;base64,...
	if strings.HasPrefix(audioURL, "data:") {
		comma := strings.IndexByte(audioURL, ',')
		if comma < 0 {
			return nil, "", fmt.Errorf("herdsman TTS: 非法 data URI")
		}
		header := audioURL[:comma]
		payload := strings.TrimSpace(audioURL[comma+1:])
		mime := strings.TrimPrefix(strings.SplitN(header, ";", 2)[0], "data:")
		if !strings.Contains(strings.ToLower(header), ";base64") {
			return []byte(payload), mime, nil
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			data, err = base64.URLEncoding.DecodeString(payload)
		}
		if err != nil {
			return nil, "", fmt.Errorf("herdsman TTS: data URI base64 解码失败: %w", err)
		}
		if len(data) == 0 {
			return nil, "", fmt.Errorf("herdsman TTS: audio 为空")
		}
		return data, mime, nil
	}

	// 绝对 URL
	if strings.HasPrefix(audioURL, "http://") || strings.HasPrefix(audioURL, "https://") {
		data, err := h.downloadAudio(audioURL)
		return data, "audio/wav", err
	}

	// 相对路径：先试 /v1 前缀，再试根路径
	if strings.HasPrefix(audioURL, "/") {
		candidates := []string{
			openAIEndpoint(h.baseURL, audioURL),
			stripV1Prefix(h.baseURL) + audioURL,
		}
		var lastErr error
		for _, c := range candidates {
			data, err := h.downloadAudio(c)
			if err == nil {
				return data, "audio/wav", nil
			}
			lastErr = err
		}
		return nil, "", fmt.Errorf("herdsman TTS: download: %w", lastErr)
	}
	return nil, "", fmt.Errorf("herdsman TTS: 未知 audio_url: %s", audioURL)
}

func (h *HerdsmanTTS) downloadAudio(u string) ([]byte, error) {
	audioResp, err := h.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer audioResp.Body.Close()

	if audioResp.StatusCode != 200 {
		return nil, fmt.Errorf("herdsman TTS: download HTTP %d", audioResp.StatusCode)
	}

	data, err := io.ReadAll(audioResp.Body)
	if err != nil {
		return nil, fmt.Errorf("herdsman TTS: read: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("herdsman TTS: audio 为空")
	}
	return data, nil
}

// speechEndpoint 处理 /v1 前缀后的合成端点
func (h *HerdsmanTTS) speechEndpoint() string {
	return openAIEndpoint(h.baseURL, "/audio/speech")
}

func (h *HerdsmanTTS) infoEndpoint() string {
	return openAIEndpoint(h.baseURL, "/audio/info")
}

// resolveVoice 返回服务端支持的有效音色：
//   - voicedesign 模式不传 voice
//   - 查询 /v1/audio/info 获取支持音色列表（带缓存），优先使用配置音色
//   - 配置音色不在列表时回退到首选默认音色
//   - 查询失败时回退 defaultVoiceForModel
func (h *HerdsmanTTS) resolveVoice() string {
	if h.voiceDescription != "" {
		return ""
	}
	voice := strings.TrimSpace(h.voice)
	if voice == "" {
		voice = defaultVoiceForModel(h.model)
	}

	// 仅对 qwen3-tts 系列模型动态查询支持音色（其他模型如 edge-tts 直接使用配置音色）
	l := strings.ToLower(h.model)
	if !strings.Contains(l, "qwen3-tts") && !strings.Contains(l, "customvoice") {
		return voice
	}

	speakers := h.SupportedSpeakers()
	if len(speakers) == 0 {
		return voice
	}
	for _, s := range speakers {
		if strings.EqualFold(s, voice) {
			return s
		}
	}
	for _, p := range preferredSpeakers {
		for _, s := range speakers {
			if strings.EqualFold(s, p) {
				return s
			}
		}
	}
	return speakers[0]
}

// SupportedSpeakers 获取模型支持音色列表（带 10 分钟缓存）
func (h *HerdsmanTTS) SupportedSpeakers() []string {
	speakerCacheMu.Lock()
	defer speakerCacheMu.Unlock()
	if e, ok := speakerCache[h.model]; ok && time.Since(e.fetched) < speakerCacheTTL {
		return e.speakers
	}

	var speakers []string
	u := h.infoEndpoint() + "?model=" + url.QueryEscape(h.model)
	resp, err := h.client.Get(u)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var info audioInfoResponse
			if decErr := json.NewDecoder(resp.Body).Decode(&info); decErr == nil && len(info.SupportedSpeakers) > 0 {
				speakers = info.SupportedSpeakers
			}
		}
	}
	if len(speakers) == 0 {
		return nil
	}
	speakerCache[h.model] = speakerCacheEntry{speakers: speakers, fetched: time.Now()}
	return speakers
}

// defaultVoiceForModel 返回各类模型的默认音色
func defaultVoiceForModel(model string) string {
	l := strings.ToLower(model)
	switch {
	case strings.Contains(l, "edge"):
		return "zh-CN-YunxiNeural"
	case strings.Contains(l, "voicedesign"):
		return ""
	case strings.Contains(l, "cosyvoice"):
		return "中文女"
	default:
		return "serena"
	}
}

// openAIEndpoint 将 OpenAI 兼容 base URL 与 API 路径拼接：
// baseURL 可能以 /v1 结尾（Herdsman/Ollama/xAI），也可能不带（DeepSeek），统一得到 .../v1<path>
func openAIEndpoint(baseURL, path string) string {
	return stripV1Prefix(baseURL) + "/v1" + path
}

func stripV1Prefix(baseURL string) string {
	return strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/v1")
}
