package backup

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// writeSQLite 创建数据库并执行一条 SQL。
func writeSQLite(path, stmt string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(stmt)
	return err
}

// writeSQLiteRows 在 WAL 模式写入若干行（制造未合并的 WAL）。
func writeSQLiteRows(path, stmt string, args ...string) error {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL")
	if err != nil {
		return err
	}
	defer db.Close()
	for _, a := range args {
		if _, err := db.Exec(stmt, a); err != nil {
			return err
		}
	}
	// 不执行 wal_checkpoint，保持 WAL 未合并
	return nil
}

// querySQLiteCount 读取表行数。
func querySQLiteCount(t *testing.T, path string) int {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM t").Scan(&n); err != nil {
		t.Fatalf("查询行数失败: %v", err)
	}
	return n
}

// listDirNames 列目录名（测试辅助）。
func listDirNames(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Logf("listDirNames %s err=%v", dir, err)
		return nil
	}
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	return out
}

// readOrEmpty 读文件内容或空串（测试辅助）。
func readOrEmpty(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return "<err:" + err.Error() + ">"
	}
	return string(data)
}

