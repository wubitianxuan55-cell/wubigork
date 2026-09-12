package db

// 刀C（v4.248）性能基线：连接池形态。改前 SetMaxOpenConns(1) 把读也串到
// 单连接（WAL 本可读并行），且「rows 未占用连接时 Exec 等待」是整类死锁
// 的根源（portrait.go / whisper fts 注脚）；改后读写并行、写由单写者锁 +
// busy_timeout 兜底。

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
)

func benchDB(b *testing.B) (*sql.DB, func()) {
	b.Helper()
	dir := b.TempDir()
	gdb := GetDatabase(dir)
	if gdb == nil {
		b.Fatal("GetDatabase nil")
	}
	if _, err := gdb.Exec(`CREATE TABLE IF NOT EXISTS bench_read(id INTEGER PRIMARY KEY, txt TEXT)`); err != nil {
		b.Fatal(err)
	}
	if _, err := gdb.Exec(`DELETE FROM bench_read`); err != nil {
		b.Fatal(err)
	}
	tx, err := gdb.Begin()
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 500; i++ {
		if _, err := tx.Exec(`INSERT INTO bench_read(id, txt) VALUES(?,?)`, i, fmt.Sprintf("基准行 %d：中等长度文本内容。", i)); err != nil {
			b.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}
	return gdb, func() { CloseDatabase(dir) }
}

// BenchmarkDBParallelReads 并发点查（b.RunParallel）：单连接池下全部串行，
// 放宽后 WAL 读并行。
func BenchmarkDBParallelReads(b *testing.B) {
	gdb, closeDB := benchDB(b)
	defer closeDB()
	var i atomic.Int64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		n := int64(0)
		for pb.Next() {
			id := i.Add(1) % 500
			var txt string
			if err := gdb.QueryRow(`SELECT txt FROM bench_read WHERE id=?`, id).Scan(&txt); err != nil {
				b.Fatal(err)
			}
			n++
		}
		_ = n
	})
}

// BenchmarkDBParallelWrites 并发写（b.RunParallel）：单写者锁 + busy_timeout
// 兜底的正确性与成本。
func BenchmarkDBParallelWrites(b *testing.B) {
	gdb, closeDB := benchDB(b)
	defer closeDB()
	var i atomic.Int64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := i.Add(1)
			if _, err := gdb.Exec(`INSERT INTO bench_read(id, txt) VALUES(?,?)`,
				100000+n, "并发写 "+strconv.FormatInt(n, 10)); err != nil {
				b.Fatal(err)
			}
		}
	})
}
