package billing

import (
	"fmt"
	"sync/atomic"
)

// testKindSeq 进程级单调递增序号：余额提供者注册表是进程级全局互斥注册
// （重复 panic）。同一测试在 `-count` 多次运行中固定 kind 会重复注册——
// 用全局计数器后缀保证每次注册的 kind 在整进程内唯一。
var testKindSeq atomic.Int64

// testKind 生成测试用唯一余额提供者 kind（前缀 + 单调序号）。
func testKind(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, testKindSeq.Add(1))
}
