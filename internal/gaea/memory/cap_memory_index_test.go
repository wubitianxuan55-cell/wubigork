package memory

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestCapMemoryIndexWithinBudgetUnchanged：预算内（len <= 4096 字节，含恰好等于
// 预算与空串）原样返回，不追加提示后缀。
func TestCapMemoryIndexWithinBudgetUnchanged(t *testing.T) {
	inputs := []string{
		"",
		"- 一条小记忆\n",
		strings.Repeat("a", memoryIndexBudgetBytes), // 恰好等于预算（边界值）
	}
	for _, in := range inputs {
		if got := capMemoryIndex(in); got != in {
			t.Errorf("预算内输入应原样返回:\n  in =%q\n  got=%q", in, got)
		}
	}
}

// TestCapMemoryIndexLineBoundary：截断按行边界，不出现半行；结果的每一行都与
// 原文对应行完全一致（结果是原文完整行的前缀），且末行不含尾随换行。
func TestCapMemoryIndexLineBoundary(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "- 条目%03d：%s\n", i, strings.Repeat("记", 7)) // 每行 36 字节
	}
	in := b.String()
	if len(in) <= memoryIndexBudgetBytes {
		t.Fatalf("测试输入应超字节预算: %d", len(in))
	}
	res := capMemoryIndex(in)
	if !strings.HasSuffix(res, memoryIndexHint) {
		t.Fatalf("超预算输入应追加提示后缀: %q", res)
	}
	body := strings.TrimSuffix(res, memoryIndexHint)
	if body == "" || strings.HasSuffix(body, "\n") {
		t.Fatalf("body 应以完整行结束且无尾随换行: %q", body)
	}
	origLines := strings.Split(in, "\n")
	gotLines := strings.Split(body, "\n")
	if len(gotLines) > len(origLines) {
		t.Fatalf("结果行数超过原文: %d > %d", len(gotLines), len(origLines))
	}
	for i, gl := range gotLines {
		if gl != origLines[i] {
			t.Fatalf("第 %d 行与原不一致: %q != %q", i, gl, origLines[i])
		}
	}
	if len(body) > memoryIndexBudgetBytes {
		t.Fatalf("截断后超出字节预算: %d > %d", len(body), memoryIndexBudgetBytes)
	}
}

// TestCapMemoryIndexByteBudget：截断结果（含提示后缀）字节数 ≤ 4096 + 提示长度
// + 1 容差；提示后缀确实被追加。
func TestCapMemoryIndexByteBudget(t *testing.T) {
	line := "- 记忆条目：" + strings.Repeat("内容", 20) + "\n"
	var b strings.Builder
	for i := 0; i < 300; i++ {
		b.WriteString(line)
	}
	in := b.String()
	if len(in) <= memoryIndexBudgetBytes {
		t.Fatalf("测试输入应超预算: %d", len(in))
	}
	res := capMemoryIndex(in)
	if !strings.HasSuffix(res, memoryIndexHint) {
		t.Fatalf("超预算应追加提示: %q", res)
	}
	limit := memoryIndexBudgetBytes + len(memoryIndexHint) + 1
	if n := len(res); n > limit {
		t.Fatalf("结果超出预算+提示+容差: %d > %d", n, limit)
	}
}

// TestCapMemoryIndexRuneByteMismatch：中文/emoji 密集索引按字节截断。旧口径按
// 3000 rune 判断不会截断本例（rune 数 < 3000），新口径按字节必须截断；结果
// 合法 UTF-8（不产生半个 rune）且字节数不超预算。
func TestCapMemoryIndexRuneByteMismatch(t *testing.T) {
	line := "记忆条目🔍" + strings.Repeat("记", 6) + "\n" // 每行 35 字节 / 12 rune
	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString(line)
	}
	in := b.String()
	if utf8.RuneCountInString(in) >= 3000 {
		t.Fatalf("测试输入 rune 数应少于旧 3000 口径（以证明按字节截断）: %d", utf8.RuneCountInString(in))
	}
	if len(in) <= memoryIndexBudgetBytes {
		t.Fatalf("测试输入应超字节预算: %d", len(in))
	}
	res := capMemoryIndex(in)
	if !utf8.ValidString(res) {
		t.Fatalf("结果不是合法 UTF-8: %q", res)
	}
	if !strings.HasSuffix(res, memoryIndexHint) {
		t.Fatalf("应追加提示: %q", res)
	}
	body := strings.TrimSuffix(res, memoryIndexHint)
	if len(body) > memoryIndexBudgetBytes {
		t.Fatalf("截断后超出字节预算: %d > %d", len(body), memoryIndexBudgetBytes)
	}
	// 按行边界截断，body 的每一行仍是原文的完整行
	origLines := strings.Split(in, "\n")
	gotLines := strings.Split(body, "\n")
	if len(gotLines) > len(origLines) {
		t.Fatalf("结果行数超过原文: %d > %d", len(gotLines), len(origLines))
	}
	for i, gl := range gotLines {
		if gl != origLines[i] {
			t.Fatalf("第 %d 行与原不一致: %q != %q", i, gl, origLines[i])
		}
	}
}

// TestCapMemoryIndexLinkIntegrity：截断不产生悬空的 markdown 链接——结果要么
// 包含完整链接行（"[...](url)" 成对），要么链接行整体被舍弃。
func TestCapMemoryIndexLinkIntegrity(t *testing.T) {
	t.Run("单行链接在预算内则完整保留", func(t *testing.T) {
		in := "line1\nline2\n[标题](https://example.com/abc)\nline4\nline5\n"
		// budget=50：预算内最后一个 '\n' 是链接行结尾（pos 44），完整链接行保留
		got := truncateIndexByLines(in, 50)
		if !strings.HasSuffix(got, "[标题](https://example.com/abc)") {
			t.Fatalf("完整链接行应保留: %q", got)
		}
		if hasUnclosedLink(got) {
			t.Fatalf("不应有未闭合链接: %q", got)
		}
	})
	t.Run("截断点切进链接行则整行舍弃", func(t *testing.T) {
		in := "line1\nline2\n[标题](https://example.com/abc)\nline4\nline5\n"
		// budget=30：预算内最后 '\n' 在 line2 结尾（pos 11），链接行在截断点之后
		got := truncateIndexByLines(in, 30)
		if strings.Contains(got, "[") {
			t.Fatalf("链接行应整体舍弃: %q", got)
		}
		if hasUnclosedLink(got) {
			t.Fatalf("不应有未闭合链接: %q", got)
		}
	})
	t.Run("多行链接被截断则回退到链接起始行之前", func(t *testing.T) {
		// budget=27：预算内最后 '\n' 在链接首行结尾（pos 26），摘录含
		// "[多行](https://x/aaaa" 但 ')' 在 pos 32 被截掉 → 未闭合 → 回退到
		// 链接起始行之前，只留 "l1"
		in := "l1\n[多行](https://x/aaaa\nbbbb)\nl2\n"
		got := truncateIndexByLines(in, 27)
		if got != "l1" {
			t.Fatalf("多行链接未闭合应回退到链接起始行之前, got %q", got)
		}
		if hasUnclosedLink(got) {
			t.Fatalf("不应有未闭合链接: %q", got)
		}
	})
	t.Run("多行链接完整则保留", func(t *testing.T) {
		// budget=34：预算内最后 '\n' 在 "bbbb)" 之后（pos 33），完整多行链接保留
		in := "l1\n[多行](https://x/aaaa\nbbbb)\nl2\n"
		got := truncateIndexByLines(in, 34)
		if !strings.HasSuffix(got, "[多行](https://x/aaaa\nbbbb)") {
			t.Fatalf("完整多行链接应保留: %q", got)
		}
		if hasUnclosedLink(got) {
			t.Fatalf("不应有未闭合链接: %q", got)
		}
	})
	t.Run("超预算索引经 capMemoryIndex 链接要么完整要么舍弃", func(t *testing.T) {
		mk := func(linkAt int) string {
			var b strings.Builder
			for i := 0; i < 130; i++ {
				if i == linkAt {
					fmt.Fprintf(&b, "[标题](https://example.com/url-%d)\n", linkAt)
				} else {
					fmt.Fprintf(&b, "- 条目%03d：%s\n", i, strings.Repeat("记", 7))
				}
			}
			return b.String()
		}
		// 链接在第 5 行（截断点之前）：完整保留
		in1 := mk(4)
		res1 := capMemoryIndex(in1)
		if !strings.Contains(res1, "[标题](https://example.com/url-4)") {
			t.Fatalf("预算内的完整链接行应保留: %q", res1)
		}
		if hasUnclosedLink(res1) {
			t.Fatalf("不应有未闭合链接: %q", res1)
		}
		// 链接在最后一行（截断点之后）：整行舍弃，且不产生悬空链接
		in2 := mk(129)
		res2 := capMemoryIndex(in2)
		if strings.Contains(res2, "url-129") {
			t.Fatalf("截断点之后的链接行应整体舍弃: %q", res2)
		}
		if hasUnclosedLink(res2) {
			t.Fatalf("不应有未闭合链接: %q", res2)
		}
	})
}

// TestCapMemoryIndexSingleLineOverBudget：单行超预算且预算内无换行时，整行舍弃
// （宁丢整行，不切半个字）并追加提示。这是文档化的决定。
func TestCapMemoryIndexSingleLineOverBudget(t *testing.T) {
	in := strings.Repeat("记", memoryIndexBudgetBytes/3+1) // 纯中文一行 > 4096 字节
	if len(in) <= memoryIndexBudgetBytes {
		t.Fatalf("测试输入应超预算: %d", len(in))
	}
	res := capMemoryIndex(in)
	if !strings.HasSuffix(res, memoryIndexHint) {
		t.Fatalf("应追加提示: %q", res)
	}
	if body := strings.TrimSuffix(res, memoryIndexHint); body != "" {
		t.Fatalf("单行超预算应整行舍弃（不切半个字）: %q", body)
	}
	if !utf8.ValidString(res) {
		t.Fatalf("结果不是合法 UTF-8: %q", res)
	}
}

// hasUnclosedLink 是测试侧的独立断言：任何 "[" 都必须在后续找到 "]"；任何
// "](...(" 形态的链接都必须有 ")" 收尾。用于校验截断结果不含悬空链接。
func hasUnclosedLink(s string) bool {
	i := 0
	for i < len(s) {
		j := strings.IndexByte(s[i:], '[')
		if j < 0 {
			return false
		}
		open := i + j
		k := strings.IndexByte(s[open+1:], ']')
		if k < 0 {
			return true
		}
		close := open + 1 + k
		// "]" 后紧跟 "(" 视为链接，需要 ")" 收尾
		if close+1 < len(s) && s[close+1] == '(' {
			if m := strings.IndexByte(s[close+2:], ')'); m < 0 {
				return true
			}
		}
		i = close + 1
	}
	return false
}
