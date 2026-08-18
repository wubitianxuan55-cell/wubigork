package main

import (
	"io/fs"
	"testing"

	"github.com/gaea/gaea/internal/prompt"
)

// TestMaskToken 脱敏格式：只保留尾 4 位，前缀固定为 ***（T6-1.3）。
func TestMaskToken(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "***"},
		{"abcd", "***"},
		{"abcde", "***bcde"},
		{"abcdefgh", "***efgh"},
		{"tok-123456", "***3456"},
	}
	for _, c := range cases {
		if got := maskToken(c.in); got != c.want {
			t.Errorf("maskToken(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestPromptTemplatesEmbedded 单文件 exe 分发兜底：prompts/ 必须随二进制嵌入，
// 且全部模板可解析（桌面副本 exe 找不到磁盘 prompts/ 时仍能生成剧情/章节等）。
func TestPromptTemplatesEmbedded(t *testing.T) {
	files, err := fs.Glob(promptTemplates, "prompts/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("prompts/*.json 未嵌入二进制（files=%v err=%v）", files, err)
	}
	eng := prompt.NewEngineWithEmbedded("", promptTemplates)
	names := []string{
		"plot-branch-browser", "create-chapter", "chapter-generate",
		"chapter-summary", "worldview-agent", "character-agent",
	}
	for _, name := range names {
		if tmpl := eng.Get(name); tmpl == nil {
			t.Errorf("内置模板 %q 应可用（实际缺失）", name)
		}
	}
}
