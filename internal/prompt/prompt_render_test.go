// RenderPlaceholders 表驱动测试（规格 进度计划/gaea-prompt-workshop-t6-20260916.md
// §3.4 五态：命中 / 缺失保留+记名 / 单层花括号不动 / 同名多次 / 空 vars）+
// 边界补充（未闭合 {{ / 缺失去重 / 替换值不再扫描）。花括号语义的「为什么」
// 见 docs/distill/06-prompt-workshop.md §3.2/§9.3（MuMu str.format 炸链路教训）。
package prompt

import (
	"reflect"
	"testing"
)

// TestRenderPlaceholders 五态主表：断言渲染结果与未解析名切片（恒非 nil）。
func TestRenderPlaceholders(t *testing.T) {
	cases := []struct {
		name        string
		in          string
		vars        map[string]string
		want        string
		wantMissing []string
	}{
		{
			name:        "命中替换",
			in:          "目标 {{word_count}} 字",
			vars:        map[string]string{"word_count": "3000"},
			want:        "目标 3000 字",
			wantMissing: []string{},
		},
		{
			name:        "缺失变量保留原文并记名",
			in:          "目标 {{missing}} 字",
			vars:        map[string]string{"word_count": "3000"},
			want:        "目标 {{missing}} 字",
			wantMissing: []string{"missing"},
		},
		{
			name:        "单层花括号不动（JSON 字面量与旧 {word_count} 语法）",
			in:          `输出 {"summary":"x"} 形如 {word_count} 的单层花括号`,
			vars:        map[string]string{"word_count": "1", "summary": "S"},
			want:        `输出 {"summary":"x"} 形如 {word_count} 的单层花括号`,
			wantMissing: []string{},
		},
		{
			name:        "同名多次出现全部替换",
			in:          "不少于{{n}}字，输出要求不少于{{n}}字",
			vars:        map[string]string{"n": "5000"},
			want:        "不少于5000字，输出要求不少于5000字",
			wantMissing: []string{},
		},
		{
			name:        "vars 为 nil 全部保留",
			in:          "{{a}} 和 {{b}}",
			vars:        nil,
			want:        "{{a}} 和 {{b}}",
			wantMissing: []string{"a", "b"},
		},
		{
			name:        "vars 为空 map 全部保留",
			in:          "{{a}}",
			vars:        map[string]string{},
			want:        "{{a}}",
			wantMissing: []string{"a"},
		},
	}
	for _, c := range cases {
		got, missing := RenderPlaceholders(c.in, c.vars)
		if got != c.want {
			t.Fatalf("%s: 渲染结果\n got  %q\n want %q", c.name, got, c.want)
		}
		if missing == nil {
			t.Fatalf("%s: 未解析名切片应恒非 nil（空切片而非 nil）", c.name)
		}
		if !reflect.DeepEqual(missing, c.wantMissing) {
			t.Fatalf("%s: 未解析名 got %v want %v", c.name, missing, c.wantMissing)
		}
	}
}

// TestRenderPlaceholdersEdges 边界补充：未闭合 {{ 保守收尾 / 缺失同名去重记一次
// （保首现序）/ 替换值含花括号不二次扫描 / 空占位符 {{}} / 尾段无占位符 / 空串。
func TestRenderPlaceholdersEdges(t *testing.T) {
	// 未闭合 {{：其后无 }}——整段照抄，不计未解析名（平衡问题归保存校验管）。
	got, missing := RenderPlaceholders("前文 {{unclosed 尾巴", map[string]string{"x": "1"})
	if got != "前文 {{unclosed 尾巴" || len(missing) != 0 {
		t.Fatalf("未闭合 {{ 应整段照抄且不记名: got %q missing %v", got, missing)
	}
	// 同名多次缺失只记一次，且保首次出现顺序。
	got, missing = RenderPlaceholders("{{b}} {{a}} {{b}} {{a}}", nil)
	if got != "{{b}} {{a}} {{b}} {{a}}" {
		t.Fatalf("缺失应保留原文: got %q", got)
	}
	if !reflect.DeepEqual(missing, []string{"b", "a"}) {
		t.Fatalf("未解析名应按首现序去重: got %v", missing)
	}
	// 命中的替换值含 {{}} 不再二次扫描（值是数据不是模板）。
	got, missing = RenderPlaceholders("值={{v}}", map[string]string{"v": "{{v2}}"})
	if got != "值={{v2}}" || len(missing) != 0 {
		t.Fatalf("替换值不应被二次扫描: got %q missing %v", got, missing)
	}
	// 双层与单层混排：只有双层是占位符。
	got, missing = RenderPlaceholders("{a} {{a}} {a}", map[string]string{"a": "V"})
	if got != "{a} V {a}" || len(missing) != 0 {
		t.Fatalf("单层 {a} 应原样不动: got %q missing %v", got, missing)
	}
	// 空占位符 {{}}：名字为空串，未命中也保留。
	got, missing = RenderPlaceholders("x{{}}y", nil)
	if got != "x{{}}y" || !reflect.DeepEqual(missing, []string{""}) {
		t.Fatalf("空占位符应保留并记空名: got %q missing %v", got, missing)
	}
	// 命中空值：ok 语义按存在性，替换为空串（不是缺失）。
	got, missing = RenderPlaceholders("x{{a}}y", map[string]string{"a": ""})
	if got != "xy" || len(missing) != 0 {
		t.Fatalf("空值变量应替换为空串: got %q missing %v", got, missing)
	}
	// 空串与无占位符。
	got, missing = RenderPlaceholders("", nil)
	if got != "" || len(missing) != 0 {
		t.Fatalf("空串应原样返回: got %q missing %v", got, missing)
	}
	got, missing = RenderPlaceholders("纯正文无占位", map[string]string{"a": "V"})
	if got != "纯正文无占位" || len(missing) != 0 {
		t.Fatalf("无占位符应原样返回: got %q missing %v", got, missing)
	}
}
