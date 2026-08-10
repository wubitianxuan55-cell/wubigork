package pins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	if got := mustLoad(t, dir); len(got) != 0 {
		t.Fatal("空工作区应返回空清单")
	}
	next, err := Add(dir, "docs/说明.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 1 || next[0] != "docs/说明.md" {
		t.Fatalf("Add 结果异常: %+v", next)
	}
	// 重复固定去重
	next, _ = Add(dir, "docs/说明.md")
	if len(next) != 1 {
		t.Fatalf("重复固定未去重: %+v", next)
	}
	// 重新加载
	got := mustLoad(t, dir)
	if len(got) != 1 || got[0] != "docs/说明.md" {
		t.Fatalf("Load 不一致: %+v", got)
	}
}

func TestAddRemove(t *testing.T) {
	dir := t.TempDir()
	_, _ = Add(dir, "a.md")
	_, _ = Add(dir, "b.md")
	next, _ := Remove(dir, "a.md")
	if len(next) != 1 || next[0] != "b.md" {
		t.Fatalf("Remove 结果异常: %+v", next)
	}
	got := mustLoad(t, dir)
	if len(got) != 1 || got[0] != "b.md" {
		t.Fatalf("Remove 未持久化: %+v", got)
	}
}

func TestCleanRelRejectsEscape(t *testing.T) {
	for _, bad := range []string{"../secret.md", "a/../../secret.md", `/abs/path.md`, "C:\\x.md", "", "./"} {
		if cleanRel(bad) != "" {
			t.Fatalf("应拒绝路径 %q，got %q", bad, cleanRel(bad))
		}
	}
	if got := cleanRel("./docs/方案.md"); got != "docs/方案.md" {
		t.Fatalf("应规范化相对路径: %q", got)
	}
}

func TestBlockIncludesTextAndListsOffice(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "说明.md"), []byte("这是固定的项目说明正文。"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "报价.xlsx"), []byte("not-a-real-xlsx"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _ = Add(dir, "docs/说明.md")
	_, _ = Add(dir, "docs/报价.xlsx")

	block := Block(dir)
	if block == "" {
		t.Fatal("Block 不应为空")
	}
	if !strings.Contains(block, "项目说明正文") {
		t.Fatalf("文本类正文应注入: %s", block)
	}
	if !strings.Contains(block, "办公文档") || !strings.Contains(block, "docs/报价.xlsx") {
		t.Fatalf("办公文档应列名: %s", block)
	}
}

func TestBlockSkipsMissingAndEmpty(t *testing.T) {
	dir := t.TempDir()
	_, _ = Add(dir, "不存在.md")
	if Block(dir) != "" {
		t.Fatal("全部文件缺失时 Block 应为空")
	}
}

func TestBlockCaps(t *testing.T) {
	dir := t.TempDir()
	writeBig := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _ = Add(dir, name)
	}
	writeBig("big.md", strings.Repeat("长", 3000))
	block := Block(dir)
	if len([]rune(block)) > maxTotalRunes+512 {
		t.Fatalf("Block 超出注入上限: %d", len([]rune(block)))
	}
}

func mustLoad(t *testing.T, dir string) []string {
	t.Helper()
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return got
}
