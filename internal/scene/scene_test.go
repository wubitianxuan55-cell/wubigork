package scene

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_CreateAndRead(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-scene-test")
	defer os.RemoveAll(dir)

	m := NewManager(dir)

	// 创建场景
	scene, err := m.Create("opening", "Opening Scene")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if scene.Meta.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if scene.Meta.Status != "draft" {
		t.Fatalf("expected status draft, got %s", scene.Meta.Status)
	}
	if scene.Meta.Order != 1 {
		t.Fatalf("expected order 1, got %d", scene.Meta.Order)
	}

	// 验证文件存在
	if _, err := os.Stat(m.contentPath(scene.Meta.ID)); os.IsNotExist(err) {
		t.Fatal("content file not created")
	}
	if _, err := os.Stat(m.metaPath(scene.Meta.ID)); os.IsNotExist(err) {
		t.Fatal("meta file not created")
	}

	// 写入内容
	scene.Content = "# Hello\n\nThis is a test."
	if err := m.Write(scene); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 读取
	got, err := m.Read(scene.Meta.ID)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if got.Content != scene.Content {
		t.Fatalf("content mismatch: got %q, want %q", got.Content, scene.Content)
	}
	if got.Meta.WordCount != len([]rune(scene.Content)) {
		t.Fatalf("word count mismatch: got %d, want %d", got.Meta.WordCount, len([]rune(scene.Content)))
	}
}

func TestManager_ListAndStitch(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-scene-stitch")
	defer os.RemoveAll(dir)

	m := NewManager(dir)

	// 创建多个场景
	s1, _ := m.Create("a", "First")
	s1.Content = "Content one"
	m.Write(s1)

	s2, _ := m.Create("b", "Second")
	s2.Content = "Content two"
	m.Write(s2)

	s3, _ := m.Create("c", "Third")
	s3.Content = "Content three"
	m.Write(s3)

	// List
	metas, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(metas) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(metas))
	}
	if metas[0].Order != 1 || metas[1].Order != 2 || metas[2].Order != 3 {
		t.Fatal("orders not sequential")
	}

	// Stitch
	stitched, err := m.Stitch()
	if err != nil {
		t.Fatalf("Stitch failed: %v", err)
	}
	if stitched != "Content one\n\n---\n\nContent two\n\n---\n\nContent three" {
		t.Fatalf("stitch mismatch: %q", stitched)
	}
}

func TestManager_Reorder(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-scene-reorder")
	defer os.RemoveAll(dir)

	m := NewManager(dir)

	s1, _ := m.Create("a", "First")
	s2, _ := m.Create("b", "Second")
	s3, _ := m.Create("c", "Third")

	// 反序排列
	err := m.Reorder([]string{s3.Meta.ID, s2.Meta.ID, s1.Meta.ID})
	if err != nil {
		t.Fatalf("Reorder failed: %v", err)
	}

	metas, _ := m.List()
	if metas[0].ID != s3.Meta.ID || metas[1].ID != s2.Meta.ID || metas[2].ID != s1.Meta.ID {
		t.Fatal("reorder did not take effect")
	}
}

func TestManager_Delete(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "wubigork-scene-delete")
	defer os.RemoveAll(dir)

	m := NewManager(dir)
	scene, _ := m.Create("tmp", "Temp")
	scene.Content = "temporary"
	m.Write(scene)

	if err := m.Delete(scene.Meta.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证文件已删除
	if _, err := os.Stat(m.contentPath(scene.Meta.ID)); !os.IsNotExist(err) {
		t.Fatal("content file not deleted")
	}
	if _, err := os.Stat(m.metaPath(scene.Meta.ID)); !os.IsNotExist(err) {
		t.Fatal("meta file not deleted")
	}
}
