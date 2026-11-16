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
