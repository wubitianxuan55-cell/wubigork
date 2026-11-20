package app

// 后台 notice 收口与 gaeaSessionSpace 的共用测试装置（自 gaea_dream_test.go
// 抽出保留——auto-dream 已删 2026-11-20，但轻语/子代理追问/knowledge 迁移
// 测试仍消费 captureGaeaNotices）。

import (
	"fmt"
	"sync"
	"testing"
)

// captureGaeaNotices 注入 notice 发射缝，返回已捕获的 notice 文案读取函数。
// 只收 gaea-event 且 kind=notice 的载荷（后台 panic 的可见通道）。
func captureGaeaNotices(t *testing.T) func() []string {
	t.Helper()
	orig := gaeaNoticeSink
	t.Cleanup(func() { gaeaNoticeSink = orig })
	var mu sync.Mutex
	var texts []string
	gaeaNoticeSink = func(name string, data map[string]interface{}) {
		if name != "gaea-event" {
			return
		}
		if kind, _ := data["kind"].(string); kind != "notice" {
			return
		}
		mu.Lock()
		texts = append(texts, fmt.Sprint(data["text"]))
		mu.Unlock()
	}
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), texts...)
	}
}
