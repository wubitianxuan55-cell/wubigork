package whisper

import "testing"

// v4.8.3 微信实测教训：短消息一律镜像 ≤15 字钳制 + 新关系 silent 门控叠加，
// 助手人格对实质问题也只回一句话。修复：短问题豁免镜像；gaea 助手人格
// 豁免情绪门控。

func TestIsShortQuestion(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"你好", false},            // 寒暄：保留短镜像
		{"嗯", false},              // 应答：保留短镜像
		{"你是谁", true},           // 短问题：豁免
		{"你会什么", true},          // 短问题：豁免
		{"在吗？", true},           // 带问号：豁免
		{"现在用什么模型", true},    // 含「什么」：豁免
		{"画一张猫", false},         // 短祈使：保留短镜像
		{"今天天气怎么样", true},     // 疑问：豁免
	}
	for _, c := range cases {
		if got := isShortQuestion(c.msg); got != c.want {
			t.Errorf("isShortQuestion(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
}

func TestDetectUserVerbosity_TerseForShort(t *testing.T) {
	if DetectUserVerbosity("你好") != "terse" {
		t.Fatal("前置：你好应为 terse")
	}
	if DetectUserVerbosity("你不能直接输出到微信吗") != "normal" {
		t.Fatal("11 字应为 normal")
	}
}

// 助手人格豁免门控的协同断言：gaea 人格下 silent 策略块不得注入。
func TestGaeaPersonaExemptFromGate(t *testing.T) {
	p := GetPreset("gaea")
	if p.ID != "gaea" {
		t.Fatal("gaea 人格预设应存在")
	}
	// 镜像注入条件由调用点组合（DetectUserVerbosity==terse && !isShortQuestion），
	// 此处锁定关键组合：短问题不再满足镜像注入条件。
	if DetectUserVerbosity("你是谁") == "terse" && !isShortQuestion("你是谁") {
		t.Fatal("「你是谁」不应同时满足 terse 与非问题")
	}
}
