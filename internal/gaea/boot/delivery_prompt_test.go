package boot

// delivery_prompt_test.go — 交付与收尾纪律段的契约测试。
// 目的：钉死三类关键约束不被人改回去（链接禁代码块、只承诺已发生的事、
// 结束前四问自检），并保证文案不含模板化客套。

import (
	"strings"
	"testing"
)

func TestDeliveryPromptSectionKeepsLinkRule(t *testing.T) {
	sec := DeliveryPromptSection()
	if !strings.Contains(sec, "不要加反引号") || !strings.Contains(sec, "不要放进代码块") {
		t.Fatalf("交付段缺少链接渲染约束（反引号/代码块），实际文案：%q", sec)
	}
	if !strings.Contains(sec, "[文件名](路径)") {
		t.Error("交付段缺少链接格式示例 [文件名](路径)")
	}
}

func TestDeliveryPromptSectionKeepsPromiseRule(t *testing.T) {
	sec := DeliveryPromptSection()
	if !strings.Contains(sec, "只有工具返回成功") {
		t.Error("交付段缺少「只承诺已发生的事」约束")
	}
	if !strings.Contains(sec, "已生成") || !strings.Contains(sec, "已打开") {
		t.Error("交付段应显式点名「已生成/已打开」这类易夸大表述")
	}
}

func TestDeliveryPromptSectionKeepsChecklist(t *testing.T) {
	sec := DeliveryPromptSection()
	if !strings.Contains(sec, "结束前自检") {
		t.Error("交付段缺少结束前自检清单")
	}
	// 模板化客套：段内只允许以否定形式提及，不得作为建议使用。
	if strings.Contains(sec, "希望这对你有帮助」」") {
		t.Error("文案异常：出现重复引号")
	}
	if !strings.Contains(sec, "不使用") {
		t.Error("交付段缺少「不使用模板化结束语」的反面约束")
	}
}

func TestDeliveryPromptSectionKeepsTimingRules(t *testing.T) {
	// 时序禁则（batch2 B1-①，灵犀 deliver_result 前置/后置禁则蒸馏）：
	// 前置=链接前不铺列表；后置=链接给出后不复述同一路径。
	sec := DeliveryPromptSection()
	if !strings.Contains(sec, "给出链接前不要先铺有序/无序列表") {
		t.Fatalf("交付段缺前置禁则（链接前不铺列表），实际文案：%q", sec)
	}
	if !strings.Contains(sec, "不要再以任何形式重复同一交付路径") {
		t.Fatalf("交付段缺后置禁则（链接后不复述路径），实际文案：%q", sec)
	}
}

func TestDeliveryPromptSectionNoEmoji(t *testing.T) {
	// 与主提示词一致：办公提示词不带表情符号（灵犀 plan 段同规约）。
	for _, r := range DeliveryPromptSection() {
		if r >= 0x1F300 && r <= 0x1FAFF {
			t.Fatalf("交付段不应包含表情符号：%q", r)
		}
	}
}
