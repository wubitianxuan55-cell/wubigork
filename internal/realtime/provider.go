// Package realtime — 端到端实时语音 Realtime 档供应商 Seam（S0）。
//
// seam 三元组（定义/提供者/消费者，范式见 internal/asr/provider.go 与
// internal/tts/provider.go 的 Register/New/Kinds）：
//   - 定义：RealtimeSession 接口（Dial + SendAudio + Events + Close）
//   - 提供者：openai（OpenAI Realtime API WebSocket，openai.go init() 自注册）
//   - 消费者：S0 仅 app 层 initVoice 探测 + VoiceHealth 汇报（seam/优雅降级
//     验证，不启动会话、不接管音频路径）；S1 起接入端到端语音路径与配置落盘
//
// 纪律：kind 互斥注册（重复即 panic）；未知 kind New 报错（fail-closed，
// 不静默降级）；工厂只做构造与配置校验，不做网络 I/O（网络统一在 Dial）。
package realtime

import (
	"context"
	"fmt"
	"sort"
)

// 事件类型常量（Event.Type；值与 OpenAI Realtime API 服务端事件名一致）。
const (
	EventSessionCreated                = "session.created"
	EventSessionUpdated                = "session.updated"
	EventInputAudioBufferSpeechStarted = "input_audio_buffer.speech_started"
	EventInputAudioBufferSpeechStopped = "input_audio_buffer.speech_stopped"
	EventResponseAudioDelta            = "response.audio.delta"
	EventResponseAudioDone             = "response.audio.done"
	EventResponseTextDelta             = "response.text.delta"
	EventError                         = "error"
	EventUnknown                       = "unknown"
)

// Event 实时会话事件（服务端 JSON 事件的规范化载体）。
type Event struct {
	Type     string // 事件类型（上方 Event* 常量；未识别事件 = EventUnknown）
	DataJSON string // 原始服务端 JSON（未识别事件凭此不丢信息）
	AudioPCM []byte // response.audio.delta 解码出的 PCM16 音频（其他事件为 nil）
}

// RealtimeSession 端到端实时语音会话接口（seam 定义）。
type RealtimeSession interface {
	// Dial 建立连接并完成会话初始化（如 session.update）；ctx 控制握手取消/超时。
	Dial(ctx context.Context) error
	// SendAudio 发送一帧输入音频（PCM16 采样，采样率按提供者约定，openai 为 16k）。
	SendAudio(pcm []byte) error
	// Events 返回服务端事件通道；Dial 成功后开始投递，会话连接断开后关闭。
	Events() <-chan Event
	// Close 关闭会话（幂等；未 Dial 直接 Close 合法，探测场景）。
	Close() error
}

// Config Realtime 会话实例配置（注册表 New 入参）。
// 各 kind 按需读取字段：openai 用 APIKey（必填）+ BaseURL/Model/Voice/Instructions。
type Config struct {
	BaseURL      string // WebSocket 端点（openai 默认 wss://api.openai.com/v1/realtime）
	Model        string // 模型 ID（openai 默认 gpt-4o-realtime-preview）
	APIKey       string // Bearer 凭据（openai 必填，构造期校验，不做网络请求）
	Instructions string // 会话系统指令（session.update.instructions；空 = 不下发）
	Voice        string // 输出音色（openai 默认 alloy）
}

// SessionFactory 按实例配置构建 Realtime 会话（kind → 实例）。
type SessionFactory func(cfg Config) (RealtimeSession, error)

// sessionRegistry kind → 工厂注册表（互斥注册，重复即 panic）。
var sessionRegistry = map[string]SessionFactory{}

// Register 注册 Realtime 会话 kind（如 "openai"）。
// 供各实现 init() 自注册；kind 为空或重复注册直接 panic（编译期接线错误）。
func Register(kind string, factory SessionFactory) {
	if kind == "" {
		panic("realtime: session kind must not be empty")
	}
	if _, dup := sessionRegistry[kind]; dup {
		panic("realtime: duplicate session kind " + kind)
	}
	sessionRegistry[kind] = factory
}

// New 按 kind 经注册表构建 Realtime 会话实例；
// 未知 kind 返回错误（附已注册 kind 列表，fail-closed 不静默降级）。
func New(kind string, cfg Config) (RealtimeSession, error) {
	factory, ok := sessionRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("realtime: unknown session kind %q (registered: %v)", kind, Kinds())
	}
	s, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("realtime: factory %q returned nil session", kind)
	}
	return s, nil
}

// Kinds 返回已注册 Realtime 会话 kind 列表（排序，供诊断/校验）。
func Kinds() []string {
	out := make([]string, 0, len(sessionRegistry))
	for k := range sessionRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
