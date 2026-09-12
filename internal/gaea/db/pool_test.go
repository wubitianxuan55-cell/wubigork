package db

// 刀C（v4.248）连接池放宽的并发正确性：多 goroutine 并发写由 SQLite 单写
// 者锁 + busy_timeout 兜底（零 BUSY 逃逸、全量落库）；写事务进行中读连接
// 可用且看到 WAL 快照（改前单连接池下本测试会死锁——读要等事务连接）。

import (
	"sync"
	"testing"
)

func TestConcurrentWritesAllPersisted(t *testing.T) {
	dir := t.TempDir()
	gdb := GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer CloseDatabase(dir)
	if _, err := gdb.Exec(`CREATE TABLE IF NOT EXISTS bench_conc(id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := gdb.Exec(`DELETE FROM bench_conc`); err != nil {
		t.Fatal(err)
	}

	const writers, perWriter = 8, 25
	var wg sync.WaitGroup
	errCh := make(chan error, writers)
	for g := 0; g < writers; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				if _, err := gdb.Exec(`INSERT INTO bench_conc(id, v) VALUES(?,?)`, g*1000+i, "并发写"); err != nil {
					errCh <- err
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("并发写失败（busy_timeout 未兜住）: %v", err)
	}
	var n int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM bench_conc`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != writers*perWriter {
		t.Fatalf("落库 = %d, want %d", n, writers*perWriter)
	}
}

func TestReadWhileWriteTxInProgress(t *testing.T) {
	dir := t.TempDir()
	gdb := GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer CloseDatabase(dir)
	if _, err := gdb.Exec(`CREATE TABLE IF NOT EXISTS bench_conc(id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := gdb.Exec(`DELETE FROM bench_conc`); err != nil {
		t.Fatal(err)
	}

	tx, err := gdb.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO bench_conc(id, v) VALUES(1, '进行中')`); err != nil {
		t.Fatal(err)
	}

	// 写事务未提交：另一连接可读（WAL 快照隔离，看不到未提交行）。
	var n int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM bench_conc`).Scan(&n); err != nil {
		t.Fatalf("写事务进行中读失败（连接池未放宽的典型死锁点）: %v", err)
	}
	if n != 0 {
		t.Fatalf("未提交行泄漏: %d", n)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM bench_conc`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("提交后可见 = %d, want 1", n)
	}
}
