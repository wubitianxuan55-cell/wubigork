package session

// 刀E（v4.249）预览缓存：会话文件整体重写（transcript save），size+mtime
// 未变 ⇒ 内容未变；重写后必须立即可见（失效语义）。

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewSessionCacheInvalidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.jsonl")
	body := `{"role":"user","content":"你好世界"}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	p1, t1 := previewSessionCached(path)
	p2, t2 := previewSessionCached(path)
	if p1 != "你好世界" || t1 != 1 || p2 != p1 || t2 != t1 {
		t.Fatalf("首读/二读 = (%q,%d)/(%q,%d)", p1, t1, p2, t2)
	}

	// 追加一轮（transcript 重写，size 增长）⇒ 缓存失效。
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"role":"user","content":"第二轮"}` + "\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	p3, t3 := previewSessionCached(path)
	if t3 != 2 || p3 != "你好世界" {
		t.Fatalf("重写后 = (%q,%d), want (你好世界,2)（缓存失效语义破坏）", p3, t3)
	}
}
