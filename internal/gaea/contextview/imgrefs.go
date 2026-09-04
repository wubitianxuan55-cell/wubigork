package contextview

import (
	"encoding/json"
	"sort"
	"strings"
)

// 图片引用提取（2.5b 后半）：gaea 的事件日志不落图片 payload，图片只以
// 文件路径形态出现——用户消息的 @附件引用、识图工具的 image_path 参数、
// 生成类工具输出里的产物路径。本文件把这些引用从自由文本/参数 JSON 里
// 确定性提出来，供详情缩略卡展示。

// maxImageRefs 单节点缩略卡上限（防长输出刷屏；诚实截断不展示更多）。
const maxImageRefs = 4

// imgExtRe 图片扩展名（与前端 IMAGE_EXT_RE 同族；ico 供 favicon 类产物）。
// 大小写不敏感，要求路径以扩展名收尾。
const imgExtSuffix = ".png|.jpg|.jpeg|.gif|.webp|.bmp|.svg|.ico"

// ExtractImageRefs 从自由文本提取图片文件引用：按空白/引号/括号切词、
// 剥 @ 前缀与尾标点，以图片扩展名收尾者入选；去重保序，上限 maxImageRefs。
// 口径与前端 @引用一致——含空格的路径在 gaea 自身 @语法下也无法表达，
// 故按无空格 token 匹配不是近似而是同约束。
func ExtractImageRefs(text string) []string {
	if text == "" {
		return nil
	}
	split := func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', '"', '\'', '(', ')', '[', ']', '<', '>', '`',
			// 中文全角标点（中文正文里路径常直接黏标点，如「out\a.png，」）。
			'，', '。', '；', '：', '！', '？', '、', '（', '）', '【', '】', '《', '》', '“', '”', '‘', '’':
			return true
		}
		return false
	}
	seen := map[string]bool{}
	var refs []string
	for _, tok := range strings.FieldsFunc(text, split) {
		tok = strings.TrimPrefix(tok, "@")
		tok = strings.TrimRight(tok, ".,;:!?")
		if !hasImageExt(tok) || seen[tok] {
			continue
		}
		seen[tok] = true
		refs = append(refs, tok)
		if len(refs) >= maxImageRefs {
			break
		}
	}
	return refs
}

// ExtractImageRefsFromArgs 从工具参数 JSON 提取图片引用：递归遍历全部
// 字符串值，以图片扩展名收尾者入选。参数不是合法 JSON 时退化为自由文本
// 提取（原始 JSON 里的 \\ 双反斜杠会导致路径失配，宁缺勿错）。
func ExtractImageRefsFromArgs(args string) []string {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(args), &v); err != nil {
		return nil // 参数非法 JSON：不用自由文本兜底（防 \\ 转义失配）
	}
	seen := map[string]bool{}
	var refs []string
	var walk func(x any)
	walk = func(x any) {
		if len(refs) >= maxImageRefs {
			return
		}
		switch t := x.(type) {
		case string:
			if hasImageExt(t) && !seen[t] {
				seen[t] = true
				refs = append(refs, t)
			}
		case []any:
			for _, e := range t {
				walk(e)
			}
		case map[string]any:
			// 键名稳定排序，保证同一参数对象提取顺序确定（测试可断言）。
			for _, k := range sortedKeys(t) {
				walk(t[k])
			}
		}
	}
	walk(v)
	return refs
}

// mergeImageRefs 合并多来源引用（去重保序，总上限 maxImageRefs）。
func mergeImageRefs(lists ...[]string) []string {
	seen := map[string]bool{}
	var refs []string
	for _, l := range lists {
		for _, r := range l {
			if seen[r] {
				continue
			}
			seen[r] = true
			refs = append(refs, r)
			if len(refs) >= maxImageRefs {
				return refs
			}
		}
	}
	return refs
}

// countImageRefs 统计一条文本/参数里的图片引用数（stats.Images 口径：引用
// 出现次数，按 ExtractImageRefs 去重后计数、单条上限 4——诚实计数不猜语义）。
func countImageRefs(text string) int {
	return len(ExtractImageRefs(text))
}

func hasImageExt(p string) bool {
	lp := strings.ToLower(p)
	for _, ext := range strings.Split(imgExtSuffix, "|") {
		if strings.HasSuffix(lp, ext) && len(lp) > len(ext) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
