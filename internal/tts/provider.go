// Package tts — TTS 提供者 Seam（Step 3c OCR/ASR/TTS Provider Seam）。
//
// seam 三元组（定义/提供者/消费者，范式见 internal/gaea/provider/provider.go 与
// internal/ai/image_backend.go 的 Register/New/Kinds）：
//   - 定义：TTSProvider 接口（Name + Synthesize + SynthesizeWithMime）
//   - 提供者：edge（Edge 在线 TTS）、herdsman（/v1/audio/speech 兼容）、
//     xai（Grok TTS /v1/tts）、sapi（Windows SAPI），各自 init() 自注册，互斥注册
//   - 消费者：app 层 tts_handler.go / voice_handler.go 只依赖 TTSProvider 与注册表
//
// 纪律：kind 互斥注册（重复即 panic）；未知 kind New 报错（fail-closed，不静默降级）。
package tts

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
)

// TTSProvider TTS 合成提供者接口（seam 定义）。
// Synthesize 返回音频字节；SynthesizeWithMime 额外返回实际 MIME（如 audio/mp3 /
// audio/wav / audio/mpeg），空 MIME 由调用方按 audio/mp3 兜底（与历史行为一致）。
type TTSProvider interface {
	// Name 返回提供者 kind（"edge" / "herdsman" / "xai" / "sapi"）。
	Name() string
	// Synthesize 合成语音，返回音频字节。
	Synthesize(text string) ([]byte, error)
	// SynthesizeWithMime 合成语音并返回音频与 MIME 类型。
	SynthesizeWithMime(text string) ([]byte, string, error)
}

// TTSConfig TTS 提供者实例配置（注册表 New 入参）。
// 各 kind 按需读取字段：herdsman 用 BaseURL+Model+Voice/VoiceDescription/RefAudio；
// xai 用 BaseURL+Voice+GetToken；edge/sapi 无需配置。
type TTSConfig struct {
	BaseURL          string                 // OpenAI 兼容引擎地址（herdsman/xai）
	Model            string                 // 模型 ID（herdsman）
	Voice            string                 // 音色（herdsman customvoice / xai）
	VoiceDescription string                 // voicedesign 模式音色描述
	RefAudio         string                 // voiceclone 模式参考音频（路径/URL/data URI）
	RefText          string                 // voiceclone 模式参考文本（可选）
	GetToken         func() (string, error) // xai Bearer token（复用 ai.Client.GetToken）
	HTTPClient       *http.Client           // 可空：xai 默认 30s 客户端
}

// TTSProviderFactory 按实例配置构建 TTS 提供者（kind → 实例）。
type TTSProviderFactory func(cfg TTSConfig) (TTSProvider, error)

// ttsProviderRegistry kind → 工厂注册表（互斥注册，重复即 panic）。
var ttsProviderRegistry = map[string]TTSProviderFactory{}

// RegisterTTSProvider 注册 TTS 提供者 kind（如 "edge" / "herdsman" / "xai" / "sapi"）。
// 供各实现 init() 自注册；kind 为空或重复注册直接 panic（编译期接线错误）。
func RegisterTTSProvider(kind string, factory TTSProviderFactory) {
	if kind == "" {
		panic("tts: provider kind must not be empty")
	}
	if _, dup := ttsProviderRegistry[kind]; dup {
		panic("tts: duplicate provider kind " + kind)
	}
	ttsProviderRegistry[kind] = factory
}

// NewTTSProvider 按 kind 经注册表构建 TTS 提供者实例；
// 未知 kind 返回错误（附已注册 kind 列表，fail-closed 不静默降级）。
func NewTTSProvider(kind string, cfg TTSConfig) (TTSProvider, error) {
	factory, ok := ttsProviderRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("tts: unknown provider kind %q (registered: %v)", kind, TTSProviderKinds())
	}
	p, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("tts: factory %q returned nil provider", kind)
	}
	return p, nil
}

// TTSProviderKinds 返回已注册 TTS 提供者 kind 列表（排序，供诊断/校验）。
func TTSProviderKinds() []string {
	out := make([]string, 0, len(ttsProviderRegistry))
	for k := range ttsProviderRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TTSChain 注册表驱动的 TTS 回退链：按序尝试各提供者，返回第一个成功的
// 音频、MIME 与提供者名（流式引擎据此钉住当前引擎，失败再重新探测整链）。
// 语义与历史 SynthesizerChain 一致（首个成功即赢，全败报错）。
type TTSChain struct {
	providers []TTSProvider
}

// NewTTSChain 创建提供者回退链（按优先级排列）。
func NewTTSChain(providers ...TTSProvider) *TTSChain {
	return &TTSChain{providers: providers}
}

// Synthesize 依次尝试各提供者，返回第一个成功的 (音频, MIME, 提供者名)。
func (c *TTSChain) Synthesize(text string) ([]byte, string, string, error) {
	if len(c.providers) == 0 {
		return nil, "", "", fmt.Errorf("tts: 无可用 TTS 提供者")
	}
	for _, p := range c.providers {
		audio, mime, err := p.SynthesizeWithMime(text)
		if err == nil && len(audio) > 0 {
			return audio, mime, p.Name(), nil
		}
		slog.Warn("TTS 提供者失败，尝试下一个", "provider", p.Name(), "error", err)
	}
	return nil, "", "", fmt.Errorf("所有 TTS 引擎均失败")
}

// ProviderByName 按提供者名返回链中实例（流式引擎钉住后复用；未命中返回 nil）。
func (c *TTSChain) ProviderByName(name string) TTSProvider {
	for _, p := range c.providers {
		if p.Name() == name {
			return p
		}
	}
	return nil
}
