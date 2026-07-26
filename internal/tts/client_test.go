package tts

import (
	"fmt"
	"os"
	"testing"
)

func TestEdgeTTS(t *testing.T) {
	e := NewEdgeTTS()
	audio, err := e.Synthesize("你好世界，这是一段测试文本。")
	if err != nil {
		t.Fatalf("EdgeTTS 失败: %v", err)
	}
	t.Logf("成功: %d bytes", len(audio))
	if len(audio) < 500 {
		t.Errorf("音频太小: %d bytes", len(audio))
	}
}

func TestSplitSentences(t *testing.T) {
	text := "你好。这是第一句！第二句呢？很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长很长的一段话，需要拆分，因为太长了；继续测试。最后一句。"
	sentences := SplitSentences(text)
	fmt.Printf("拆分为 %d 句:\n", len(sentences))
	for i, s := range sentences {
		fmt.Printf("  [%d] %s\n", i, s)
	}
	if len(sentences) < 4 {
		t.Errorf("句子数太少: %d", len(sentences))
	}
}

func TestSynthesizeStreaming(t *testing.T) {
	binaryPath := `D:\AI\voxcpm-cpp\build\examples\voxcpm_tts.exe`
	modelPath := `D:\AI\voxcpm-cpp\models\voxcpm-0.5b-q4_k-audiovae-f16.gguf`

	if _, err := os.Stat(binaryPath); err != nil {
		t.Skipf("voxcpm_tts 未找到: %v", err)
	}
	if _, err := os.Stat(modelPath); err != nil {
		t.Skipf("模型未找到: %v", err)
	}

	client := NewClient(binaryPath, modelPath, "cpu")

	// 模拟流式场景：多句话
	text := "这是流式朗读测试的第一句话。这是第二句，验证连续合成是否正常。最后一句，测试完成。"
	sentences := SplitSentences(text)
	t.Logf("共 %d 句", len(sentences))

	for i, sentence := range sentences {
		audio, err := client.Synthesize(sentence)
		if err != nil {
			t.Fatalf("第 %d 句合成失败: %v", i, err)
		}
		t.Logf("第 %d 句: %d bytes (文本: %s)", i, len(audio), sentence)
		if len(audio) < 100 {
			t.Errorf("第 %d 句音频太小: %d bytes", i, len(audio))
		}
	}
}
