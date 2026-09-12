package memory

// 刀B（v4.246）性能基线：记忆写路径。Save 改前=facts UPSERT 与 memory_events
// 事件两条独立写事务（WAL 下双倍 commit/fsync）；Load 改前在控制器全局锁内
// 全库重载（SELECT 全部 facts 含 body + 全量分词）。

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// benchSQLiteStore 预置 n 条事实的 SQLite 后端（TempDir 隔离，勿打真实用户库）。
func benchSQLiteStore(b *testing.B, n int) (Store, *sql.DB, func()) {
	b.Helper()
	dir := b.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		b.Fatal("GetDatabase nil")
	}
	s := SQLiteStoreFor(gdb, dir, "/bench/proj")
	for i := 0; i < n; i++ {
		if _, err := s.Save(Memory{
			Name:        fmt.Sprintf("fact-%03d", i),
			Title:       fmt.Sprintf("事实 %d", i),
			Description: fmt.Sprintf("第 %d 条基准事实：中等长度描述，供索引与注入使用。", i),
			Type:        TypeProject,
			Kind:        KindSemantic,
			Body:        fmt.Sprintf("第 %d 条事实的正文：包含若干段与关键词（造价、进度、验收），约两百字节的体量，用于让分词与 SELECT 的成本进入基准。%s", i, string(make([]byte, 160))),
		}); err != nil {
			b.Fatal(err)
		}
	}
	return s, gdb, func() { db.CloseDatabase(dir) }
}

// BenchmarkMemoryLoad 全库重载（改前在控制器全局锁内执行的成本本体）。
func BenchmarkMemoryLoad(b *testing.B) {
	for _, n := range []int{50, 300} {
		b.Run(fmt.Sprintf("facts=%d", n), func(b *testing.B) {
			_, gdb, closeDB := benchSQLiteStore(b, n)
			defer closeDB()
			dir := b.TempDir()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Load(Options{UserDir: dir, DB: gdb, CWD: "/bench/proj"})
			}
		})
	}
}

// BenchmarkMemorySave 单条写入（facts UPSERT + 事件留痕）。
func BenchmarkMemorySave(b *testing.B) {
	for _, n := range []int{50, 300} {
		b.Run(fmt.Sprintf("facts=%d", n), func(b *testing.B) {
			s, _, closeDB := benchSQLiteStore(b, n)
			defer closeDB()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := s.Save(Memory{
					Name:        fmt.Sprintf("bench-save-%d", i),
					Title:       "基准写入",
					Description: "每轮一条新事实，度量写放大。",
					Type:        TypeProject,
					Kind:        KindSemantic,
					Body:        "基准写入正文。",
				}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMemoryTouch 触达（引用解析热路径：每个被引用事实一次）。
func BenchmarkMemoryTouch(b *testing.B) {
	s, _, closeDB := benchSQLiteStore(b, 50)
	defer closeDB()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := s.Touch("fact-000"); err != nil {
			b.Fatal(err)
		}
	}
}
