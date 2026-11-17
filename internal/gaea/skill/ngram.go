package skill

import "strings"

// 字符 n-gram 路（codex 蒸馏 T14，character_ngram 同构）：2–5 字符窗口在
// 字符级匹配，不依赖分词——中文无空格，「导出表格」与「表格导出」能部分命中。
// 字段权重 8/4/1（名/描述/正文），分数为共享 gram 数 × 字段权重的整数和
// （上游浮点 8.0/4.0/1.0 在整数域等价，省一次类型转换）。查询超 4KiB 截断。

// ngram 路硬上限（与词法路一致，全部截断不报错）。
const (
	ngramMinLen    = 2
	ngramMaxLen    = 5
	ngramMaxQLen   = 4 << 10
	weightNameGram = 8
	weightDescGram = 4
	weightBodyGram = 1
)

// charGrams 取字符串的 2–5 字符 gram 集合（rune 级，小写归一，去重）。
func charGrams(s string) map[string]bool {
	runes := []rune(strings.ToLower(s))
	out := map[string]bool{}
	for i := 0; i < len(runes); i++ {
		for l := ngramMinLen; l <= ngramMaxLen && i+l <= len(runes); l++ {
			out[string(runes[i:i+l])] = true
		}
	}
	return out
}

// ngramScore 计算查询与技能各字段的共享 gram 分：Σ 权重 × 共享 gram 数。
// 字段顺序固定（名→描述→正文），同输入同结果。返回分数与查询截断标志。
func ngramScore(queryGrams map[string]bool, queryBytes int, name, desc, body string) (uint32, bool) {
	truncated := queryBytes > ngramMaxQLen
	score := uint32(0)
	for _, fc := range []struct {
		text   string
		weight int
	}{{name, weightNameGram}, {desc, weightDescGram}, {body, weightBodyGram}} {
		dg := charGrams(fc.text)
		for g := range queryGrams {
			if dg[g] {
				score += uint32(fc.weight)
			}
		}
	}
	return score, truncated
}
