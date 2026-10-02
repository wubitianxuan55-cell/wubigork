package strutil

// truncate.go —— 截断族原语的单一源（审计 2026-10-02 X1-06 / X1-18 / GA2-11）。
//
// 收敛前全仓有 40 余份 truncate/clip 私有实现，同名不同义（纯截断 / 补 "…" /
// 补 "..." / 取 n-1 补 "…" / n≤0 特例），命名不表达行为，grep 无法判等。本文件
// 只提供三个名字即说明行为的原语：
//
//	TruncateRunes         前 n 个 rune，无后缀，总长 ≤ n
//	TruncateRunesEllipsis 前 n-1 个 rune + "…"，总长恰为 n
//	TruncateRunesSuffix   前 n 个 rune + suffix，总长 ≤ n + rune(suffix)
//
// 三个原语一律先判「未超长」再判 n≤0，与收敛前各站点实现（`if len(r) <= n
// { return s }` 在前）逐字节等价：空串配 n=0 返回空串而不是 suffix。
//
// 站点若还带本地政策（先 TrimSpace、n≤0 特例、截断后再 TrimSpace），保留一层
// 一行薄包装并在注释里写明差别，核心切片逻辑一律走本文件；按字节/按行预算的
// 截断（archive/office 的字节切、memory 文档字节预算等）语义不同，不在此收敛。

// TruncateRunes 按 rune 截断 s 到最多 n 个 rune，不追加任何后缀。
//
//	总长：结果 rune 数 ≤ n（本来就 ≤ n 时原样返回，不加后缀）。
//	边界：n ≤ 0 返回 ""，不 panic（历史实现在 n<0 时切片越界 panic）。
func TruncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 0 {
		return ""
	}
	return string(r[:n])
}

// TruncateRunesEllipsis 按 rune 截断到总长恰好 n（省略号计入 n）。
//
//	总长：未超长（rune 数 ≤ n）时原样返回 s、不补省略号；超长时
//	      返回前 n-1 个 rune + "…"，结果 rune 数恰为 n。
//	边界：n == 1 时放不下省略号，返回首个 rune（保留历史实现语义，不退化成 ""）；
//	      n ≤ 0 返回 ""，不 panic。
func TruncateRunesEllipsis(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 0 {
		return ""
	}
	if n == 1 {
		return string(r[:1])
	}
	return string(r[:n-1]) + "…"
}

// TruncateRunesSuffix 保留前 n 个 rune，再追加 suffix 作为截断标记。
//
//	总长：未超长（rune 数 ≤ n）时原样返回 s、不追加后缀；超长时返回前 n 个
//	      rune + suffix，结果 rune 数 ≤ n + len([]rune(suffix))。
//	边界：n ≤ 0 时只返回 suffix（"保留 0 个 rune"），不 panic；唯一例外是
//	      s 本来就满足 rune 数 ≤ n（空串配 n=0），此时按"未超长"原样返回 s。
func TruncateRunesSuffix(s string, n int, suffix string) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 0 {
		return suffix
	}
	return string(r[:n]) + suffix
}
