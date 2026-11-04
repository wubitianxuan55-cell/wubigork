// Package native — 语音管线原生引擎底座（VAD/EOT/STT 进程内推理）。
//
// Windows/AMD64 走 sherpa-onnx(链接 DLL) + onnxruntime_go(运行时加载同一只
// onnxruntime.dll，LoadLibrary 按路径去重后共享实例)；其余平台返回 stub 错误。
// 模型布局（C:\AI\models\voice\）：silero_vad.onnx / smart-turn-v3.2-cpu.onnx /
// model.int8.onnx(paraformer int8) + tokens.txt。
package native

import "errors"

// Config 引擎装配参数。模型路径为空字符串时跳过对应引擎（按需装配）。
type Config struct {
	// ORTRuntimePath 指向 onnxruntime.dll；空 = 按默认搜索序（exe 目录→PATH）。
	// 必须与 sherpa-onnx-c-api.dll 链接的是同一只 DLL（版本配对 ORT 1.28.x）。
	ORTRuntimePath       string
	SileroModelPath      string
	SmartTurnModelPath   string
	ParaformerModelPath  string
	ParaformerTokensPath string
	// NumThreads EOT/STT 推理线程数；0 = 2（对齐 sidecar smart_turn_cpu_count=2）。
	NumThreads int
}

// ErrNotSupported 非 Windows/AMD64 平台的 stub 返回值。
var ErrNotSupported = errors.New("voice native engine 仅支持 windows/amd64")

// SampleRate 管线工作采样率：16 kHz 单声道 PCM16。
const SampleRate = 16000
