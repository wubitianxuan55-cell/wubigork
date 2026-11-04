//go:build !windows || !amd64 || !cgo

// engine_other.go — 非 Windows/AMD64 stub：保持包可编译（CI linux job），引擎返回不支持错误。
package native

import "time"

// Engine stub 句柄。
type Engine struct{}

// New stub。
func New(cfg Config) (*Engine, error) { return nil, ErrNotSupported }

// Close stub。
func (e *Engine) Close() {}

// Transcribe stub。
func (e *Engine) Transcribe(pcm []float32, sampleRate int) (string, error) {
	return "", ErrNotSupported
}

// EndOfTurnProbability stub。
func (e *Engine) EndOfTurnProbability(pcm []float32, sampleRate int) (float32, error) {
	return 0, ErrNotSupported
}

// EndOfTurnTiming stub。
func (e *Engine) EndOfTurnTiming(pcm []float32, sampleRate int) (float32, time.Duration, error) {
	return 0, 0, ErrNotSupported
}

// VADSegments stub。
func (e *Engine) VADSegments(pcm []float32) [][]float32 { return nil }

// OrtDLLPath stub。
func OrtDLLPath() string { return "" }
