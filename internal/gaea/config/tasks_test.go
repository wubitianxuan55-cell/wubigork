package config

import (
	"testing"

	"github.com/BurntSushi/toml"
)

// TestTasksConfigTOML 验证 [tasks] 段的 TOML 解析（v4.5.1a 红线补课：S1.4
// 任务按空间分账的生产配置面）。显式 `per_space = {}` 解码为空 map（非 nil），
// 供应用层区分「缺省启用分账」与「显式关闭回退全局 sem」。
func TestTasksConfigTOML(t *testing.T) {
	var cfg Config
	src := `
[tasks]
max_concurrent = 2
per_space = {}

[tasks.priority]
file_index = 5
price_fetch = 30
`
	if _, err := toml.Decode(src, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Tasks.MaxConcurrent != 2 {
		t.Fatalf("max_concurrent = %d, want 2", cfg.Tasks.MaxConcurrent)
	}
	if cfg.Tasks.PerSpace == nil || len(cfg.Tasks.PerSpace) != 0 {
		t.Fatalf("per_space = %#v, want 空 map（显式关闭分账）", cfg.Tasks.PerSpace)
	}
	if got := cfg.Tasks.Priority["file_index"]; got != 5 {
		t.Fatalf("priority.file_index = %d, want 5", got)
	}
	if got := cfg.Tasks.Priority["price_fetch"]; got != 30 {
		t.Fatalf("priority.price_fetch = %d, want 30", got)
	}
}

// TestTasksConfigDefaultNil 缺省（段缺失）时字段为零值：PerSpace/Priority 为
// nil，由应用层 taskSchedulerOptions 落默认（空间分账启用）。
func TestTasksConfigDefaultNil(t *testing.T) {
	var cfg Config
	if _, err := toml.Decode("", &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Tasks.PerSpace != nil {
		t.Fatalf("缺省 per_space = %#v, want nil", cfg.Tasks.PerSpace)
	}
	if cfg.Tasks.Priority != nil {
		t.Fatalf("缺省 priority = %#v, want nil", cfg.Tasks.Priority)
	}
}
