package feedback

import (
	"os"
	"strings"
	"testing"
)

func TestAppendListDelete(t *testing.T) {
	dir := t.TempDir()
	a, err := Append(dir, Item{Title: "read_file 采样危险", Detail: "混合采样对核对数字的任务危险", Category: "tool-friction", Severity: "high", Board: "office"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" || a.CreatedAt == "" {
		t.Fatalf("Append 应自动补 ID/CreatedAt: %+v", a)
	}
	b, _ := Append(dir, Item{Title: "第二条", Category: "other"})

	items := List(dir)
	if len(items) != 2 {
		t.Fatalf("List = %d 条, want 2", len(items))
	}
	if items[0].ID != b.ID || items[1].ID != a.ID {
		t.Fatalf("List 应新→旧: %s, %s", items[0].ID, items[1].ID)
	}

	ok, err := Delete(dir, a.ID)
	if err != nil || !ok {
		t.Fatalf("Delete: ok=%v err=%v", ok, err)
	}
	if got := List(dir); len(got) != 1 || got[0].ID != b.ID {
		t.Fatalf("删除后应剩第二条: %+v", got)
	}
	// 幂等：再删同 ID 不报错
	if ok, _ := Delete(dir, a.ID); ok {
		t.Fatal("重复删除应返回 false")
	}
}

func TestBadLinesSkipped(t *testing.T) {
	dir := t.TempDir()
	good, _ := Append(dir, Item{Title: "好条目"})
	// 手工塞坏行（模拟历史损坏）——List 跳过不炸
	f, _ := os.OpenFile(Path(dir), os.O_WRONLY|os.O_APPEND, 0o644)
	f.WriteString("not-json\n{\"id\":\"\"}\n")
	f.Close()
	items := List(dir)
	if len(items) != 1 || items[0].ID != good.ID {
		t.Fatalf("坏行应跳过: %+v", items)
	}
}

func TestAppendValidation(t *testing.T) {
	dir := t.TempDir()
	if _, err := Append(dir, Item{Title: ""}); err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("空 title 应拒绝: %v", err)
	}
	if _, err := Append(dir, Item{Title: "x", Detail: strings.Repeat("a", 9<<10)}); err == nil || !strings.Contains(err.Error(), "8KiB") {
		t.Fatalf("超限 detail 应拒绝: %v", err)
	}
	if _, err := Append("", Item{Title: "x"}); err == nil {
		t.Fatal("空 userDir 应拒绝")
	}
	// 空 detail 合法（title 已承载主信息）
	if _, err := Append(dir, Item{Title: "只有标题"}); err != nil {
		t.Fatalf("空 detail 合法: %v", err)
	}
}
