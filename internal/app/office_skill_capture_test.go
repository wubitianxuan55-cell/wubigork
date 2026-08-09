package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

func TestGaeaCaptureSkill(t *testing.T) {
	tmp := t.TempDir()
	old := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: tmp}
	defer func() { ga.cfg = old }()
	// 隔离 HOME/USERPROFILE，避免镜像写入真实全局技能目录
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("HOME", tmp)

	a := &App{}
	res, err := a.GaeaCaptureSkill(SkillCaptureInput{
		Name:        "weekly-report",
		Description: "周报生成",
		Task:        "生成本周工作周报",
		Solution:    "1. 汇总本周完成事项\n2. 列出下周计划\n3. 输出 Markdown",
	})
	if err != nil {
		t.Fatalf("GaeaCaptureSkill: %v", err)
	}
	want := filepath.Join(tmp, ".gaea", "skills", "weekly-report", "SKILL.md")
	if res.Path != want {
		t.Fatalf("path = %q, want %q", res.Path, want)
	}
	b, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	content := string(b)
	for _, wantSub := range []string{
		"---\nname: weekly-report",
		"description: 周报生成",
		"# weekly-report 技能",
		"## 适用场景\n生成本周工作周报",
		"## 操作步骤\n1. 汇总本周完成事项",
	} {
		if !strings.Contains(content, wantSub) {
			t.Errorf("skill content missing %q:\n%s", wantSub, content)
		}
	}
}

func TestGaeaCaptureSkillValidation(t *testing.T) {
	a := &App{}
	if _, err := a.GaeaCaptureSkill(SkillCaptureInput{Name: "bad name!", Task: "t", Solution: "s"}); err == nil {
		t.Fatal("expected invalid-name error")
	}
	if _, err := a.GaeaCaptureSkill(SkillCaptureInput{Name: "ok-name", Task: "", Solution: "s"}); err == nil {
		t.Fatal("expected empty-task error")
	}
	if _, err := a.GaeaCaptureSkill(SkillCaptureInput{Name: "ok-name", Task: "t", Solution: ""}); err == nil {
		t.Fatal("expected empty-solution error")
	}
}
