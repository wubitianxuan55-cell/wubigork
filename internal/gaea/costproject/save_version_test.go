package costproject

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// TestSaveVersionSequenceAndUniqueIndex（2026-09-19 审计 P2 落地回归）：
// ①版本号连续（INSERT 内 SELECT MAX+1 自算，读改写窗口消除）；
// ②SchemaV22 唯一索引存在——直接插重复 (project_id, version) 被硬约束拒绝。
func TestSaveVersionSequenceAndUniqueIndex(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := Open(gdb)

	pid, err := s.SaveProject(Project{Name: "版本序列测算", ProjectType: "房建"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveItem(Item{ProjectID: pid, Name: "c30", Title: "C30 混凝土", Quantity: 10, Price: 400}); err != nil {
		t.Fatal(err)
	}

	v1, err := s.SaveVersion(pid, "第一版")
	if err != nil {
		t.Fatalf("SaveVersion 1: %v", err)
	}
	v2, err := s.SaveVersion(pid, "第二版")
	if err != nil {
		t.Fatalf("SaveVersion 2: %v", err)
	}
	if v1.Version != 1 || v2.Version != 2 {
		t.Fatalf("版本应连续 1,2: got %d,%d", v1.Version, v2.Version)
	}

	// 唯一索引硬约束：绕过 SaveVersion 直插重复版本号必须被拒。
	if _, err := gdb.Exec(`INSERT INTO cost_estimate_versions(project_id, version, total, snapshot, note, created_at)
		VALUES(?, 2, 0, '[]', '绕过', '2026-01-01T00:00:00Z')`, pid); err == nil {
		t.Fatal("重复 (project_id, version) 应被唯一索引拒绝")
	}
}
