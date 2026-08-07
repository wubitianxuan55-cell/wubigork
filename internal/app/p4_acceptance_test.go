package app

import (
	"strings"
	"testing"
)

// 完成判据：轻语（右脑）记住的甲方偏好，在方案写作时自动可用。
func TestP4AcceptanceWhisperMemoryFeedsProposal(t *testing.T) {
	bs := &BrainStore{
		main:  &fakeMainBrain{rows: map[string]string{}},
		left:  &fakeLeftBrain{},
		right: &fakeRightBrain{rows: map[string]string{"甲方A|偏好": "保守报价"}},
	}
	mats := buildBrainMaterials(bs, "土壤修复标书", "报价")
	if len(mats) == 0 || !strings.Contains(mats[0], "保守报价") {
		t.Fatalf("判据失败：右脑材料未注入 = %+v", mats)
	}
	// 主脑一句话派发（P3 能力）仍可用：标书 → office。
	module, intent := classifyMainBrainIntent("帮我把标书写了")
	if module != "office" || intent != "create" {
		t.Fatalf("主脑派发 = (%q,%q)", module, intent)
	}
}
