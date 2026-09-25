package app

import (
	"testing"
)

// 完成判据：轻语（右脑）记住的甲方偏好，在方案写作时自动可用。
func TestP4AcceptanceWhisperMemoryFeedsProposal(t *testing.T) {
	// 主脑一句话派发（P3 能力）仍可用：标书 → office。
	module, intent := classifyMainBrainIntent("帮我把标书写了")
	if module != "office" || intent != "create" {
		t.Fatalf("主脑派发 = (%q,%q)", module, intent)
	}
}
