package config

import (
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

// S2 双空间：space.mode / session.space 配置键解析（仿 session.log_format
// 回退开关的测试方式：TOML 解码 → Config 方法语义）。

func TestDefaultSpaceConfig(t *testing.T) {
	cfg := Default()
	if !cfg.SpaceModeIsOn() {
		t.Fatal("缺省 space.mode 应为 on")
	}
	if got := cfg.SessionSpace(); got != spaces.SpaceWork {
		t.Fatalf("缺省 session.space = %q, want work", got)
	}
	if got := cfg.EffectiveSessionSpace(); got != spaces.SpaceWork {
		t.Fatalf("缺省生效空间 = %q, want work", got)
	}
}

func TestSpaceModeOff(t *testing.T) {
	cfg := Default()
	cfg.Space.Mode = "off"
	cfg.Session.Space = "play"
	if cfg.SpaceModeIsOn() {
		t.Fatal("显式 off 应关闭分区")
	}
	if got := cfg.EffectiveSessionSpace(); got != "" {
		t.Fatalf("mode=off 生效空间 = %q, want 空（回退平铺/日志不写 space）", got)
	}
	// SessionSpace 只解析落点值，不受 mode 影响（读端仍需要归一化语义）
	if got := cfg.SessionSpace(); got != spaces.SpacePlay {
		t.Fatalf("SessionSpace = %q, want play", got)
	}
}

func TestSessionSpaceNormalize(t *testing.T) {
	cases := map[string]string{
		"":      spaces.SpaceWork,
		"play":  spaces.SpacePlay,
		"Play":  spaces.SpacePlay, // trim + 大小写归一
		" play": spaces.SpacePlay,
		"work":  spaces.SpaceWork,
		"bogus": spaces.SpaceWork,
	}
	for raw, want := range cases {
		cfg := Default()
		cfg.Session.Space = raw
		if got := cfg.SessionSpace(); got != want {
			t.Errorf("SessionSpace(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestSpaceConfigTOML 验证 [space] mode 与 [session] space 键的 TOML 解析
// （与既有键风格一致： BurntSushi/toml 直接解码到嵌套结构）。
func TestSpaceConfigTOML(t *testing.T) {
	var cfg Config
	src := `
[space]
mode = "off"

[session]
log_format = "event"
space = "play"
`
	if _, err := toml.Decode(src, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.SpaceModeIsOn() {
		t.Fatal("[space].mode=off 未生效")
	}
	if got := cfg.EffectiveSessionSpace(); got != "" {
		t.Fatalf("mode=off 生效空间 = %q, want 空", got)
	}
	// 恢复 on：session.space 应独立生效
	cfg.Space.Mode = "on"
	if got := cfg.EffectiveSessionSpace(); got != spaces.SpacePlay {
		t.Fatalf("[session].space=play 生效空间 = %q, want play", got)
	}
}

func TestWorkspaceSessionDirSpaces(t *testing.T) {
	ws := `C:\ws`
	if got, want := WorkspaceSessionDir(ws, ""), filepath.Join(ws, ".gaea", "sessions"); got != want {
		t.Errorf("WorkspaceSessionDir(空) = %q, want 平铺 %q（mode=off 回退形态）", got, want)
	}
	if got, want := WorkspaceSessionDir(ws, spaces.SpaceWork), filepath.Join(ws, ".gaea", "sessions", "work"); got != want {
		t.Errorf("WorkspaceSessionDir(work) = %q, want %q", got, want)
	}
	if got, want := WorkspaceSessionDir(ws, spaces.SpacePlay), filepath.Join(ws, ".gaea", "sessions", "play"); got != want {
		t.Errorf("WorkspaceSessionDir(play) = %q, want %q", got, want)
	}
	// 非法值不做静默分区（调用方应传归一化后的值）
	if got, want := WorkspaceSessionDir(ws, "bogus"), filepath.Join(ws, ".gaea", "sessions"); got != want {
		t.Errorf("WorkspaceSessionDir(bogus) = %q, want 平铺 %q", got, want)
	}
}
