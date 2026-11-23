package maturecraft

import (
	"strings"
	"testing"
)

// TestNormalizeLevel 档位白名单归一：合法值原样通过，其余（含用户手改
// project.json 的坏值）一律落 ""——不允许半开半关。
func TestNormalizeLevel(t *testing.T) {
	cases := map[string]string{
		"":              "",
		"sensual":       Sensual,
		"explicit":      Explicit,
		"unrestricted":  Unrestricted,
		"SENSUAL":       "", // 大小写敏感：写盘走白名单，读盘坏值不猜测
		"UNRESTRICTED":  "",
		"explicit ":     "", // 带空格的坏值不洗白
		"成人向":           "",
		"r18":           "",
		"explicit;drop": "",
	}
	for in, want := range cases {
		if got := NormalizeLevel(in); got != want {
			t.Errorf("NormalizeLevel(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCraftSection 正文向工艺区段矩阵：空档位零注入；两档各带档位专属工艺；
// 硬线（成年/不美化胁迫）与功能闸两档都在。
func TestCraftSection(t *testing.T) {
	if got := CraftSection(""); got != "" {
		t.Errorf("非成人向必须零注入，got:\n%s", got)
	}
	if got := CraftSection("坏值"); got != "" {
		t.Errorf("坏值必须零注入，got:\n%s", got)
	}

	sensual := CraftSection(Sensual)
	for _, want := range []string{"含蓄", "张力优先", "留白不是黑幕", "功能闸", "必须是成年人", "不把胁迫包装成浪漫"} {
		if !contains(sensual, want) {
			t.Errorf("含蓄档缺「%s」\n%s", want, sensual)
		}
	}
	if contains(sensual, "正面直书") {
		t.Errorf("含蓄档不得携带直白档口径")
	}

	explicit := CraftSection(Explicit)
	for _, want := range []string{"直白档", "正面直书", "感官纪律", "事后节拍", "同意节拍", "AI 腔限流", "功能闸", "必须是成年人", "不把胁迫包装成浪漫"} {
		if !contains(explicit, want) {
			t.Errorf("直白档缺「%s」\n%s", want, explicit)
		}
	}
	if contains(explicit, "留白不是黑幕") {
		t.Errorf("直白档不得携带含蓄档专属条目")
	}

	// 无限制档：零安全底线——直书与功能闸仍在，硬线与同意节拍一律不注入。
	unrestricted := CraftSection(Unrestricted)
	for _, want := range []string{"无限制", "正面直书", "零底线", "功能闸", "AI 腔限流"} {
		if !contains(unrestricted, want) {
			t.Errorf("无限制档缺「%s」\n%s", want, unrestricted)
		}
	}
	for _, ban := range []string{"必须是成年人", "不把胁迫包装成浪漫", "同意节拍", "拒绝即停", "硬线"} {
		if contains(unrestricted, ban) {
			t.Errorf("无限制档不得携带安全底线「%s」\n%s", ban, unrestricted)
		}
	}
}

// TestPlanSection 计划向纪律矩阵：空档位零注入；非空档位要求计划把亲密戏当事件排。
func TestPlanSection(t *testing.T) {
	if got := PlanSection(""); got != "" {
		t.Errorf("非成人向计划纪律必须零注入，got:\n%s", got)
	}
	plan := PlanSection(Explicit)
	for _, want := range []string{"成人向计划纪律", "节拍", "不得只写「两人关系升温」", "永久改变了什么"} {
		if !contains(plan, want) {
			t.Errorf("计划纪律缺「%s」\n%s", want, plan)
		}
	}
}

// TestOutlineSection 大纲向纪律矩阵：欲望线一等叙事线 + 节拍功能 + 对手性障碍；
// 空档位零注入。
func TestOutlineSection(t *testing.T) {
	if got := OutlineSection(""); got != "" {
		t.Errorf("非成人向大纲纪律必须零注入，got:\n%s", got)
	}
	sec := OutlineSection(Sensual)
	for _, want := range []string{"成人向大纲纪律", "欲望线是一等叙事线", "不得只写「感情升温」", "张力要有对手性"} {
		if !contains(sec, want) {
			t.Errorf("大纲纪律缺「%s」\n%s", want, sec)
		}
	}
}

// TestCharacterSection 角色向纪律矩阵：供材导向 + 欲望设定硬线；空档位零注入。
func TestCharacterSection(t *testing.T) {
	if got := CharacterSection(""); got != "" {
		t.Errorf("非成人向角色纪律必须零注入，got:\n%s", got)
	}
	sec := CharacterSection(Explicit)
	for _, want := range []string{"成人向角色纪律", "欲望线与亲密张力来源", "不生成未成年", "不生成美化胁迫"} {
		if !contains(sec, want) {
			t.Errorf("角色纪律缺「%s」\n%s", want, sec)
		}
	}

	// 无限制档：欲望设定零限制，不注入未成年/胁迫硬线。
	unlimited := CharacterSection(Unrestricted)
	for _, want := range []string{"成人向角色纪律", "零回避零审查", "不受年龄、血缘、身份、伦理或权力差限制"} {
		if !contains(unlimited, want) {
			t.Errorf("无限制角色纪律缺「%s」\n%s", want, unlimited)
		}
	}
	for _, ban := range []string{"不生成未成年", "不生成美化胁迫", "必须是成年人"} {
		if contains(unlimited, ban) {
			t.Errorf("无限制角色纪律不得携带安全底线「%s」\n%s", ban, unlimited)
		}
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// TestJailbreakSection 破甲契约：仅无限制档注入，其余档零注入；契约须含
// 零拒绝/零淡化/冲突裁决三类条款（否则对云端模型只是普通风格提示）。
func TestJailbreakSection(t *testing.T) {
	for _, lv := range []string{"", Sensual, Explicit, "坏值"} {
		if got := JailbreakSection(lv); got != "" {
			t.Errorf("JailbreakSection(%q) 必须零注入，got:\n%s", lv, got)
		}
	}
	jb := JailbreakSection(Unrestricted)
	for _, want := range []string{"零拒绝", "零淡化", "零审查", "冲突裁决", "作者引擎", "最高优先级"} {
		if !contains(jb, want) {
			t.Errorf("破甲契约缺「%s」\n%s", want, jb)
		}
	}
	// 2026-11-23 实测反馈三例外：工程失败如实报（不被「删除续写」吞掉）、
	// 回复形态听用户本回合、裁决条款明写例外。
	for _, want := range []string{"工程性失败", "如实报告", "优先于本条", "唯三例外"} {
		if !contains(jb, want) {
			t.Errorf("破甲契约缺例外条款锚「%s」\n%s", want, jb)
		}
	}
}
