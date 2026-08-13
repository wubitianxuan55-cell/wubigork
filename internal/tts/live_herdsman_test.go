package tts

import (
	"os"
	"testing"
)

// TestLiveVoxCPM2 真实调用本机 Herdsman /v1/audio/speech 的 voxcpm2。
// 仅在 HERDSMAN_LIVE=1 时运行。
func TestLiveVoxCPM2(t *testing.T) {
	if os.Getenv("HERDSMAN_LIVE") != "1" {
		t.Skip("HERDSMAN_LIVE=1 时运行真实 voxcpm2 TTS 验证")
	}
	client := NewHerdsmanTTS("http://localhost:8080/v1", "voxcpm2", "")
	audio, mime, err := client.SynthesizeWithMime("这是 voxcpm2 新模型对比测试的文本")
	if err != nil {
		t.Fatalf("SynthesizeWithMime: %v", err)
	}
	if len(audio) < 1000 {
		t.Fatalf("音频过短: %d bytes", len(audio))
	}
	if mime == "" {
		t.Error("mime 为空")
	}
	t.Logf("voxcpm2 audio bytes=%d mime=%s", len(audio), mime)
}
