package strutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// truncate_test.go —— X1-06 / X1-18 截断族收敛的等价性锁。
//
// 本文件把收敛期的「对拍快照」固化为表驱动用例：old* 系列是收敛前各站点私有
// 实现的逐字复制（参照实现），用 Inputs × Ns 全矩阵断言 strutil 原语与旧实现
// 逐字节相同。**不可达的 n** 不靠"差不多"，而是逐站点用 reachable 声明可达域，
// 域外的行为单列在 TestUnreachableNBoundaries 里断言（旧实现 panic / 语义不同）。
//
// 站点 → 原语映射（收敛后：能 1:1 的调用点直呼 strutil，带本地政策的留薄包装）：
//
//	原语                 站点（旧名）                                         旧行为
//	TruncateRunes        characterstate/wssearch.truncateRunes               r[:n]，无守卫（n<0 panic）
//	TruncateRunes        pins/bookimport.truncateRunes                       r[:n]，n≤0→""
//	TruncateRunes        costimport.truncateRunes（薄包装，先 TrimSpace）      TrimSpace 后 r[:n]
//	TruncateRunesEllipsis skill.clipRunes（薄包装，max<1→1）                    r[:max-1]+"…"，总长恰为 max
//	TruncateRunesEllipsis memory/truncateMorningDesc 等                       r[:N-1]+"…"
//	TruncateRunesSuffix  analysis.truncateRunes / util.Truncate              r[:n]+"..."
//	TruncateRunesSuffix  weixin/whisper.truncateRunes|truncateStr|truncateString r[:n]+"…"
//	TruncateRunesSuffix  hook.clipRunes（薄包装，max<1→""）                    r[:max]+"…"
//	TruncateRunes        tool/builtin.truncate（薄包装，先 TrimSpace）         **原为字节切 s[:n]（GA2-10 修正）**

// snapshotInputs 是边界输入集：空串 / ASCII / 恰好长度 / 中文 / emoji / 混合 /
// 首尾空白 / 超长（含中文与 emoji）。
func snapshotInputs() []string {
	return []string{
		"",
		"abc",
		"abcde",
		"abcdefghij",
		"中文测试",
		"中文测试文本内容很长很长",
		"a中b文c\u200d🌙emoji🎉x",
		"  hello  ",
		"\n换行开头的中文内容，用于测试截断行为\n",
		strings.Repeat("中a", 200) + "🎉",
		strings.Repeat("🎉", 300),
		strings.Repeat("x", 400),
	}
}

// snapshotNs 是边界 n 集：-1 / 0 / 1 / n-1 / n / n+1 与各站点实际调用的字面量。
func snapshotNs() []int {
	return []int{-1000, -1, 0, 1, 2, 3, 4, 5, 6, 9, 10, 11, 29, 30, 31,
		39, 40, 41, 49, 50, 51, 59, 60, 61, 79, 80, 81, 99, 100, 101,
		119, 120, 121, 159, 160, 161, 199, 200, 201, 299, 300, 301,
		399, 400, 401, 511, 512, 513, 1599, 1600, 1601, 7999, 8000, 1000000}
}

// ── 收敛前站点的旧实现（逐字复制，参照实现）──

// oldPure 是 characterstate/wssearch 的 truncateRunes：无 n≤0 守卫（n<0 时 panic）。
func oldPure(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// oldPureGuarded 是 pins 的 truncateRunes：r[:n]，n≤0 → ""。
func oldPureGuarded(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 0 {
		return ""
	}
	return string(r[:n])
}

// oldBookimportGuarded 是 bookimport 的 truncateRunes：max≤0 → ""，再 r[:max]。
func oldBookimportGuarded(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// oldTrimmedPure 是 costimport 的 truncateRunes：先 TrimSpace 再纯截断。
func oldTrimmedPure(s string, n int) string {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) <= n {
		return string(rs)
	}
	return string(rs[:n])
}

// oldSuffixDots 是 analysis/util 的旧实现：r[:n] + "..."。
func oldSuffixDots(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// oldSuffixEllipsis 是 weixin/whisper 的旧实现：r[:n] + "…"。
func oldSuffixEllipsis(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// oldClipHook 是 hook.clipRunes：max<1 → ""，否则 r[:max] + "…"。
func oldClipHook(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max < 1 {
		return ""
	}
	return string(r[:max]) + "…"
}

// oldClipSkill 是 skill.clipRunes：max<1 按 1 处理，总长恰为 max。
func oldClipSkill(s string, max int) string {
	if max < 1 {
		max = 1
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max-1 < 1 {
		return string(r[:1])
	}
	return string(r[:max-1]) + "…"
}

// oldEllipsisN 是 app/gaea_subagents 与 memory/truncateMorningDesc 的 r[:n-1]+"…"。
func oldEllipsisN(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// oldByteCut 是 tool/builtin 收敛前的 websearch.truncate：**按字节**切（GA2-10）。
func oldByteCut(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// ── 收敛后的站点表达式（薄包装 = 本地政策 + strutil 原语）──

func siteHookClip(s string, max int) string {
	if max < 1 {
		return ""
	}
	return TruncateRunesSuffix(s, max, "…")
}

func siteSkillClip(s string, max int) string {
	if max < 1 {
		max = 1
	}
	return TruncateRunesEllipsis(s, max)
}

func siteCostTrimmed(s string, n int) string {
	return TruncateRunes(strings.TrimSpace(s), n)
}

func siteBuiltinTruncate(s string, maxLen int) string {
	return TruncateRunes(strings.TrimSpace(s), maxLen)
}

func siteSuffix(suffix string) func(string, int) string {
	return func(s string, n int) string { return TruncateRunesSuffix(s, n, suffix) }
}

func panics(f func()) (p bool) {
	defer func() {
		if recover() != nil {
			p = true
		}
	}()
	f()
	return false
}

// siteMapping 是站点 → 原语的等价性对照。
// reachable 声明该站点可达的 n 域（调用点的实参来源见注释），域外行为在
// TestUnreachableNBoundaries 单列断言——不可达不当作"等价"的借口。
type siteMapping struct {
	site      string
	old       func(string, int) string
	new       func(string, int) string
	reachable func(n int) bool
}

func alwaysReachable(int) bool { return true }
func positiveN(n int) bool     { return n >= 1 }
func nonNegativeN(n int) bool  { return n >= 0 }
func atLeastTwoN(n int) bool   { return n >= 2 }

func siteMappings() []siteMapping {
	return []siteMapping{
		// 调用点实参：characterstate.go:115 字面量 50；wssearch.go:137 maxDocRunes=200000、
		// wssearch.go:384 字面量 120 → 全为正数，n<0 不可达（旧实现该域 panic）。
		{"characterstate/wssearch.truncateRunes → TruncateRunes", oldPure, TruncateRunes, positiveN},
		// pins.go:177 字面量 maxFileRunes=1600；pins.go:179 maxTotalRunes-total **可为 0 或负**
		// → 全域（含负数）必须逐字节等价，故 reachable=always。
		{"pins.truncateRunes → TruncateRunes", oldPureGuarded, TruncateRunes, alwaysReachable},
		// bookimport：标题截断（306 titleMaxRunes / 412,426 字面量 40）+ reconstruct 的字面量
		// 与 *MaxRunes 常量（1000/100/60/…），全为正 → n≤0 不可达（旧实现 max≤0→"" 亦然）。
		{"bookimport.truncateRunes → TruncateRunes", oldBookimportGuarded, TruncateRunes, positiveN},
		// costimport 7 处调用点均为正字面量（3000/40/30/80/160），n<0 不可达。
		{"costimport.truncateRunes(TrimSpace 薄包装) → TruncateRunes", oldTrimmedPure, siteCostTrimmed, positiveN},
		// analysis.go:236 字面量 50；util.Truncate 调用点含 create_chapter_context.go:105 的
		// budget-3（可为 0）→ n≥0 全域等价。
		{"analysis.truncateRunes/util.Truncate → TruncateRunesSuffix(..., \"...\")", oldSuffixDots, siteSuffix("..."), nonNegativeN},
		// weixin 120/300/512/200；whisper 60/100/120/200/400/1200，全为正 → n<0 不可达。
		{"weixin/whisper.truncateRunes|truncateStr|truncateString → TruncateRunesSuffix(..., \"…\")", oldSuffixEllipsis, siteSuffix("…"), positiveN},
		// hook 唯一调用点传字面量 60；薄包装保留 max<1→""，故全域等价。
		{"hook.clipRunes(max<1→\"\") → TruncateRunesSuffix(..., \"…\")", oldClipHook, siteHookClip, alwaysReachable},
		// skill 调用点 base=130-len(name)-len(tag) 可为负；薄包装保留 max<1→1，故全域等价。
		{"skill.clipRunes(max<1→1) → TruncateRunesEllipsis", oldClipSkill, siteSkillClip, alwaysReachable},
		// memory/truncateMorningDesc 用常量 morningDescRunes=120；app/gaea_subagents 用
		// 40/80/120/160/200 → n≥2 可达；n==1 时旧实现给 "…"、原语给首 rune，域外。
		{"memory.truncateMorningDesc/gaea_subagents → TruncateRunesEllipsis", oldEllipsisN, TruncateRunesEllipsis, atLeastTwoN},
	}
}

// TestSiteMappingEquivalence 用全边界矩阵断言「收敛前后同一输入输出逐字节相同」
// （各站点可达域内）。任一站点在任一格不等价即失败——这是收敛的硬证据。
func TestSiteMappingEquivalence(t *testing.T) {
	for _, m := range siteMappings() {
		for _, in := range snapshotInputs() {
			for _, n := range snapshotNs() {
				if !m.reachable(n) {
					continue
				}
				want := m.old(in, n)
				got := m.new(in, n)
				if got != want {
					t.Errorf("%s\n  in=%q n=%d\n  old=%q\n  new=%q", m.site, in, n, want, got)
				}
			}
		}
	}
}

// TestUnreachableNBoundaries 把每个站点域外的 n 行为单列断言：要么旧实现 panic
// （收敛后原语给确定值，属有意偏离），要么语义本就不同（app 的 n==1 → "…"）。
func TestUnreachableNBoundaries(t *testing.T) {
	// n<0：无守卫的旧实现切片越界 panic（对拍快照 12 行 PANIC 的来源）；
	// 收敛后一律返回确定值，绝不 panic。
	for name, f := range map[string]func(){
		"characterstate/wssearch.truncateRunes": func() { _ = oldPure("abc", -1) },
		"costimport.truncateRunes":              func() { _ = oldTrimmedPure("abc", -1) },
		"analysis/util.Truncate":                func() { _ = oldSuffixDots("abc", -1) },
		"weixin/whisper.r[:n]+\"…\"":            func() { _ = oldSuffixEllipsis("abc", -1) },
	} {
		if !panics(f) {
			t.Errorf("%s：参照实现在 n=-1 时应 panic", name)
		}
	}
	for _, in := range snapshotInputs() {
		for _, n := range []int{-1000, -1} {
			if got := TruncateRunes(in, n); got != "" {
				t.Errorf("TruncateRunes(%q, %d) = %q, want \"\"", in, n, got)
			}
			if got := TruncateRunesEllipsis(in, n); got != "" {
				t.Errorf("TruncateRunesEllipsis(%q, %d) = %q, want \"\"", in, n, got)
			}
			if got := TruncateRunesSuffix(in, n, "…"); got != "…" {
				t.Errorf("TruncateRunesSuffix(%q, %d) = %q, want \"…\"", in, n, got)
			}
		}
	}
	// n==1：app/gaea_subagents.go:285 的 r[:n-1]+"…" 给 "…"，skill 的口径给首 rune；
	// 二者语义本就分叉，各自站点只走自己的可达域（app 传 120 等字面量，skill 会钳到 1）。
	if got := oldEllipsisN("abc", 1); got != "…" {
		t.Errorf("reference oldEllipsisN(\"abc\", 1) = %q, want \"…\"", got)
	}
	if got := TruncateRunesEllipsis("abc", 1); got != "a" {
		t.Errorf("TruncateRunesEllipsis(\"abc\", 1) = %q, want \"a\"（skill 口径）", got)
	}
	if got := siteSkillClip("abc", 1); got != oldClipSkill("abc", 1) {
		t.Errorf("skill 薄包装在 n==1 与旧实现不等价：%q vs %q", got, oldClipSkill("abc", 1))
	}
}

// TestTruncateRunesBoundaries 钉死 TruncateRunes 的总长/边界口径。
func TestTruncateRunesBoundaries(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"空串 n=0 原样", "", 0, ""},
		{"空串 n=-1 不 panic", "", -1, ""},
		{"未超长原样返回", "abc", 3, "abc"},
		{"超长取前 n rune", "abcde", 3, "abc"},
		{"中文按 rune 计数", "中文测试", 2, "中文"},
		{"emoji 不切裂", "🎉🎉🎉", 2, "🎉🎉"},
		{"n=0 非空返回空", "abc", 0, ""},
		{"n=-1 返回空", "abc", -1, ""},
		{"总长恰为 n", strings.Repeat("中", 10), 10, strings.Repeat("中", 10)},
	}
	for _, c := range cases {
		if got := TruncateRunes(c.in, c.n); got != c.want {
			t.Errorf("%s: TruncateRunes(%q, %d) = %q, want %q", c.name, c.in, c.n, got, c.want)
		}
		if n := utf8.RuneCountInString(TruncateRunes(c.in, c.n)); n > c.n && c.n > 0 {
			t.Errorf("%s: 总长 %d 超过 n=%d", c.name, n, c.n)
		}
	}
}

// TestTruncateRunesEllipsisBoundaries 钉死「总长恰为 n」口径与 n==1 特例。
func TestTruncateRunesEllipsisBoundaries(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"未超长不补省略号", "abc", 3, "abc"},
		{"未超长（n+1 边界）", "abcd", 5, "abcd"},
		{"超长总长恰为 n", "abcde", 3, "ab…"},
		{"n=1 放不下省略号取首 rune", "abc", 1, "a"},
		{"n=1 单 rune 原样", "a", 1, "a"},
		{"n=0 返回空", "abc", 0, ""},
		{"n=-1 返回空", "abc", -1, ""},
		{"空串 n=0 原样", "", 0, ""},
		{"中文总长恰为 n", "中文测试文本", 4, "中文测…"},
		{"emoji 不切裂", "🎉🎉🎉🎉", 3, "🎉🎉…"},
	}
	for _, c := range cases {
		got := TruncateRunesEllipsis(c.in, c.n)
		if got != c.want {
			t.Errorf("%s: TruncateRunesEllipsis(%q, %d) = %q, want %q", c.name, c.in, c.n, got, c.want)
		}
		if c.n > 0 && utf8.RuneCountInString(c.in) > c.n && utf8.RuneCountInString(got) != c.n {
			t.Errorf("%s: 截断后总长应为 %d，实为 %d", c.name, c.n, utf8.RuneCountInString(got))
		}
	}
}

// TestTruncateRunesSuffixBoundaries 钉死「总长 ≤ n + rune(suffix)」口径。
func TestTruncateRunesSuffixBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		n      int
		suffix string
		want   string
	}{
		{"未超长不追加后缀", "abc", 3, "…", "abc"},
		{"超长前 n rune + 后缀", "abcde", 3, "…", "abc…"},
		{"后缀计入总长（n+rune(suffix)）", "abcde", 3, "...", "abc..."},
		{"n=0 只返回后缀", "abc", 0, "…", "…"},
		{"n=0 空串按未超长原样", "", 0, "…", ""},
		{"n=-1 只返回后缀", "abc", -1, "…", "…"},
		{"中文", "中文测试文本", 2, "…", "中文…"},
		{"emoji 不切裂", "🎉🎉🎉", 2, "…", "🎉🎉…"},
	}
	for _, c := range cases {
		if got := TruncateRunesSuffix(c.in, c.n, c.suffix); got != c.want {
			t.Errorf("%s: TruncateRunesSuffix(%q, %d, %q) = %q, want %q", c.name, c.in, c.n, c.suffix, got, c.want)
		}
	}
}

// TestWebsearchByteCutRegression 是 GA2-10 的回归线：旧实现按字节切会把中文/
// emoji 切在 rune 中间（产出非法 UTF-8，JSON 编码后退化为 U+FFFD）；收敛后的
// siteBuiltinTruncate 必须始终输出合法 UTF-8 且不含 U+FFFD。
func TestWebsearchByteCutRegression(t *testing.T) {
	const maxLen = 300
	inputs := []string{
		strings.Repeat("中文摘要内容", 100),
		"a中b文c🎉" + strings.Repeat("混排内容", 200),
		"  " + strings.Repeat("🎉", 300) + "  ",
	}
	broke := false
	for _, in := range inputs {
		if old := oldByteCut(in, maxLen); !utf8.ValidString(old) {
			broke = true // 旧实现确实切裂了 rune（这就是 GA2-10）
		}
		got := siteBuiltinTruncate(in, maxLen)
		if !utf8.ValidString(got) {
			t.Errorf("截断结果不是合法 UTF-8：%q", got)
		}
		if strings.ContainsRune(got, '\uFFFD') {
			t.Errorf("截断结果含 U+FFFD：%q", got)
		}
		if n := utf8.RuneCountInString(got); n > maxLen {
			t.Errorf("结果 rune 数 %d 超过上限 %d", n, maxLen)
		}
		if want := strings.TrimSpace(in); utf8.RuneCountInString(want) <= maxLen && got != want {
			t.Errorf("未超长时应原样返回（已 TrimSpace）：got %q", got)
		}
	}
	if !broke {
		t.Error("旧字节切实现未复现切裂（用例强度不足）")
	}
}

// TestTruncateNeverSplitsRune 全矩阵守住「绝不切裂 rune」。
func TestTruncateNeverSplitsRune(t *testing.T) {
	for _, in := range snapshotInputs() {
		for _, n := range snapshotNs() {
			for _, got := range []string{
				TruncateRunes(in, n),
				TruncateRunesEllipsis(in, n),
				TruncateRunesSuffix(in, n, "…"),
			} {
				if !utf8.ValidString(got) {
					t.Fatalf("in=%q n=%d 产出非法 UTF-8: %q", in, n, got)
				}
			}
		}
	}
}
