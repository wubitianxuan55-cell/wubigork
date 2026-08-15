package voice

import (
	"errors"
	"testing"

	"github.com/gaea/gaea/internal/asr"
)

// mockASRProvider 测试用 ASR 提供者：实现 asr.ASRProvider 接口，验证
// voice.Manager 以接口注入消费（不依赖具体 herdsman 实现，seam 纪律）。
type mockASRProvider struct {
	gotBase64 string
	gotMime   string
}

func (m *mockASRProvider) Name() string { return "mock" }
func (m *mockASRProvider) TranscribeBase64(audioBase64, mimeType string) (*asr.TranscriptionResult, error) {
	m.gotBase64 = audioBase64
	m.gotMime = mimeType
	return &asr.TranscriptionResult{Text: "接口注入识别结果"}, nil
}
func (m *mockASRProvider) TranscribeBytes(audioData []byte, filename string) (*asr.TranscriptionResult, error) {
	return nil, errors.New("not used")
}

// TestManager_ASRInterfaceInjection 接口注入：SetASRProvider 接受任何
// asr.ASRProvider 实现（不止 herdsman），transcribe 经接口调用并返回结果。
func TestManager_ASRInterfaceInjection(t *testing.T) {
	m, _ := newTestManager()
	if m.ASRReady() {
		t.Fatal("未注入时应 not ready")
	}

	mock := &mockASRProvider{}
	m.SetASRProvider(mock)
	if !m.ASRReady() {
		t.Fatal("注入后应 ready")
	}

	text, err := m.transcribe([]byte{0x00, 0x01, 0x02})
	if err != nil {
		t.Fatalf("transcribe: %v", err)
	}
	if text != "接口注入识别结果" {
		t.Errorf("text = %q, want 接口注入识别结果", text)
	}
	if mock.gotMime != "audio/wav" {
		t.Errorf("mime = %q, want audio/wav", mock.gotMime)
	}
	if mock.gotBase64 == "" {
		t.Error("应传入 base64 音频")
	}
}

// TestManager_ASRProviderCompileAssertion 编译期断言：herdsman 满足接口。
func TestManager_ASRProviderCompileAssertion(t *testing.T) {
	var _ asr.ASRProvider = asr.NewHerdsmanASR("http://x/v1", "whisper-base")
}
