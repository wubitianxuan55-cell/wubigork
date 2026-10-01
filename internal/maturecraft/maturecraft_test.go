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
		"SENSUAL":       "", // 大小写敏感：写盘走白名单，读盘坏值不猜测
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
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
