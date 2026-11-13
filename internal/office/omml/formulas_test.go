// formulas_test.go — FormulasOf 键规则测试：d:/i: 前缀、tag 剥离、去重、
// latex 字段、无 $ 文本跳过、扫描语义（$$ 优先、(?!\$) 前瞻等价）。
package omml

import (
	"strings"
	"testing"
)

// TestFormulasOfKeysAndSources 键名与 source 规则（对齐上游 formulasOf）。
func TestFormulasOfKeysAndSources(t *testing.T) {
	ops := []any{
		map[string]any{"op": "set_text", "text": "质能 $E=mc^2$ 与行间 $$\\int_0^1 x\\,dx$$ 混排"},
		map[string]any{"op": "set_text", "text": "带编号 $$a+b \\tag{3}$$"},
		map[string]any{"latex": "$x+y$"},
		map[string]any{"op": "set_text", "text": "无公式"},
		map[string]any{"nested": []any{map[string]any{"deep": "$x_1$"}}},
		3.14, true, nil,
	}
	got := FormulasOf(ops, nil)
	// 行内 → i: 前缀。
	if _, ok := got["i:E=mc^2"]; !ok {
		t.Fatalf("缺 i:E=mc^2：%v", keysOf(got))
	}
	// 行间 → d: 前缀，source 原样（\tag 在键里不存在）。
	if om, ok := got["d:\\int_0^1 x\\,dx"]; !ok || !strings.Contains(om, "<m:nary>") {
		t.Fatalf("缺 d:\\int_0^1 x\\,dx 或值非 nary：%v", keysOf(got))
	}
	// tag 剥离进 source；OMML 里不得带 tag。
	if om, ok := got["d:a+b"]; !ok || strings.Contains(om, "tag") {
		t.Fatalf("tag 应剥离进 source（键 d:a+b）且不入 OMML：%v", keysOf(got))
	}
	if _, ok := got["d:a+b \\tag{3}"]; ok {
		t.Fatal("tag 不得残留在键名里")
	}
	// latex 字段：剥首尾 $ 后按 display 入 d: 键。
	if _, ok := got["d:x+y"]; !ok {
		t.Fatalf("latex 字段应入 d:x+y：%v", keysOf(got))
	}
	// 嵌套结构下钻。
	if _, ok := got["i:x_1"]; !ok {
		t.Fatalf("嵌套文本应下钻提取 i:x_1：%v", keysOf(got))
	}
	if len(got) != 5 {
		t.Fatalf("应恰有 5 条：%v", keysOf(got))
	}
}

// TestFormulasOfSkips 无公式文本不产生条目；转换失败的公式不进 map。
func TestFormulasOfSkips(t *testing.T) {
	got := FormulasOf([]any{"纯文本", map[string]any{"k": 1.0, "s": "没有分隔符"}, "$\\unknowncmd$"}, nil)
	if len(got) != 0 {
		t.Fatalf("不应产生条目：%v", got)
	}
}

// TestFormulasOfDedup 同键只留一条（同一 source 同一转换，先到先得）。
func TestFormulasOfDedup(t *testing.T) {
	got := FormulasOf([]any{"$a$ 再来一次 $a$"}, nil)
	if len(got) != 1 {
		t.Fatalf("重复公式应去重：%v", got)
	}
	if _, ok := got["i:a"]; !ok {
		t.Fatalf("键应为 i:a：%v", got)
	}
	// found 复用：已有键跳过，不覆盖。
	base := map[string]string{"i:a": "<m:oMath>SENTINEL</m:oMath>"}
	out := FormulasOf([]any{"$a$"}, base)
	if out["i:a"] != "<m:oMath>SENTINEL</m:oMath>" {
		t.Fatal("已有键应跳过不覆盖")
	}
}

// TestFormulasOfDisplayVsInline 同 source 的行内/行间是两个键。
func TestFormulasOfDisplayVsInline(t *testing.T) {
	got := FormulasOf([]any{"$x$ $$x$$"}, nil)
	if _, ok := got["i:x"]; !ok {
		t.Fatal("缺 i:x")
	}
	if _, ok := got["d:x"]; !ok {
		t.Fatal("缺 d:x")
	}
}

// TestFormulasOfScanSemantics 扫描对齐 DOLLARS 正则语义的边角：
// `$a$$b$` → 只有 b 成行内公式（a 的闭合 $ 后紧跟 $，前瞻否决）；
// `$$$$`（空内容）不成公式；`$$a$$` 优先于单 $。
func TestFormulasOfScanSemantics(t *testing.T) {
	got := FormulasOf([]any{"$a$$b$"}, nil)
	if len(got) != 1 {
		t.Fatalf(`"$a$$b$" 应只出 1 条：%v`, got)
	}
	if _, ok := got["i:b"]; !ok {
		t.Fatalf(`应恰为 i:b：%v`, got)
	}
	if got2 := FormulasOf([]any{"$$$$"}, nil); len(got2) != 0 {
		t.Fatalf("空内容不成公式：%v", got2)
	}
	got3 := FormulasOf([]any{"$$a$$"}, nil)
	if _, ok := got3["d:a"]; !ok || len(got3) != 1 {
		t.Fatalf("$$ 应优先配对：%v", got3)
	}
	// 含换行：$ 段内不跨行（上游 [^$\r\n]）。
	got4 := FormulasOf([]any{"$a\nb$"}, nil)
	if len(got4) != 0 {
		t.Fatalf("公式内换行应截断匹配：%v", got4)
	}
}

// keysOf 排序键列表（报错信息用）。
func keysOf(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
