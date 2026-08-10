package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGaeaTaskTemplates(t *testing.T) {
	a := &App{}
	tmpls := a.GaeaTaskTemplates()
	if len(tmpls) < 5 {
		t.Fatalf("模板库应不少于 5 个: %d", len(tmpls))
	}
	seen := map[string]bool{}
	for _, tm := range tmpls {
		if tm.Name == "" || tm.Title == "" || tm.Prompt == "" {
			t.Fatalf("模板字段缺失: %+v", tm)
		}
		if seen[tm.Name] {
			t.Fatalf("模板名重复: %s", tm.Name)
		}
		seen[tm.Name] = true
	}
}

func TestEnsureTaskTemplateCommands(t *testing.T) {
	dir := t.TempDir()
	if err := ensureTaskTemplateCommands(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".gaea", "commands", "weekly-report.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("应生成 weekly-report.md: %v", err)
	}
	if !strings.Contains(string(b), "description:") || !strings.Contains(string(b), "周报") {
		t.Fatalf("模板文件内容异常: %s", string(b))
	}
	// 幂等：不覆盖已有文件
	if err := os.WriteFile(path, []byte("---\ndescription: 用户自定义\n---\n\n我的命令"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureTaskTemplateCommands(dir); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(path)
	if !strings.Contains(string(b), "用户自定义") {
		t.Fatal("幂等安装覆盖了用户文件")
	}
}
