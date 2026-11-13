// formulas.go — 从 ops（任意 JSON 反序列化后的值）提取 $…$ / $$…$$ 公式并
// 预转 OMML。键规则对齐上游 omml.ts formulasOf：
//   - display（独立成行）→ "d:"+source，行内 → "i:"+source；
//   - source = 去掉 $ 定界后 trim → 剥首个 \tag{…} → 再 trim；
//   - 同键已存在跳过（不覆盖）；转换失败不进 map（helper 对查不到的公式
//     fail-soft 保留原文，Word 档完全不受影响）；
//   - 另：名为 "latex" 的字符串字段视为 display 公式（公式卡片写法，剥首尾
//     所有 $ 后入 d: 键）。
//
// 扫描语义对齐上游 DOLLARS 正则 /\$\$([^$\r\n]+?)\$\$|\$([^$\r\n]+?)\$(?!\$)/g：
// $$…$$ 内容不得含 $ 与换行；$…$ 内容不得含 $ 与换行、且闭合 $ 后不得紧跟 $。
// Go regexp（RE2）不支持否定前瞻 (?!\$)，故用手写扫描等价实现。
package omml

import (
	"regexp"
	"strings"
)

// tagRe 剥公式里的 \tag{…}（只剥首个，同上游 TAG 正则语义）。
var tagRe = regexp.MustCompile(`\\tag\s*\{([^}]*)\}`)

// latexEdgeDollars 剥 "latex" 字段首尾的所有 $（同上游 /^\$+|\$+$/g）。
var latexEdgeDollars = regexp.MustCompile(`^\$+|\$+$`)

// FormulasOf 遍历 value（[]any / map[string]any / string / float64 等任意
// JSON 反序列化值），把所有能转成 OMML 的公式收进 found（可为 nil）并返回。
func FormulasOf(value any, found map[string]string) map[string]string {
	if found == nil {
		found = map[string]string{}
	}
	walkFormulas(value, found)
	return found
}

// walkFormulas 递归遍历；字符串走 $ 扫描，数组/映射下钻，
// "latex" 字符串字段按 display 公式处理。
func walkFormulas(value any, found map[string]string) {
	switch v := value.(type) {
	case string:
		if !strings.Contains(v, "$") {
			return // 上游同款短路：不含 $ 的文本免扫
		}
		scanDollars(v, found)
	case []any:
		for _, item := range v {
			walkFormulas(item, found)
		}
	case map[string]any:
		for name, item := range v {
			// 名为 "latex" 的字符串字段：整字段即公式（无 $ 包裹）。
			if name == "latex" {
				if s, isStr := item.(string); isStr {
					addLatexField(s, found)
					continue
				}
			}
			walkFormulas(item, found)
		}
	}
}

// scanDollars 手写扫描，语义等价上游 DOLLARS 正则（含 (?!\$) 否定前瞻——
// RE2 不支持，逐位置模拟正则引擎的两种尝试与失败前进）。
func scanDollars(s string, found map[string]string) {
	n := len(s)
	for i := 0; i < n; {
		if s[i] != '$' {
			i++
			continue
		}
		// 优先按 $$…$$（display）尝试：内容不含 $ \r \n 且至少 1 字符。
		if i+1 < n && s[i+1] == '$' {
			j := i + 2
			for j < n && s[j] != '$' && s[j] != '\r' && s[j] != '\n' {
				j++
			}
			// 闭合 $$ 必须紧跟内容（+? 至少 1 字符 → j > i+2）。
			if j > i+2 && j+1 < n && s[j] == '$' && s[j+1] == '$' {
				addFormula(s[i+2:j], true, found)
				i = j + 2
				continue
			}
			// 该起点两种形态都不成（内容以 $ 开头，单 $ 分支必然失败），
			// 正则引擎前进一格。
			i++
			continue
		}
		// 单 $：内容不含 $ \r \n 且至少 1 字符，闭合 $ 后不得紧跟 $（(?!\$)）。
		j := i + 1
		for j < n && s[j] != '$' && s[j] != '\r' && s[j] != '\n' {
			j++
		}
		if j > i+1 && j < n && s[j] == '$' && (j+1 >= n || s[j+1] != '$') {
			addFormula(s[i+1:j], false, found)
			i = j + 1
			continue
		}
		i++
	}
}

// addFormula 提取并转换一个公式：source=剥定界后 trim → 剥首个 \tag{…} →
// 再 trim；键 d:/i: 前缀；空 source、重复键、转换失败均跳过。
func addFormula(raw string, display bool, found map[string]string) {
	source := strings.TrimSpace(raw)
	if at := tagRe.FindStringIndex(source); at != nil {
		source = source[:at[0]] + source[at[1]:]
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return
	}
	key := "i:" + source
	if display {
		key = "d:" + source
	}
	if _, dup := found[key]; dup {
		return
	}
	if made, ok := LaTeXToOMML(source, display); ok {
		found[key] = made
	}
}

// addLatexField "latex" 字段：trim → 剥首尾所有 $ → trim，按 display 入 d: 键
// （上游同款：此处不剥 \tag）。
func addLatexField(raw string, found map[string]string) {
	source := strings.TrimSpace(raw)
	source = latexEdgeDollars.ReplaceAllString(source, "")
	source = strings.TrimSpace(source)
	if source == "" {
		return
	}
	key := "d:" + source
	if _, dup := found[key]; dup {
		return
	}
	if made, ok := LaTeXToOMML(source, true); ok {
		found[key] = made
	}
}
