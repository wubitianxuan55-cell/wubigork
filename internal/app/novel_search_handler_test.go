package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
)

// TestNovelSearch_Guards 覆盖全文检索守卫分支（不读取真实项目文件）。
func TestNovelSearch_Guards(t *testing.T) {
	a := newCharacterLibTestApp(t)

	if _, err := a.NovelSearch("剑修"); err == nil || !strings.Contains(err.Error(), "请先打开项目") {
		t.Fatalf("无项目时应报错: %v", err)
	}
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	if _, err := a.NovelSearch("剑修"); err == nil || !strings.Contains(err.Error(), "大纲未初始化") {
		t.Fatalf("outlineAgent 为空时应报错: %v", err)
	}
	hits, err := a.NovelSearch("   ")
	if err != nil || len(hits) != 0 {
		t.Fatalf("空查询应返回空结果: %v %v", hits, err)
	}
}

func TestSnippetAround(t *testing.T) {
	body := strings.Repeat("甲", 100) + "目标词" + strings.Repeat("乙", 100)
	idx := strings.Index(body, "目标词")
	s := snippetAround(body, idx, 3)
	if !strings.Contains(s, "目标词") || !strings.HasPrefix(s, "…") || !strings.HasSuffix(s, "…") {
		t.Fatalf("snippet 应包含命中词且带省略号: %q", s)
	}
	short := snippetAround("目标词", 0, 3)
	if short != "目标词" {
		t.Fatalf("短文本不应加省略号: %q", short)
	}
}
