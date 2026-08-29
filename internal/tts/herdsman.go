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
	refAudio         string // voiceclone 模式的参考音频（路径、URL 或 data URI）
	refText          string // voiceclone 模式的参考音频文本（可选）
	client           *http.Client
}

// NewHerdsmanTTS 创建 Herdsman TTS 客户端（customvoice 模式）
func NewHerdsmanTTS(baseURL, model, voice string) *HerdsmanTTS {
	return &HerdsmanTTS{
		baseURL: baseURL,
		model:   model,
		voice:   voice,
		client:  netclient.NewSimpleClient(ttsTimeoutForModel(model)),
	}
}

// NewHerdsmanTTSWithDesc 创建 Herdsman TTS 客户端（voicedesign 模式）
func NewHerdsmanTTSWithDesc(baseURL, model, voiceDescription string) *HerdsmanTTS {
	return &HerdsmanTTS{
		baseURL:          baseURL,
		model:            model,
		voiceDescription: voiceDescription,
		client:           netclient.NewSimpleClient(ttsTimeoutForModel(model)),
	}
}

// NewHerdsmanTTSWithClone 创建 Herdsman TTS 客户端（voiceclone 模式）。
// refAudio 支持本地路径、URL 或 data URI，refText 为可选参考文本。
func NewHerdsmanTTSWithClone(baseURL, model, refAudio, refText string) *HerdsmanTTS {
	return &HerdsmanTTS{
		baseURL:  baseURL,
		model:    model,
		refAudio: refAudio,
		refText:  refText,
		client:   netclient.NewSimpleClient(ttsTimeoutForModel(model)),
	}
}

// ttsTimeoutForModel 为本地重模型（Qwen3-TTS / VoxCPM2）放宽 HTTP 超时，
// 避免冷启动或较长音频在 30 秒内被误判失败。
func ttsTimeoutForModel(model string) time.Duration {
	l := strings.ToLower(model)
	if strings.Contains(l, "qwen3-tts") || strings.Contains(l, "voxcpm") {
		return 180 * time.Second
	}
	return 30 * time.Second
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

// Name 返回提供者 kind（seam 提供者自注册用）。
func (h *HerdsmanTTS) Name() string { return "herdsman" }

func init() {
	// herdsman kind 工厂：按模型/配置路由到 customvoice / voicedesign / voiceclone 构造器，
	// 收敛 app 层 tryEngineTTS 的按模型分支（voicedesign/voxcpm 无描述时用空音色，
	// 与历史行为一致）。
	RegisterTTSProvider("herdsman", func(cfg TTSConfig) (TTSProvider, error) {
		l := strings.ToLower(cfg.Model)
		switch {
		case cfg.RefAudio != "" || strings.Contains(l, "voiceclone"):
			return NewHerdsmanTTSWithClone(cfg.BaseURL, cfg.Model, cfg.RefAudio, cfg.RefText), nil
		case strings.Contains(l, "voicedesign"), strings.Contains(l, "voxcpm"), strings.Contains(l, "cosyvoice"):
			// cosyvoice 同样走 voicedesign 分支：有描述则构造 WithDesc，
			// 不再落 default 分支丢弃 voiceDescription（v4.3d 修复）。
			if cfg.VoiceDescription != "" {
				return NewHerdsmanTTSWithDesc(cfg.BaseURL, cfg.Model, cfg.VoiceDescription), nil
			}
			return NewHerdsmanTTS(cfg.BaseURL, cfg.Model, cfg.Voice), nil
		default:
			return NewHerdsmanTTS(cfg.BaseURL, cfg.Model, cfg.Voice), nil
		}
	})
}

// SynthesizeWithMime 合成并返回音频和实际 MIME 类型（默认参数，兼容路径）。
func (h *HerdsmanTTS) SynthesizeWithMime(text string) ([]byte, string, error) {
	return h.SynthesizeWithParams(text, TTSParams{})
}

// SynthesizeWithParams 携带单次合成参数合成并返回音频和实际 MIME 类型。
// Speed>0 时请求体带 speed；Emotion/Style 非空时带 emotion/style（v4.3d）。
func (h *HerdsmanTTS) SynthesizeWithParams(text string, p TTSParams) ([]byte, string, error) {
	voice := h.resolveVoice()
	body := h.buildBodyParams(text, voice, p)
	jsonBody, _ := json.Marshal(body)

	audio, mime, err := h.requestAudio(jsonBody, voice)
	if err == nil {
		return audio, mime, nil
	}

	// 首选音色失败时（如配置了 Cherry 但服务端已不支持），用默认音色重试一次
	dv := defaultVoiceForModel(h.model)
	if voice != dv && dv != "" {
		defaultBody := h.buildBodyParams(text, dv, p)
		if db, mErr := json.Marshal(defaultBody); mErr == nil {
			if audio2, mime2, err2 := h.requestAudio(db, dv); err2 == nil {
				return audio2, mime2, nil
			}
		}
	}
	return nil, "", err
}

// buildBody 构造 /v1/audio/speech 请求体（无单次合成参数，兼容既有调用）。
func (h *HerdsmanTTS) buildBody(text, voice string) map[string]interface{} {
	return h.buildBodyParams(text, voice, TTSParams{})
}

// buildBodyParams 构造 /v1/audio/speech 请求体（含单次合成参数）。
// voiceclone 优先，其次 voicedesign，最后 customvoice/edge-tts 的 voice；
// Speed>0 时携带 speed、Emotion/Style 非空时携带 emotion/style。
func (h *HerdsmanTTS) buildBodyParams(text, voice string, p TTSParams) map[string]interface{} {
	body := map[string]interface{}{
		"model": h.model,
		"input": text,
	}
	switch {
	case h.refAudio != "":
		body["ref_audio"] = h.refAudio
		if h.refText != "" {
			body["ref_text"] = h.refText
		}
	case h.voiceDescription != "":
		body["voice_description"] = h.voiceDescription
	case voice != "":
		body["voice"] = voice
	}
	if p.Speed > 0 {
		body["speed"] = p.Speed
	}
	if p.Emotion != "" {
		body["emotion"] = p.Emotion
	}
	if p.Style != "" {
		body["style"] = p.Style
	}
	return body
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
	if h.voiceDescription != "" || h.refAudio != "" || isVoiceCloneModel(h.model) {
		return ""
	}
	voice := strings.TrimSpace(h.voice)
	if voice == "" {
		voice = defaultVoiceForModel(h.model)
	}

	// 仅对 qwen3-tts 系列模型动态查询支持音色（其他模型如 edge-tts 直接使用配置音色）
	l := strings.ToLower(h.model)
	if !isPresetVoiceModel(l) {
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
	if isVoiceCloneModel(h.model) || strings.TrimSpace(h.voiceDescription) != "" {
		return nil
	}
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
	case strings.Contains(l, "voiceclone"):
		return ""
	case strings.Contains(l, "voxcpm"):
		return ""
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

// isPresetVoiceModel 判断模型是否使用服务端 supported_speakers 音色列表。
func isPresetVoiceModel(model string) bool {
	l := strings.ToLower(model)
	return strings.Contains(l, "qwen3-tts") || strings.Contains(l, "customvoice")
}

// isVoiceCloneModel 判断是否为 qwen3-tts-voiceclone 克隆模式。
func isVoiceCloneModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "voiceclone")
}

// openAIEndpoint 将 OpenAI 兼容 base URL 与 API 路径拼接：
// baseURL 可能以 /v1 结尾（Herdsman/Ollama/xAI），也可能不带（DeepSeek），统一得到 .../v1<path>
func openAIEndpoint(baseURL, path string) string {
	return stripV1Prefix(baseURL) + "/v1" + path
}

func stripV1Prefix(baseURL string) string {
	return strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/v1")
}
