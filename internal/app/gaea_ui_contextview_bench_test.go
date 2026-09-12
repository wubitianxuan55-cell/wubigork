package app

// 刀E（v4.249）性能基线：看板条目读取。改前四个绑定每次轮询刷新都全量
// 读+逐行解析整份日志；缓存形态在文件未变时 O(1)。

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func benchSessionLog(b *testing.B, target int) string {
	b.Helper()
	dir := b.TempDir()
	sessionPath := filepath.Join(dir, "bench.jsonl")
	logPath := session.LogPathFor(sessionPath)
	w, err := session.OpenLog(logPath, "", "work")
	if err != nil {
		b.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]string{"text": string(make([]byte, 500))})
	written := 0
	for written < target {
		if _, err := w.AppendRaw("text", payload); err != nil {
			b.Fatal(err)
		}
		written += len(payload) + 60
	}
	if err := w.Close(); err != nil {
		b.Fatal(err)
	}
	return sessionPath
}

// BenchmarkContextEntriesRead 改前=直读全量解析；改后=缓存命中（size+mtime
// 未变，O(1)）。
func BenchmarkContextEntriesRead(b *testing.B) {
	sessionPath := benchSessionLog(b, 2<<20)
	b.Run("fresh", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := session.ReadEntriesFor(sessionPath); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("cached", func(b *testing.B) {
		if _, err := readEntriesForCached(sessionPath); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := readEntriesForCached(sessionPath); err != nil {
				b.Fatal(err)
			}
		}
	})
}
