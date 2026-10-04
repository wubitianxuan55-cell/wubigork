package strutil

import "unicode"

// ContainsHan reports whether s contains at least one Han (Chinese) character.
//
// 收敛自两份逐字相同的本地实现（2026-10-04 第三轮审计 §3.2）：costimport.hasHan
// 与 knowledge.isCJStr（后者名不符实——名字说 CJK，实际只判 Han，此处取诚实名）。
// 语义冻结：逐 rune 判 unicode.Is(unicode.Han, r)，命中即真。
func ContainsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
