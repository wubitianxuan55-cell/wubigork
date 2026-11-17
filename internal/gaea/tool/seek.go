package tool

import "strings"

// 三档行匹配（codex 蒸馏 T1，apply_patch seek_sequence 同构）：编辑工具的
// old_block 在目标文件被无关改动后往往整块失配。按行定位、逐档放宽，
// 让「上下文还在、空白漂了」的编辑仍然命中。
//
// 语义约定：lines / pat 都必须是逻辑行（不含 \r\n 行尾符，调用方负责剥）。
// 档位：0=逐字节相等；1=忽略行尾空白（空格/制表）；2=忽略两侧空白。
// 命中要求整段连续，绝不跨段拼凑。

// FindSequence 在 lines 中自 at 行起找 pat 的最早命中，三档放宽。
// 先全表扫档 0，再档 1、档 2——命中取「最低档位下的最早位置」，同输入同结果。
// 未命中返回 idx=-1, tier=-1。空 pat / at 越界 / pat 长于剩余行一律未命中。
func FindSequence(lines []string, pat []string, at int) (idx, tier int) {
	if len(pat) == 0 || at < 0 || len(pat) > len(lines)-at {
		return -1, -1
	}
	for tier := 0; tier <= 2; tier++ {
		for i := at; i+len(pat) <= len(lines); i++ {
			if matchTier(lines[i:i+len(pat)], pat, tier) {
				return i, tier
			}
		}
	}
	return -1, -1
}

// matchTier 判定窗口 have 与 pat 在指定档位下是否整段连续相等。
func matchTier(have []string, pat []string, tier int) bool {
	for p := range pat {
		a, b := have[p], pat[p]
		switch tier {
		case 0:
			if a != b {
				return false
			}
		case 1:
			if strings.TrimRight(a, " \t") != strings.TrimRight(b, " \t") {
				return false
			}
		default:
			if strings.TrimSpace(a) != strings.TrimSpace(b) {
				return false
			}
		}
	}
	return true
}

// NearestMatches 返回首行与 pat 首行最相似的至多 n 处 1-based 行号，
// 供未命中错误的「最接近的候选」提示。相似度=去空白后的最长公共前缀长度，
// 并列取靠前的行；全无公共前缀时返回空。
func NearestMatches(lines []string, pat []string, n int) []int {
	if len(pat) == 0 || len(lines) == 0 || n <= 0 {
		return nil
	}
	first := strings.TrimSpace(pat[0])
	if first == "" {
		return nil
	}
	type cand struct {
		line  int
		score int
	}
	var cands []cand
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		m := commonPrefixLen(t, first)
		if m > 0 {
			cands = append(cands, cand{line: i + 1, score: m})
		}
	}
	// 分数降序、行号升序——插入排序式选择即可，行数级别无性能压力。
	for i := 1; i < len(cands); i++ {
		for j := i; j > 0 && (cands[j].score > cands[j-1].score ||
			(cands[j].score == cands[j-1].score && cands[j].line < cands[j-1].line)); j-- {
			cands[j], cands[j-1] = cands[j-1], cands[j]
		}
	}
	if len(cands) > n {
		cands = cands[:n]
	}
	if len(cands) == 0 {
		return nil
	}
	out := make([]int, len(cands))
	for i, c := range cands {
		out[i] = c.line
	}
	return out
}

// commonPrefixLen 返回两串的最长公共前缀长度（按字节；仅用于相对比较）。
func commonPrefixLen(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}
