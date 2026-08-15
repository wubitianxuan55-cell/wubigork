// Package asr — ASR 提供者 Seam（Step 3c OCR/ASR/TTS Provider Seam）。
//
// seam 三元组（定义/提供者/消费者，范式见 internal/ai/image_backend.go 的
// Register/New/Kinds）：
//   - 定义：ASRProvider 接口（Name + TranscribeBase64 + TranscribeBytes）
//   - 提供者：herdsman（当前唯一实现，init() 自注册）
//   - 消费者：voice.Manager（SetASRProvider 接口注入，不再依赖具体类型）
//
// 纪律：kind 互斥注册（重复即 panic）；未知 kind New 报错（fail-closed）。
package asr

import (
	"fmt"
	"sort"
)

// ASRProvider 语音识别提供者接口（seam 定义）。
type ASRProvider interface {
	// Name 返回提供者 kind（如 "herdsman"）。
	Name() string
	// TranscribeBase64 通过 base64 编码的音频数据进行识别。
	TranscribeBase64(audioBase64, mimeType string) (*TranscriptionResult, error)
	// TranscribeBytes 通过原始音频字节进行识别（multipart/form-data 上传）。
	TranscribeBytes(audioData []byte, filename string) (*TranscriptionResult, error)
}

// ASRConfig ASR 提供者实例配置（注册表 New 入参）。
type ASRConfig struct {
	BaseURL string // OpenAI 兼容引擎地址（/v1/audio/transcriptions）
	Model   string // 模型 ID（whisper-base / sherpa-onnx-streaming-zipformer-zh-14m / funasr）
}

// ASRProviderFactory 按实例配置构建 ASR 提供者（kind → 实例）。
type ASRProviderFactory func(cfg ASRConfig) (ASRProvider, error)

// asrProviderRegistry kind → 工厂注册表（互斥注册，重复即 panic）。
var asrProviderRegistry = map[string]ASRProviderFactory{}

// RegisterASRProvider 注册 ASR 提供者 kind（如 "herdsman"）。
// 供各实现 init() 自注册；kind 为空或重复注册直接 panic（编译期接线错误）。
func RegisterASRProvider(kind string, factory ASRProviderFactory) {
	if kind == "" {
		panic("asr: provider kind must not be empty")
	}
	if _, dup := asrProviderRegistry[kind]; dup {
		panic("asr: duplicate provider kind " + kind)
	}
	asrProviderRegistry[kind] = factory
}

// NewASRProvider 按 kind 经注册表构建 ASR 提供者实例；
// 未知 kind 返回错误（附已注册 kind 列表，fail-closed 不静默降级）。
func NewASRProvider(kind string, cfg ASRConfig) (ASRProvider, error) {
	factory, ok := asrProviderRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("asr: unknown provider kind %q (registered: %v)", kind, ASRProviderKinds())
	}
	p, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("asr: factory %q returned nil provider", kind)
	}
	return p, nil
}

// ASRProviderKinds 返回已注册 ASR 提供者 kind 列表（排序，供诊断/校验）。
func ASRProviderKinds() []string {
	out := make([]string, 0, len(asrProviderRegistry))
	for k := range asrProviderRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
