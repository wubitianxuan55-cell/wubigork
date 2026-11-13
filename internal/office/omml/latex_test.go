// latex_test.go — LaTeXToOMML 表驱动测试：覆盖支持子集的关键 OMML 形态
// （断言关键片段与 oMath 包裹）与失败语义（超出子集一律 ok=false）；
// 另有抗畸形输入（fuzz 心态）测试：任何路径不许 panic。
package omml

import (
	"math/rand"
	"strings"
	"testing"
)

// TestOmmlLatexToOMML 表驱动：ok=true 断言 oMath 包裹 + 关键片段；
// ok=false 断言整体失败（fail-soft，返回空串）。
func TestOmmlLatexToOMML(t *testing.T) {
	cases := []struct {
		name string
		tex  string
		ok   bool
		want []string // 成功时必须包含的 OMML 片段
	}{
		// 希腊字母（含变体）与符号
		{"希腊字母", `\alpha+\beta-\Omega`, true, []string{"α", "β", "Ω"}},
		{"希腊变体", `\varepsilon \phi \varphi`, true, []string{"ε", "ϕ", "φ"}},
		{"常用符号", `a\times b\cdot c \pm \infty`, true, []string{"×", "·", "±", "∞"}},
		{"关系符号", `a \leq b \geq c \neq d`, true, []string{"≤", "≥", "≠"}},
		{"集合符号", `x \in A \cup B \cap \emptyset`, true, []string{"∈", "∪", "∩", "∅"}},
		{"箭头", `x \to y \Rightarrow z`, true, []string{"→", "⇒"}},
		{"省略号", `1, 2, \ldots, n`, true, []string{"…"}},
		// 上下标
		{"上标", `x^2`, true, []string{"<m:sSup>", "<m:sup>"}},
		{"下标", `x_1`, true, []string{"<m:sSub>", "<m:sub>"}},
		{"上下标组合", `x_i^2`, true, []string{"<m:sSubSup>", "<m:sub>", "<m:sup>"}},
		{"组上标", `x^{n+1}`, true, []string{"<m:sSup>", ">n+1</m:t>"}},
		{"撇号上标", `f'(x)`, true, []string{"<m:sSup>", "′"}},
		{"结构上标", `\frac{a}{b}^2`, true, []string{"<m:sSup><m:e><m:f>"}},
		// 分数与根号
		{"分数", `\frac{a}{b}`, true, []string{"<m:f>", "<m:num>", "<m:den>"}},
		{"dfrac", `\dfrac{1}{2}`, true, []string{"<m:f>"}},
		{"单 token 参数", `\frac12`, true, []string{"<m:f>", ">1</m:t>", ">2</m:t>"}},
		{"根号", `\sqrt{x}`, true, []string{"<m:rad>", `<m:degHide m:val="1"/>`, "<m:deg/>"}},
		{"n次根", `\sqrt[3]{x}`, true, []string{"<m:rad>", "<m:deg>", ">3</m:t>"}},
		{"根号套分数", `\sqrt{\frac{a}{b}}`, true, []string{"<m:rad>", "<m:f>"}},
		// 自适应括号
		{"圆括号", `\left(a+b\right)`, true, []string{"<m:d>", `<m:begChr m:val="("/>`, `<m:endChr m:val=")"/>`}},
		{"方括号", `\left[ x \right]`, true, []string{`<m:begChr m:val="["/>`, `<m:endChr m:val="]"/>`}},
		{"花括号定界", `\left\{ x \right\}`, true, []string{`<m:begChr m:val="{"/>`, `<m:endChr m:val="}"/>`}},
		{"竖线定界", `\left| x \right|`, true, []string{`<m:begChr m:val="|"/>`}},
		{"left点", `\left. x \right.`, true, []string{`<m:begChr m:val=""/>`, `<m:endChr m:val=""/>`}},
		{"嵌套定界", `\left( \left[ a \right] \right)`, true, []string{"<m:d><m:dPr>"}},
		// 矩阵与 cases
		{"pmatrix", `\begin{pmatrix} a & b \\ c & d \end{pmatrix}`, true, []string{"<m:d>", "<m:m>", "<m:mr>", "<m:e>"}},
		{"bmatrix", `\begin{bmatrix} 1 \end{bmatrix}`, true, []string{`<m:begChr m:val="["/>`, "<m:m>"}},
		{"vmatrix", `\begin{vmatrix} a \\ b \end{vmatrix}`, true, []string{`<m:begChr m:val="‖"/>`, "<m:mr>"}},
		{"matrix 裸", `\begin{matrix} a & b \end{matrix}`, true, []string{"<m:m>", "<m:mcs>"}},
		{"矩阵列补齐", `\begin{pmatrix} a & b \\ c \end{pmatrix}`, true, []string{"<m:e/>"}},
		{"行尾多余行隔容忍", `\begin{pmatrix} a \\ \end{pmatrix}`, true, []string{"<m:mr>"}},
		{"cases", `\begin{cases} x & \text{if } x>0 \\ 0 & \text{otherwise} \end{cases}`, true, []string{`<m:begChr m:val="{"/>`, "<m:m>", "<m:nor/>"}},
		// 装饰
		{"hat", `\hat{y}`, true, []string{"<m:acc>", "\u0302"}},
		{"vec", `\vec{v}`, true, []string{"<m:acc>", "\u20d7"}},
		{"dot ddot", `\dot{x} \ddot{y}`, true, []string{"\u0307", "\u0308"}},
		{"overline", `\overline{x}`, true, []string{"<m:bar>", `<m:pos m:val="top"/>`}},
		{"underline", `\underline{x}`, true, []string{"<m:bar>", `<m:pos m:val="bot"/>`}},
		// nary 带限
		{"求和带限", `\sum_{i=1}^{n} i`, true, []string{"<m:nary>", `<m:chr m:val="∑"/>`, `<m:limLoc m:val="undOvr"/>`, "<m:sub>", "<m:sup>"}},
		{"裸求和隐藏限位", `\sum x_i`, true, []string{`<m:subHide m:val="1"/>`, `<m:supHide m:val="1"/>`}},
		{"积分限位", `\int_0^1 f`, true, []string{`<m:chr m:val="∫"/>`, `<m:limLoc m:val="subSup"/>`}},
		{"二重积分", `\iint_D f`, true, []string{`<m:chr m:val="∬"/>`}},
		{"连乘", `\prod_{k} a_k`, true, []string{`<m:chr m:val="∏"/>`, "undOvr"}},
		// lim 与 overset/underset
		{"极限", `\lim_{x \to 0}`, true, []string{"<m:limLow>", "<m:lim>", "<m:nor/>", "→"}},
		{"overset", `\overset{a}{b}`, true, []string{"<m:limUpp>", "<m:lim>"}},
		{"underset", `\underset{n}{x}`, true, []string{"<m:limLow>", "<m:lim>"}},
		// 函数名直立与文字
		{"函数名", `\sin x`, true, []string{`<m:nor/>`, ">sin</m:t>"}},
		{"对数带底", `\log_2 x`, true, []string{"<m:sSub>", `<m:nor/>`, ">log</m:t>"}},
		{"operatorname", `\operatorname{sgn}x`, true, []string{`<m:nor/>`, ">sgn</m:t>"}},
		{"mathrm", `\mathrm{d}x`, true, []string{`<m:sty m:val="p"/>`}},
		{"mathbf", `\mathbf{v}`, true, []string{`<m:sty m:val="b"/>`}},
		{"text", `\text{if }x`, true, []string{"<m:nor/>", ">if </m:t>"}},
		// 转义字面量、否定、空距、分组
		{"转义花括号", `\{a\}`, true, []string{">{a}</m:t>"}},
		{"转义字面量", `50\% \#3 \$5`, true, []string{"%", "#", "$"}},
		{"否定等价", `x\not\equiv y`, true, []string{"≢"}},
		{"否定小于", `a\not<b`, true, []string{"≮"}},
		{"neq", `a \neq b`, true, []string{"≠"}},
		{"空距", `a\,b`, true, []string{">a b</m:t>"}},
		{"displaystyle 剥包装", `{\displaystyle x}`, true, []string{">x</m:t>"}},
		{" Cambria 字体", `q`, true, []string{`<w:rFonts w:ascii="Cambria Math" w:hAnsi="Cambria Math"/>`}},
		// 失败例：fail-soft，超出子集一律 false
		{"未知命令", `\unknowncmd{x}`, false, nil},
		{"left 不平衡", `\left(a`, false, nil},
		{"孤立上标", `^2`, false, nil},
		{"未闭合组", `{a`, false, nil},
		{"多余右括号", `a}`, false, nil},
		{"双上标", `x^2^3`, false, nil},
		{"脚本缺参", `x^`, false, nil},
		{"环境外 &", `a & b`, false, nil},
		{"未知环境", `\begin{align} a \end{align}`, false, nil},
		{"环境未闭合", `\begin{matrix} a`, false, nil},
		{"end 单独出现", `a\end{matrix}`, false, nil},
		{"right 单独出现", `a\right)`, false, nil},
		{"未知否定", `\not\unknown`, false, nil},
		{"裸行分隔", `a\\b`, false, nil},
		{"空公式", `   `, false, nil},
		{"负空距仅剩空", `\!`, false, nil},
		{"分数缺参", `\frac{a}`, false, nil},
		{"sqrt 缺参", `\sqrt`, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := LaTeXToOMML(tc.tex, true)
			if ok != tc.ok {
				t.Fatalf("LaTeXToOMML(%q) ok=%v, want %v（got=%.200s）", tc.tex, ok, tc.ok, got)
			}
			if !ok {
				if got != "" {
					t.Fatalf("失败例应返回空串，got=%.100s", got)
				}
				return
			}
			if !strings.HasPrefix(got, "<m:oMath>") || !strings.HasSuffix(got, "</m:oMath>") {
				t.Fatalf("应被 <m:oMath> 包裹：%.200s", got)
			}
			for _, frag := range tc.want {
				if !strings.Contains(got, frag) {
					t.Fatalf("缺关键片段 %q：%.300s", frag, got)
				}
			}
		})
	}
}

// TestOmmlDisplayInvariant display 只影响 helper 的 oMathPara 包裹，不改变
// 本函数的 OMML 内容（同一 source 两种模式输出一致）。
func TestOmmlDisplayInvariant(t *testing.T) {
	for _, tex := range []string{`x^2`, `\frac{a}{b}`, `\sum_{i=1}^n i`, `\alpha`} {
		inline, ok1 := LaTeXToOMML(tex, false)
		disp, ok2 := LaTeXToOMML(tex, true)
		if ok1 != ok2 || inline != disp {
			t.Fatalf("display 不应改变 OMML：%q → %v/%v", tex, inline, disp)
		}
	}
}

// TestOmmlNoPanicOnMalformed 畸形/病态输入一律不得 panic，超深嵌套不得栈溢出
// （红线：转换器任何路径不许 panic）。成功路径的产物仍须是良构 oMath。
func TestOmmlNoPanicOnMalformed(t *testing.T) {
	nasty := []string{
		"", " ", "$", "\\", "&", "^", "_", "}", "]", "'",
		`\\`, `\left`, `\right`, `\frac`, `\frac{`, `\frac{a}`, `\sqrt[`,
		`\sqrt[`, `\begin`, `\begin{`, `\begin{matrix}a`, `\begin{matrix}&\end{x}`,
		`x^`, `x_{`, `\not`, `\not\`, `\hat`, `\operatorname`, `\text{`,
		"\x00", "\v", "\u2028", "ααα", strings.Repeat(`\left(`, 100),
		strings.Repeat("{", 200), strings.Repeat("}", 200),
		strings.Repeat(`\frac{`, 100), strings.Repeat(`\sum`, 300),
		strings.Repeat(`x^`, 200), strings.Repeat(`\begin{pmatrix}`, 50),
		`\begin{matrix}` + strings.Repeat("a&", 500) + `\end{matrix}`,
	}
	for _, s := range nasty {
		got, ok := LaTeXToOMML(s, true)
		if ok {
			if !strings.HasPrefix(got, "<m:oMath>") || !strings.HasSuffix(got, "</m:oMath>") {
				t.Fatalf("畸形输入 %q 意外成功且产物不良构：%.200s", s, got)
			}
		} else if got != "" {
			t.Fatalf("失败例应返回空串：%.100s", got)
		}
		// display=false 同样不许 panic。
		LaTeXToOMML(s, false)
	}
	// 固定种子的伪随机串：不 panic 即通过；成功则产物必须良构。
	rng := rand.New(rand.NewSource(20261113))
	alphabet := []rune(`$\{}^_&'ab1. \fracsumqrtbeginmatrix`)
	for i := 0; i < 500; i++ {
		buf := make([]rune, rng.Intn(48))
		for j := range buf {
			buf[j] = alphabet[rng.Intn(len(alphabet))]
		}
		got, _ := LaTeXToOMML(string(buf), i%2 == 0)
		if got != "" && (!strings.HasPrefix(got, "<m:oMath>") || !strings.HasSuffix(got, "</m:oMath>")) {
			t.Fatalf("随机输入产物不良构：%.100s", got)
		}
	}
}
