package types

import (
	"strconv"
	"strings"
)

// ── 章节文件名 ↔ 章号（跨域共用口径）──────────────────────

// ChapterNumOf 从章节文件名派生章号："001.md"→1、"001a.md"→1；空/无数字→0。
//
// 权威口径（handoff §2-2）：**不设章号冗余字段**，一律由文件名派生。
// 这是 internal/app.chapterNumOf 的契约层镜像（同一算法的单一来源声明），
// 各域不得再写第三份实现。
//
// v4.519：原住在 interfaces.go（设计期共享接口层，25 个导出类型全仓零引用
// 已删）；本函数是其中唯一被消费的符号，随迁独立成文件。
func ChapterNumOf(filename string) int {
	n := strings.IndexFunc(filename, func(r rune) bool { return r < '0' || r > '9' })
	digits := filename
	if n >= 0 {
		digits = filename[:n]
	}
	if digits == "" {
		return 0
	}
	v, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}
	return v
}
