package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/prompt"
)

// TestPickCreateChapterTemplate_FirstChapterUsesOpening 第一章（章号==1）应选
// 开篇专用模板 create-chapter-first；其余章号走通用模板。
func TestPickCreateChapterTemplate_FirstChapterUsesOpening(t *testing.T) {
	eng := prompt.NewEngine("../../prompts")
	if eng.Get("create-chapter") == nil || eng.Get("create-chapter-first") == nil {
		t.Fatal("缺少 create-chapter / create-chapter-first 模板（../../prompts）")
	}
	if got := pickCreateChapterTemplate(eng, 1); got.Name != "create-chapter-first" {
		t.Errorf("第一章应选 create-chapter-first，got %q", got.Name)
	}
	for _, n := range []int{0, 2, 10, 999} {
		if got := pickCreateChapterTemplate(eng, n); got.Name != "create-chapter" {
			t.Errorf("章号 %d 应选 create-chapter，got %q", n, got.Name)
		}
	}
}

// TestPickCreateChapterTemplate_FallbackWithoutOpening 只有通用模板的引擎
// （旧磁盘 prompts/ / 测试桩）第一章应回落 create-chapter，不因缺文件而失败。
func TestPickCreateChapterTemplate_FallbackWithoutOpening(t *testing.T) {
	dir := t.TempDir()
	minimal := `{"name":"create-chapter","system":"s","task":"t","input_sections":{},` +
		`"output":{"format":"markdown","description":"d"},"constraints":{"must":[],"forbidden":[]}}`
	if err := os.WriteFile(filepath.Join(dir, "create-chapter.json"), []byte(minimal), 0o644); err != nil {
		t.Fatal(err)
	}
	eng := prompt.NewEngine(dir)
	if got := pickCreateChapterTemplate(eng, 1); got == nil || got.Name != "create-chapter" {
		t.Errorf("缺开篇模板时第一章应回落 create-chapter，got %v", got)
	}
}
