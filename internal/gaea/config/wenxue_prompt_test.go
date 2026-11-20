package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveWenxueSystemPrompt 锁定图文板块身份解析链（2026-11-16 图文身份
// 分板）：双键空 = 代码默认（恒不回落办公 DefaultSystemPrompt）；wenxue_
// system_prompt 非空生效；wenxue_system_prompt_file 优先于内联键，读失败硬
// 报错（显式配置的档位缺档不许静默吞）。
func TestResolveWenxueSystemPrompt(t *testing.T) {
	t.Run("empty_keys_fall_back_to_board_default", func(t *testing.T) {
		cfg := Default()
		got, err := cfg.ResolveWenxueSystemPrompt()
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if got != DefaultWenxueSystemPrompt {
			t.Fatalf("空键应回落图文默认，got %.80q", got)
		}
		if strings.Contains(got, "土壤修复") {
			t.Fatal("图文默认人格不得携带办公域语义")
		}
	})

	t.Run("inline_key_overrides", func(t *testing.T) {
		cfg := Default()
		cfg.Agent.WenxueSystemPrompt = "你是测试用图文人格。"
		got, err := cfg.ResolveWenxueSystemPrompt()
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if got != "你是测试用图文人格。" {
			t.Fatalf("内联键应生效，got %q", got)
		}
	})

	t.Run("file_key_wins_and_read_error_is_hard", func(t *testing.T) {
		dir := t.TempDir()
		good := filepath.Join(dir, "wenxue.md")
		if err := os.WriteFile(good, []byte("你是文件版图文人格。\n"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		cfg := Default()
		cfg.Agent.WenxueSystemPrompt = "内联键"
		cfg.Agent.WenxueSystemPromptFile = good
		got, err := cfg.ResolveWenxueSystemPrompt()
		if err != nil || got != "你是文件版图文人格。" {
			t.Fatalf("文件键应优先：(%.40q,%v)", got, err)
		}

		cfg.Agent.WenxueSystemPromptFile = filepath.Join(dir, "missing.md")
		if _, err := cfg.ResolveWenxueSystemPrompt(); err == nil {
			t.Fatal("文件读失败应硬报错（仿 office system_prompt_file 语义）")
		}
	})
}

// TestWenxueSystemPromptV1FrozenBody 文学身份稿 v1.0 冻结锚（2026-11-18 众神殿
// pan_1795007052552，DeepSeek r1 附录「文学 A」原样）：范围=各类小说（非小说
// 体裁与办公域明确排除）、确认制记忆、workbench 指路句在场；旧「长篇小说
// 创作助手」自称与办公工具手册不得回流。
func TestWenxueSystemPromptV1FrozenBody(t *testing.T) {
	p := DefaultWenxueSystemPrompt
	for _, sub := range []string{
		"长篇、中篇、短篇小说",
		"不覆盖散文、剧本、诗歌和非小说评论",
		"read_skill 读 wenxue-workbench",
		"主动调用 remember",
	} {
		if !strings.Contains(p, sub) {
			t.Errorf("文学身份稿缺 v1.0 锚 %q", sub)
		}
	}
	for _, banned := range []string{
		"长篇小说创作助手", // 旧自称（范围已放宽到各类小说）
		"format_convert", "semantic_search", "office_", "fact_add",
	} {
		if strings.Contains(p, banned) {
			t.Errorf("文学身份稿含禁词 %q", banned)
		}
	}
}
