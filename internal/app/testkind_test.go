package app

import (
	"fmt"
	"sync/atomic"
	"testing"
)

// testKindSeq 进程级单调递增序号：provider 注册表是进程级全局互斥注册（重复
// panic）。同一测试在 `-count` 多次运行中固定 kind 会重复注册——用全局计数
// 器后缀保证每次注册的 kind 在整进程内唯一（-count 任意次均不撞）。
var testKindSeq atomic.Int64

// testKind 生成测试用唯一 provider kind（前缀 + 单调序号）。
func testKind(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, testKindSeq.Add(1))
}

// cleanupWhisperSession 注册 t.Cleanup：从进程级 whisperSessions 缓存删除
// personalityID 对应的会话。whisperSessions 是包级全局缓存（whisper_handler.go），
// 固定会话 ID 在 `-count` 多次运行下会命中上次运行的 orch（跨 app 实例串扰，
// 测试各自 fresh app + fresh data root），用完即删——与
// gaea_whisper_proactive_test / whisper_persist_concurrency_test 既有模式一致。
func cleanupWhisperSession(t *testing.T, personalityID string) {
	t.Helper()
	t.Cleanup(func() {
		whisperSessionsMu.Lock()
		delete(whisperSessions, "whisper_"+personalityID)
		whisperSessionsMu.Unlock()
	})
}
