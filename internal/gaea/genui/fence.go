package genui

import (
	"regexp"
	"strings"
)

// GenUI 围栏剥离（记忆/压缩侧共享，审计 docs/gaea-genui-memoryfence-audit-2026-09.md
// #1/#2/#3/#4 的公共底座）。
//
// 语义照搬前端 frontend/src/genui/parse.ts splitGenuiFences 的行扫描：
//   - 开栏行：行首 ``` 加语言标记（genui / dsh-ui），行尾可有空白（正则
//     ^```([\w-]+)\s*$，与前端同款，行中间出现不算开栏）；
//   - 闭栏行：^\s*```\s*$（与前端同款，允许缩进与尾随空白）；
//   - 围栏体内再出现 ```lang 也不重新开栏（与前端一致，markdown 围栏不嵌套）；
//   - 未闭合围栏：折叠到文本末尾。
//
// 折叠口径：开栏行 + 围栏体 + 闭栏行整体替换为一行占位 UIFencePlaceholder，
// **不保留围栏标记行**——残留的 ``` 会与后续正文里的代码块误配对成新围栏。
// 围栏外正文逐字保留（Split/Join 往返字节无损）。只允许用于「喂给提炼/摘要/
// 记忆」的输入；会话正文、压缩保留尾部、子代理转录、导出一律不剥。

// UIFencePlaceholder 围栏体折叠后的占位行。
const UIFencePlaceholder = "[genui 组件]"

// 围栏语言与前端 spec.ts GENUI_FENCE_LANGS 同源（改动须两边同步）。
func isUIFenceLang(lang string) bool {
	return lang == "genui" || lang == "dsh-ui"
}

var (
	// 开栏行：```lang（行首，无缩进；行尾空白容忍）——与前端 parse.ts 同款。
	uiFenceOpenRe = regexp.MustCompile("^```([\\w-]+)\\s*$")
	// 闭栏行：允许缩进与尾随空白——与前端 parse.ts 同款。
	uiFenceCloseRe = regexp.MustCompile("^\\s*```\\s*$")
)

// StripUIFences 把文本里的 genui / dsh-ui 围栏整体折叠为一行占位，围栏外
// 正文逐字保留。无围栏时返回值与输入逐字节相同（无反引号走快路径原样返回）。
func StripUIFences(text string) string {
	if !strings.Contains(text, "```") {
		return text // 快路径：无反引号必无围栏
	}
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inFence := false
	for _, line := range lines {
		if inFence {
			// 围栏体内：只认闭栏行收尾，其余整行丢弃（含疑似开栏行）。
			if uiFenceCloseRe.MatchString(line) {
				inFence = false
			}
			continue
		}
		if m := uiFenceOpenRe.FindStringSubmatch(line); m != nil && isUIFenceLang(m[1]) {
			inFence = true
			out = append(out, UIFencePlaceholder)
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
