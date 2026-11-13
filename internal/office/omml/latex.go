// Package omml — LaTeX 公式子集 → OMML XML 纯函数转换器（dsh-office COM 档
// 波2「WPS 公式预转 math 腿」）。零外部依赖。
//
// 背景：上游 dsh-office 对 WPS 用 temml(LaTeX→MathML)→OMML 预转公式，随 edit
// 调用的 math 字段下发（map：d:<source>/i:<source> → <m:oMath>…</m:oMath>）；
// helper 侧（internal/office/comhelper/OfficeHelper.cs，逐字节保真，勿改）在
// WPS 档按 source 查表 InsertXML，查不到的公式保留 $…$ 原文不报错；Word 档走
// 自带 LaTeX 解析器，完全不受影响。本包不经 MathML 中间层，直接 LaTeX→OMML，
// 输出形态对齐上游 .tmp/dsh-office-upstream-src/src/omml.ts（m:r 的
// m:rPr/w:rPr 结构、Cambria Math 字体、m:f/m:rad/m:sSup/m:d/m:m/m:nary 等）。
//
// 红线：任何路径不许 panic——不认识的命令/括号不平衡/不支持的环境/超深嵌套
// 一律返回 ("", false)，调用方 fail-soft（不进 math map，helper 保留原文）。
package omml

import (
	"strings"
	"unicode"
)

const (
	// ommlFont 与上游 omml.ts 的 FONT 常量逐字节一致：数学 run 的字体。
	ommlFont = `<w:rPr><w:rFonts w:ascii="Cambria Math" w:hAnsi="Cambria Math"/></w:rPr>`
	// parseDepthLimit 嵌套深度上限（组/参数/nary 操作数/环境均计深度）：
	// 防 `{`、`\frac`、`\sum\sum…` 等病态输入把递归打成栈溢出——超限即失败。
	parseDepthLimit = 80
	// 矩阵规模上限（fuzz 心态：超大表直接失败）。
	maxMatrixRows = 100
	maxMatrixCols = 50
)

// LaTeXToOMML 把一段 LaTeX（已剥 $ 与 \tag）转成 OMML XML 字符串
// （<m:oMath>…</m:oMath>）。超出支持子集（含空公式）时返回 ("", false)——
// 调用方不把该公式放进 math map，helper 保留原文。display 只影响上游 helper
// 的 oMathPara 包裹，不改变本函数的 OMML 内容。
func LaTeXToOMML(tex string, display bool) (out string, ok bool) {
	// 红线兜底：深度上限是主防线，recover 是保险——任何路径不许 panic。
	defer func() {
		if r := recover(); r != nil {
			out, ok = "", false
		}
	}()
	p := &mathParser{src: []rune(tex)}
	if _, sOK := p.parseSeq(stopSet{}); !sOK {
		return "", false
	}
	p.flushRun()
	if len(p.out) == 0 {
		return "", false // 空公式（上游 body 为空 → undefined 同语义）
	}
	return "<m:oMath>" + string(p.out) + "</m:oMath>", true
}

// ── 词表 ─────────────────────────────────────────────────────────────────

// symbols 希腊字母全表（含变体）+ 常用符号表：命令名 → Unicode 字符。
// 取值对齐 KaTeX/temml 语义（如 \epsilon=ϵ 与 \varepsilon=ε、\phi=ϕ 与
// \varphi=φ 有别）。
var symbols = map[string]string{
	// 希腊字母（小写与变体）
	"alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ",
	"epsilon": "ϵ", "varepsilon": "ε",
	"zeta": "ζ", "eta": "η",
	"theta": "θ", "vartheta": "ϑ",
	"iota": "ι", "kappa": "κ", "lambda": "λ",
	"mu": "μ", "nu": "ν", "xi": "ξ", "omicron": "ο",
	"pi": "π", "varpi": "ϖ",
	"rho": "ρ", "varrho": "ϱ",
	"sigma": "σ", "varsigma": "ς",
	"tau": "τ", "upsilon": "υ",
	"phi": "ϕ", "varphi": "φ",
	"chi": "χ", "psi": "ψ", "omega": "ω",
	// 希腊字母（大写）
	"Gamma": "Γ", "Delta": "Δ", "Theta": "Θ", "Lambda": "Λ",
	"Xi": "Ξ", "Pi": "Π", "Sigma": "Σ", "Upsilon": "Υ",
	"Phi": "Φ", "Psi": "Ψ", "Omega": "Ω",
	// 运算与关系
	"times": "×", "cdot": "·", "div": "÷",
	"pm": "±", "mp": "∓",
	"leq": "≤", "le": "≤", "geq": "≥", "ge": "≥",
	"neq": "≠", "ne": "≠",
	"approx": "≈", "equiv": "≡", "sim": "∼", "simeq": "≃", "cong": "≅",
	"propto": "∝",
	// 裸符号
	"infty": "∞", "partial": "∂", "nabla": "∇",
	"hbar": "ℏ", "ell": "ℓ", "Re": "ℜ", "Im": "ℑ", "prime": "′",
	"circ": "∘", "bullet": "∙", "star": "⋆", "ast": "∗",
	"oplus": "⊕", "otimes": "⊗", "odot": "⊙",
	// 集合与逻辑
	"in": "∈", "notin": "∉",
	"subset": "⊂", "subseteq": "⊆", "supset": "⊃", "supseteq": "⊇",
	"cup": "∪", "cap": "∩", "emptyset": "∅",
	"forall": "∀", "exists": "∃", "neg": "¬",
	"land": "∧", "lor": "∨", "wedge": "∧", "vee": "∨",
	// 箭头
	"Rightarrow": "⇒", "rightarrow": "→", "to": "→",
	"leftarrow": "←", "gets": "←", "leftrightarrow": "↔",
	"mapsto": "↦", "uparrow": "↑", "downarrow": "↓",
	// 省略号与其他
	"ldots": "…", "cdots": "⋯", "vdots": "⋮", "ddots": "⋱", "dots": "…",
	"angle": "∠", "perp": "⊥", "parallel": "∥", "mid": "∣",
}

// funcNames 函数名：经命令进来才直立（m:rPr m:nor），裸字母序列仍是变量斜体。
var funcNames = map[string]bool{
	"sin": true, "cos": true, "tan": true, "cot": true, "sec": true, "csc": true,
	"arcsin": true, "arccos": true, "arctan": true,
	"sinh": true, "cosh": true, "tanh": true, "coth": true,
	"log": true, "ln": true, "lg": true, "exp": true,
	"lim": true, "min": true, "max": true, "sup": true, "inf": true,
	"arg": true, "det": true, "dim": true, "gcd": true,
	"hom": true, "ker": true, "deg": true,
}

// naryOps n 元算符：chr + 限位位置（\int 系=subSup，\sum 系=undOvr）。
var naryOps = map[string]struct {
	chr    string
	undOvr bool
}{
	"sum":    {"∑", true},
	"prod":   {"∏", true},
	"bigcup": {"⋃", true},
	"bigcap": {"⋂", true},
	"int":    {"∫", false},
	"iint":   {"∬", false},
	"iiint":  {"∭", false},
	"oint":   {"∮", false},
}

// accents 重音命令 → OMML m:chr 用的组合附加符（同上游 ACCENTS 映射）。
var accents = map[string]string{
	"hat":   "\u0302",
	"tilde": "\u0303",
	"vec":   "\u20d7",
	"dot":   "\u0307",
	"ddot":  "\u0308",
}

// negByChar / negByCmd \not 否定查表：后跟字符或命令；查不到 → 整体失败。
var negByChar = map[rune]string{'=': "≠", '<': "≮", '>': "≯"}
var negByCmd = map[string]string{
	"equiv": "≢", "in": "∉", "notin": "∉",
	"subset": "⊄", "supset": "⊅",
	"subseteq": "⊈", "supseteq": "⊉",
	"leq": "≰", "le": "≰", "geq": "≱", "ge": "≱",
	"approx": "≉", "sim": "≁", "simeq": "≄", "cong": "≇",
}

// matrixEnvs 支持的矩阵/环境名。
var matrixEnvs = map[string]bool{
	"matrix": true, "pmatrix": true, "bmatrix": true, "vmatrix": true, "cases": true,
}

// ── 解析器 ────────────────────────────────────────────────────────────────

// atomVal 一个原子：xml 为结构化片段（空 = 文字已入 run 缓冲或 no-op 命令）；
// nary/lim 标记影响后续脚本（^ _ '）的组装方式。
type atomVal struct {
	xml    string
	nary   bool
	chr    string // nary 符号
	undOvr bool   // nary 限位：true=undOvr（\sum 系），false=subSup（\int 系）
	lim    bool   // \lim（下标组装为 m:limLow）
}

// stopSet 序列解析的停止条件（都由调用方负责消费停止符——除 bracket/right 中
// 已注明的部分）。
type stopSet struct {
	brace   bool // 停在 }（不消费，调用方消费）
	cell    bool // 停在 & / \\ / \end（矩阵环境单元，均不消费）
	right   bool // 停在 \right（不消费，\left 分支消费）
	bracket bool // 停在 ]（消费 ]）
}

// 序列停止原因。
const (
	stopEnd     = "end"     // 输入结束（仅顶层合法）
	stopBrace   = "brace"   // 停在 }
	stopAmp     = "amp"     // 停在 &（列分隔）
	stopRowsep  = "rowsep"  // 停在 \\（行分隔）
	stopEndEnv  = "endenv"  // 停在 \end（环境收尾）
	stopRight   = "right"   // 停在 \right
	stopBracket = "bracket" // 停在 ]
)

type mathParser struct {
	src   []rune
	pos   int
	depth int
	// style 当前文字样式：""=数学默认（斜体）、"p"=\mathrm 直立、
	// "b"=\mathbf 粗体、"nor"=\text/函数名普通文字。样式变化即分 run。
	style string
	// runBuf 待合并进同一 m:r 的连续文字（可跨原子合并，显示等价）。
	runBuf   string
	runStyle string
	// out 累计 OMML 输出（可截断：组/单元格/参数经 takeFrom 取值为字符串）。
	out []byte
}

func (p *mathParser) outLen() int       { return len(p.out) }
func (p *mathParser) writeOut(s string) { p.out = append(p.out, s...) }

// skipSpaces 跳过空白（数学模式空白不参与输出；\text 内不走这里）。
func (p *mathParser) skipSpaces() {
	for p.pos < len(p.src) && unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
}

// flushRun 把缓冲文字落成 m:r 写入 out，返回该 XML（无缓冲时返回 ""）。
func (p *mathParser) flushRun() string {
	if p.runBuf == "" {
		return ""
	}
	r := ommlRun(p.runBuf, p.runStyle)
	p.runBuf = ""
	p.writeOut(r)
	return r
}

// putTextIn 追加文字到 run 缓冲；样式不同则先落盘旧 run（m:r 边界=样式边界）。
func (p *mathParser) putTextIn(style, s string) {
	if s == "" {
		return
	}
	if p.runBuf != "" && p.runStyle != style {
		p.flushRun()
	}
	if p.runBuf == "" {
		p.runStyle = style
	}
	p.runBuf += s
}

// emit 落盘缓冲后写入结构化 XML（保证顺序：文字永远先于其后出现的结构）。
func (p *mathParser) emit(s string) {
	p.flushRun()
	p.writeOut(s)
}

// takeFrom 取走 out 自 mark 起的片段（先落盘缓冲），用于把已渲染内容作为
// 子树（参数/单元格/基底）重新组装。
func (p *mathParser) takeFrom(mark int) string {
	p.flushRun()
	if mark > len(p.out) {
		return ""
	}
	v := string(p.out[mark:])
	p.out = p.out[:mark]
	return v
}

// parseSeq 解析一个 token 序列直到输入结束或命中停止条件。
// 返回停止原因；ok=false 表示语法失败（不平衡/不认识/超深）。
func (p *mathParser) parseSeq(stop stopSet) (string, bool) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > parseDepthLimit {
		return "", false
	}
	for {
		p.skipSpaces()
		if p.pos >= len(p.src) {
			if stop.brace || stop.cell || stop.right || stop.bracket {
				return "", false // 括号/环境/定界未闭合
			}
			return stopEnd, true
		}
		c := p.src[p.pos]
		switch {
		case c == '}':
			if !stop.brace {
				return "", false // 多余的 }
			}
			return stopBrace, true
		case c == '&':
			if !stop.cell {
				return "", false // 环境外不得出现 &（KaTeX 同为错误）
			}
			return stopAmp, true
		case c == ']' && stop.bracket:
			p.pos++
			return stopBracket, true
		case c == '\\' && p.rowsepAt():
			if !stop.cell {
				return "", false // 行分隔符出现在非表格上下文
			}
			return stopRowsep, true
		case c == '\\' && p.cmdPeekIs("right"):
			if !stop.right {
				return "", false // 无配对 \left
			}
			return stopRight, true // 不消费，\left 分支消费 \right 及定界符
		case c == '\\' && p.cmdPeekIs("end"):
			if !stop.cell {
				return "", false // \end 无环境上下文
			}
			return stopEndEnv, true
		case c == '^' || c == '_' || c == '\'':
			return "", false // 脚本缺基底（KaTeX 同为错误）
		default:
			if !p.parseTerm() {
				return "", false
			}
		}
	}
}

// rowsepAt 当前是否为 \\ 行分隔符。
func (p *mathParser) rowsepAt() bool {
	return p.pos+1 < len(p.src) && p.src[p.pos] == '\\' && p.src[p.pos+1] == '\\'
}

// cmdPeekIs 当前是否为指定命令名（\ + name + 非字母），不消费。
func (p *mathParser) cmdPeekIs(name string) bool {
	if p.pos >= len(p.src) || p.src[p.pos] != '\\' {
		return false
	}
	rest := string(p.src[p.pos+1:])
	if len(rest) < len(name) || rest[:len(name)] != name {
		return false
	}
	if len(rest) > len(name) && isASCIILetter(rune(rest[len(name)])) {
		return false // 更长的命令名（如 \right 拒绝 \rightarrow 匹配）
	}
	return true
}

// cmdNameAt 读取 \ 后的命令名（不消费）。
func (p *mathParser) cmdNameAt() string {
	start := p.pos + 1
	end := start
	for end < len(p.src) && isASCIILetter(p.src[end]) {
		end++
	}
	return string(p.src[start:end])
}

// parseTerm 一个原子及其后缀脚本（^ _ '），nary 原子额外吃紧随的一个原子作
// 操作数（m:e）。这是 OMML 组装的核心。
func (p *mathParser) parseTerm() bool {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > parseDepthLimit {
		return false
	}
	mark0 := p.outLen()
	a, ok := p.parseAtom()
	if !ok {
		return false
	}
	// 收集脚本：^/_/'（撇号 = \prime 连打）；同一脚本重复出现 → 失败。
	var sub, sup string
	hasSub, hasSup := false, false
	for {
		p.skipSpaces()
		if p.pos >= len(p.src) {
			break
		}
		c := p.src[p.pos]
		if c == '^' && !hasSup {
			p.pos++
			v, ok2 := p.argValue()
			if !ok2 {
				return false
			}
			sup, hasSup = v, true
			continue
		}
		if c == '_' && !hasSub {
			p.pos++
			v, ok2 := p.argValue()
			if !ok2 {
				return false
			}
			sub, hasSub = v, true
			continue
		}
		if c == '\'' && !hasSup {
			n := 0
			for p.pos < len(p.src) && p.src[p.pos] == '\'' {
				p.pos++
				n++
			}
			sup = ommlRun(strings.Repeat("′", n), "")
			hasSup = true
			continue
		}
		break
	}
	if a.nary {
		// m:nary：chr + limLoc + 缺省隐藏 + 操作数（紧随的一个原子链）。
		p.flushRun()
		mark := p.outLen()
		operand := ""
		p.skipSpaces()
		if p.atAtomStart() {
			if !p.parseTerm() {
				return false
			}
			operand = p.takeFrom(mark)
		}
		subEl, supEl := sub, sup
		hide := ""
		if !hasSub {
			hide += `<m:subHide m:val="1"/>`
			subEl = ""
		}
		if !hasSup {
			hide += `<m:supHide m:val="1"/>`
			supEl = ""
		}
		loc := "subSup"
		if a.undOvr {
			loc = "undOvr"
		}
		p.emit(`<m:nary><m:naryPr><m:chr m:val="` + escXML(a.chr) + `"/><m:limLoc m:val="` + loc + `"/>` + hide +
			`</m:naryPr><m:sub>` + subEl + `</m:sub><m:sup>` + supEl + `</m:sup><m:e>` + operand + `</m:e></m:nary>`)
		return true
	}
	if a.xml != "" {
		// 结构命令（frac/sqrt/\left…/矩阵/装饰/限位）：atomVal.xml 携带成品
		// 子树，必须在此 emit——2026-11-13 实锤：无脚本路径漏 emit，xml 被
		// 静默丢弃（out 为空 → 顶层判空 false，\frac{a}{b} 全体红）。
		// 带脚本时结构作基底（如 \sqrt{x}^2）。
		if !hasSub && !hasSup {
			p.emit(a.xml)
			return true
		}
		p.flushRun()
		switch {
		case hasSub && hasSup:
			p.emit(`<m:sSubSup><m:e>` + a.xml + `</m:e><m:sub>` + sub + `</m:sub><m:sup>` + sup + `</m:sup></m:sSubSup>`)
		case hasSub:
			p.emit(`<m:sSub><m:e>` + a.xml + `</m:e><m:sub>` + sub + `</m:sub></m:sSub>`)
		default:
			p.emit(`<m:sSup><m:e>` + a.xml + `</m:e><m:sup>` + sup + `</m:sup></m:sSup>`)
		}
		return true
	}
	if !hasSub && !hasSup {
		return true // 文字留在缓冲（后续合并），结构已在 out，no-op 无输出
	}
	base := p.takeFrom(mark0)
	if base == "" {
		return false // 脚本缺实体基底（如 no-op 命令后接 ^）
	}
	switch {
	case a.lim && hasSub && hasSup:
		// \lim_{a}^{b}：上下限齐备，嵌套组装（同上游 munderover → limUpp(limLow)）。
		p.emit(`<m:limUpp><m:e><m:limLow><m:e>` + base + `</m:e><m:lim>` + sub + `</m:lim></m:limLow></m:e><m:lim>` + sup + `</m:lim></m:limUpp>`)
	case a.lim && hasSub:
		p.emit(`<m:limLow><m:e>` + base + `</m:e><m:lim>` + sub + `</m:lim></m:limLow>`)
	case a.lim:
		p.emit(`<m:limUpp><m:e>` + base + `</m:e><m:lim>` + sup + `</m:lim></m:limUpp>`)
	case hasSub && hasSup:
		p.emit(`<m:sSubSup><m:e>` + base + `</m:e><m:sub>` + sub + `</m:sub><m:sup>` + sup + `</m:sup></m:sSubSup>`)
	case hasSub:
		p.emit(`<m:sSub><m:e>` + base + `</m:e><m:sub>` + sub + `</m:sub></m:sSub>`)
	default:
		p.emit(`<m:sSup><m:e>` + base + `</m:e><m:sup>` + sup + `</m:sup></m:sSup>`)
	}
	return true
}

// atAtomStart 当前是否为原子起点（nary 操作数判定用）。
func (p *mathParser) atAtomStart() bool {
	if p.pos >= len(p.src) {
		return false
	}
	c := p.src[p.pos]
	if c == '{' || unicode.IsLetter(c) || unicode.IsDigit(c) {
		return true
	}
	if c == '\\' && p.pos+1 < len(p.src) && isASCIILetter(p.src[p.pos+1]) {
		name := p.cmdNameAt()
		return name != "right" && name != "end" // 停止符不算原子
	}
	return false
}

// parseAtom 一个原子：花括号组（纯分组，不产 XML）、命令、数字串（含小数）、
// 单个字母或符号。文字原子进入 run 缓冲（xml 返回空）。
func (p *mathParser) parseAtom() (atomVal, bool) {
	p.skipSpaces()
	if p.pos >= len(p.src) {
		return atomVal{}, false
	}
	c := p.src[p.pos]
	switch {
	case c == '{':
		p.pos++
		reason, ok := p.parseSeq(stopSet{brace: true})
		if !ok || reason != stopBrace {
			return atomVal{}, false
		}
		p.pos++ // 消费 }
		return atomVal{}, true
	case c == '\\':
		return p.parseCommand()
	case unicode.IsDigit(c):
		start := p.pos
		for p.pos < len(p.src) && isASCIIDigit(p.src[p.pos]) {
			p.pos++
		}
		// 小数点：仅当点后还有数字（3.14 一个数字原子；孤立句点仍是符号）。
		if p.pos < len(p.src) && p.src[p.pos] == '.' && p.pos+1 < len(p.src) && isASCIIDigit(p.src[p.pos+1]) {
			p.pos++
			for p.pos < len(p.src) && isASCIIDigit(p.src[p.pos]) {
				p.pos++
			}
		}
		p.putTextIn(p.style, string(p.src[start:p.pos]))
		return atomVal{}, true
	default:
		// 字母（含希腊/CJK，unicode.IsLetter）与其余符号字符：逐字符入缓冲
		// 合并成同一 m:r（同上游「逐字符 run 合并」的显示等价取舍）。
		p.pos++
		p.putTextIn(p.style, string(c))
		return atomVal{}, true
	}
}

// parseCommand 解析 \ 命令。词表命中（符号/函数名/nary）产文字或标记；
// 结构命令（分数/根号/括号/矩阵/装饰/限位/样式）产 XML。
func (p *mathParser) parseCommand() (atomVal, bool) {
	p.pos++ // 消费 '\'
	if p.pos >= len(p.src) {
		return atomVal{}, false
	}
	c := p.src[p.pos]
	if !isASCIILetter(c) {
		// 单字符命令：转义字面量、空距与定界。
		p.pos++
		switch c {
		case '{', '}', '#', '$', '&', '_', '%':
			p.putTextIn(p.style, string(c)) // 转义字面量
			return atomVal{}, true
		case ' ', ',', ';', ':':
			p.putTextIn(p.style, " ") // 空距命令统一为普通空格（可合并）
			return atomVal{}, true
		case '!':
			return atomVal{}, true // 负细距：忽略
		case '|':
			p.putTextIn(p.style, "‖")
			return atomVal{}, true
		}
		return atomVal{}, false
	}
	start := p.pos
	for p.pos < len(p.src) && isASCIILetter(p.src[p.pos]) {
		p.pos++
	}
	name := string(p.src[start:p.pos])
	if ch, ok := symbols[name]; ok {
		p.putTextIn(p.style, ch)
		return atomVal{}, true
	}
	if funcNames[name] {
		p.putTextIn("nor", name) // 函数名直立（m:rPr m:nor）
		return atomVal{lim: name == "lim"}, true
	}
	if op, ok := naryOps[name]; ok {
		return atomVal{nary: true, chr: op.chr, undOvr: op.undOvr}, true
	}
	switch name {
	case "not":
		return p.parseNot()
	case "hat", "tilde", "vec", "dot", "ddot":
		v, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		return atomVal{xml: `<m:acc><m:accPr><m:chr m:val="` + accents[name] + `"/></m:accPr><m:e>` + v + `</m:e></m:acc>`}, true
	case "bar", "overline":
		v, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		return atomVal{xml: `<m:bar><m:barPr><m:pos m:val="top"/></m:barPr><m:e>` + v + `</m:e></m:bar>`}, true
	case "underline":
		v, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		return atomVal{xml: `<m:bar><m:barPr><m:pos m:val="bot"/></m:barPr><m:e>` + v + `</m:e></m:bar>`}, true
	case "frac", "dfrac", "tfrac":
		num, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		den, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		return atomVal{xml: `<m:f><m:num>` + num + `</m:num><m:den>` + den + `</m:den></m:f>`}, true
	case "sqrt":
		p.skipSpaces()
		deg := ""
		if p.pos < len(p.src) && p.src[p.pos] == '[' {
			p.pos++
			mark := p.outLen()
			reason, ok := p.parseSeq(stopSet{bracket: true})
			if !ok || reason != stopBracket {
				return atomVal{}, false
			}
			deg = p.takeFrom(mark)
		}
		body, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		if deg == "" {
			return atomVal{xml: `<m:rad><m:radPr><m:degHide m:val="1"/></m:radPr><m:deg/><m:e>` + body + `</m:e></m:rad>`}, true
		}
		return atomVal{xml: `<m:rad><m:deg>` + deg + `</m:deg><m:e>` + body + `</m:e></m:rad>`}, true
	case "overset":
		a, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		b, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		return atomVal{xml: `<m:limUpp><m:e>` + b + `</m:e><m:lim>` + a + `</m:lim></m:limUpp>`}, true
	case "underset":
		a, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		b, ok := p.argValue()
		if !ok {
			return atomVal{}, false
		}
		return atomVal{xml: `<m:limLow><m:e>` + b + `</m:e><m:lim>` + a + `</m:lim></m:limLow>`}, true
	case "mathrm", "mathbf", "operatorname", "text":
		// 样式包裹：\mathrm→p、\mathbf→b、\operatorname/\text→nor；组内容
		// 原位渲染（文字仍走 run 缓冲，样式切换自然分 run）。
		saved := p.style
		switch name {
		case "mathrm":
			p.style = "p"
		case "mathbf":
			p.style = "b"
		default:
			p.style = "nor"
		}
		var ok bool
		if name == "text" {
			ok = p.textArg()
		} else {
			ok = p.argGroup()
		}
		p.style = saved
		if !ok {
			return atomVal{}, false
		}
		return atomVal{}, true
	case "left":
		beg, ok := p.delim()
		if !ok {
			return atomVal{}, false
		}
		p.flushRun() // 先落盘之前的文字，防止混进 m:d 体
		mark := p.outLen()
		reason, ok := p.parseSeq(stopSet{right: true})
		if !ok || reason != stopRight {
			return atomVal{}, false
		}
		if !p.cmdPeekIs("right") {
			return atomVal{}, false
		}
		p.pos += len(`\right`)
		end, ok := p.delim()
		if !ok {
			return atomVal{}, false
		}
		body := p.takeFrom(mark)
		return atomVal{xml: `<m:d><m:dPr><m:begChr m:val="` + escXML(beg) + `"/><m:endChr m:val="` + escXML(end) + `"/></m:dPr><m:e>` + body + `</m:e></m:d>`}, true
	case "begin":
		env, ok := p.envName()
		if !ok {
			return atomVal{}, false
		}
		if matrixEnvs[env] {
			return p.parseMatrixEnv(env)
		}
		return atomVal{}, false // 不支持的环境名
	case "displaystyle", "limits", "nolimits":
		return atomVal{}, true // no-op：{\displaystyle …} 剥包装、限位声明忽略
	case "end", "right":
		return atomVal{}, false // 配对符单独出现
	}
	return atomVal{}, false // 不认识的命令 → fail-soft
}

// parseNot 处理 \not 否定：后跟字符/命令查表换否定符；查不到 → 失败。
func (p *mathParser) parseNot() (atomVal, bool) {
	p.skipSpaces()
	if p.pos >= len(p.src) {
		return atomVal{}, false
	}
	c := p.src[p.pos]
	if ch, ok := negByChar[c]; ok {
		p.pos++
		p.putTextIn(p.style, ch)
		return atomVal{}, true
	}
	if c == '\\' {
		p.pos++
		start := p.pos
		for p.pos < len(p.src) && isASCIILetter(p.src[p.pos]) {
			p.pos++
		}
		if p.pos == start {
			return atomVal{}, false
		}
		ch, ok := negByCmd[string(p.src[start:p.pos])]
		if !ok {
			return atomVal{}, false
		}
		p.putTextIn(p.style, ch)
		return atomVal{}, true
	}
	return atomVal{}, false
}

// argValue 取一个命令参数为字符串（{组} | \命令 | 单字符）。
// 用于 ^/_ 脚本、\frac、\sqrt、装饰等；no-op 命令做参数 → 失败。
func (p *mathParser) argValue() (string, bool) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > parseDepthLimit {
		return "", false
	}
	p.flushRun() // 先把基文字落盘，保证取值不含基底
	p.skipSpaces()
	if p.pos >= len(p.src) {
		return "", false
	}
	switch c := p.src[p.pos]; {
	case c == '{':
		p.pos++
		mark := p.outLen()
		reason, ok := p.parseSeq(stopSet{brace: true})
		if !ok || reason != stopBrace {
			return "", false
		}
		v := p.takeFrom(mark)
		p.pos++ // 消费 }
		return v, true
	case c == '\\':
		mark := p.outLen()
		if _, ok := p.parseAtom(); !ok {
			return "", false
		}
		v := p.takeFrom(mark)
		if v == "" {
			return "", false // no-op 命令（\displaystyle 等）不做参数
		}
		return v, true
	default:
		if c == '}' || c == '&' || c == '^' || c == '_' || c == ']' {
			return "", false // 结构字符不做参数（x^} 等畸形）
		}
		mark := p.outLen()
		p.putTextIn(p.style, string(c))
		p.pos++
		v := p.takeFrom(mark)
		if v == "" {
			return "", false
		}
		return v, true
	}
}

// argGroup 原位渲染 {…} 组内容（\mathrm/\operatorname 样式包裹用）。
func (p *mathParser) argGroup() bool {
	p.skipSpaces()
	if p.pos >= len(p.src) || p.src[p.pos] != '{' {
		return false
	}
	p.pos++
	reason, ok := p.parseSeq(stopSet{brace: true})
	if !ok || reason != stopBrace {
		return false
	}
	p.pos++ // 消费 }
	return true
}

// textArg 扫描 \text{…} 原文（保留空格、花括号分组剥壳、只解码转义字面量）。
func (p *mathParser) textArg() bool {
	p.skipSpaces()
	if p.pos >= len(p.src) || p.src[p.pos] != '{' {
		return false
	}
	p.pos++
	depth := 1
	var sb strings.Builder
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == '{' {
			depth++
			p.pos++
			continue
		}
		if c == '}' {
			depth--
			p.pos++
			if depth == 0 {
				p.putTextIn("nor", sb.String())
				return true
			}
			continue
		}
		if c == '\\' {
			p.pos++
			if p.pos >= len(p.src) {
				return false
			}
			e := p.src[p.pos]
			switch e {
			case '{', '}', '#', '$', '&', '_', '%', '\\':
				sb.WriteRune(e)
			case ' ', ',', ';', ':':
				sb.WriteRune(' ')
			default:
				return false
			}
			p.pos++
			continue
		}
		sb.WriteRune(c)
		p.pos++
	}
	return false // 未闭合
}

// delim 定界符：常见字符或 \| \{ \}；\left./\right. → 空串（OMML 空属性）。
func (p *mathParser) delim() (string, bool) {
	p.skipSpaces()
	if p.pos >= len(p.src) {
		return "", false
	}
	switch c := p.src[p.pos]; c {
	case '(', ')', '[', ']', '{', '}', '|':
		p.pos++
		return string(c), true
	case '.':
		p.pos++
		return "", true
	case '\\':
		if p.pos+1 < len(p.src) {
			switch p.src[p.pos+1] {
			case '{':
				p.pos += 2
				return "{", true
			case '}':
				p.pos += 2
				return "}", true
			case '|':
				p.pos += 2
				return "‖", true
			}
		}
	}
	return "", false
}

// envName 解析 \begin{env} 的环境名。
func (p *mathParser) envName() (string, bool) {
	p.skipSpaces()
	if p.pos >= len(p.src) || p.src[p.pos] != '{' {
		return "", false
	}
	p.pos++
	start := p.pos
	for p.pos < len(p.src) && isASCIILetter(p.src[p.pos]) {
		p.pos++
	}
	name := string(p.src[start:p.pos])
	if name == "" || p.pos >= len(p.src) || p.src[p.pos] != '}' {
		return "", false
	}
	p.pos++
	return name, true
}

// consumeEnd 消费 \end{env} 并核对环境名（seq 已停在 \end）。
func (p *mathParser) consumeEnd(env string) bool {
	if !p.cmdPeekIs("end") {
		return false
	}
	p.pos += len(`\end`)
	got, ok := p.envName()
	return ok && got == env
}

// parseMatrixEnv 解析矩阵/类矩阵环境：& 分列、\\ 分行、\end{env} 收尾；
// pmatrix/bmatrix/vmatrix 外包 m:d 定界，cases 左侧 {，matrix 裸 m:m。
func (p *mathParser) parseMatrixEnv(env string) (atomVal, bool) {
	var rows [][]string
	var cur []string
	for {
		p.flushRun() // 单元格文字独立成型，不与格前文字合并
		mark := p.outLen()
		reason, ok := p.parseSeq(stopSet{cell: true})
		if !ok {
			return atomVal{}, false
		}
		cur = append(cur, p.takeFrom(mark))
		switch reason {
		case stopAmp:
			p.pos++ // 消费 &（parseSeq 停止符不消费契约；漏消费=原地死循环，
			// 2026-11-13 pmatrix 测试实锤——runtime 内存打爆即源于此）
			continue
		case stopRowsep:
			p.pos += 2 // 消费 \\
			rows = append(rows, cur)
			cur = nil
			if len(rows) > maxMatrixRows {
				return atomVal{}, false
			}
			continue
		case stopEndEnv:
			rows = append(rows, cur)
			if !p.consumeEnd(env) {
				return atomVal{}, false
			}
		default:
			return atomVal{}, false // 序列意外结束（未遇 \end）
		}
		break
	}
	// 容忍行尾多余的 \\：丢弃末尾全空行。
	for len(rows) > 0 {
		last := rows[len(rows)-1]
		empty := len(last) == 0
		for _, cell := range last {
			if cell != "" {
				empty = false
				break
			}
		}
		if !empty {
			break
		}
		rows = rows[:len(rows)-1]
	}
	if len(rows) == 0 {
		return atomVal{}, false // 完全空环境
	}
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	if cols > maxMatrixCols {
		return atomVal{}, false
	}
	if cols == 0 {
		cols = 1
	}
	// 列对齐：cases 左对齐（同 temml 的 tml-left），其余居中。
	align := "center"
	if env == "cases" {
		align = "left"
	}
	mcs := ""
	for i := 0; i < cols; i++ {
		mcs += `<m:mc><m:mcPr><m:count m:val="1"/><m:mcJc m:val="` + align + `"/></m:mcPr></m:mc>`
	}
	body := ""
	for _, row := range rows {
		body += `<m:mr>`
		for i := 0; i < cols; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			if cell == "" {
				body += `<m:e/>`
			} else {
				body += `<m:e>` + cell + `</m:e>`
			}
		}
		body += `</m:mr>`
	}
	mm := `<m:m><m:mPr><m:mcs>` + mcs + `</m:mcs></m:mPr>` + body + `</m:m>`
	switch env {
	case "pmatrix":
		return atomVal{xml: `<m:d><m:dPr><m:begChr m:val="("/><m:endChr m:val=")"/></m:dPr><m:e>` + mm + `</m:e></m:d>`}, true
	case "bmatrix":
		return atomVal{xml: `<m:d><m:dPr><m:begChr m:val="["/><m:endChr m:val="]"/></m:dPr><m:e>` + mm + `</m:e></m:d>`}, true
	case "vmatrix":
		return atomVal{xml: `<m:d><m:dPr><m:begChr m:val="‖"/><m:endChr m:val="‖"/></m:dPr><m:e>` + mm + `</m:e></m:d>`}, true
	case "cases":
		return atomVal{xml: `<m:d><m:dPr><m:begChr m:val="{"/><m:endChr m:val=""/></m:dPr><m:e>` + mm + `</m:e></m:d>`}, true
	default: // matrix：无定界
		return atomVal{xml: mm}, true
	}
}

// ── OMML 生成 ────────────────────────────────────────────────────────────

// escXML XML 文本/属性转义（同上游 esc：& < > "）。
func escXML(s string) string {
	if !strings.ContainsAny(s, `&<>"`) {
		return s
	}
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// ommlRun 一个 m:r：样式 props（m:rPr）+ Cambria Math（w:rPr）+ m:t，
// 结构与上游 omml.ts 的 run() 一致。
func ommlRun(text, style string) string {
	if text == "" {
		return ""
	}
	props := ""
	switch style {
	case "nor":
		props = `<m:rPr><m:nor/></m:rPr>`
	case "p":
		props = `<m:rPr><m:sty m:val="p"/></m:rPr>`
	case "b":
		props = `<m:rPr><m:sty m:val="b"/></m:rPr>`
	}
	return `<m:r>` + props + ommlFont + `<m:t xml:space="preserve">` + escXML(text) + `</m:t></m:r>`
}

// isASCIILetter / isASCIIDigit 窄字符判定（命令名/数字只认 ASCII）。
func isASCIILetter(r rune) bool { return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') }
func isASCIIDigit(r rune) bool  { return r >= '0' && r <= '9' }
