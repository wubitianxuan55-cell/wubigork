package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	gaeaconfig "github.com/gaea/gaea/internal/gaea/config"
)

// walkthrough.go — 走查沙箱第三数据面：用户主目录派生配置与书架（v4.441.1）。
//
// 家族史：v4.418 封 workspace（会话/记忆/交付物）→ v4.428.1 封 APPDATA
// （os.UserConfigDir：sin/小说/角色库数据根）→ 本刀封第三面：internal/config
// 的 os.UserHomeDir 派生面——`.gaea_config.json`（读+写）与 `.gaea_token.json`、
// 以及小说书架 NovelsDir（硬编码默认 C:\AI\xiaoshuo + 用户配置可覆盖）。
// 2026-10-02 走查预检实证：沙箱壳建档会直接写进用户真实书架，走查班被迫中止。
//
// 两处必须同调纪律（v4.426 在册）：gaeaLoadConfig 与 internal/config.Load 各有
// 一套配置装配，任何一面改动须双查。

// IsWalkthrough 走查沙箱开关（显式 GAEA_WALKTHROUGH=1 才生效）。
func IsWalkthrough() bool { return os.Getenv("GAEA_WALKTHROUGH") == "1" }

// userHome 用户主目录解析：走查态重定向到沙箱 home（Temp\gaea-walkthrough\home），
// 真实 `~/.gaea_config.json`/`.gaea_token.json` 在走查壳内不可读不可写。
// 非走查态与 os.UserHomeDir() 等价。沙箱建不出来必须 fail-closed：宁可使配置面
// 报错，不静默回落真实主目录。
func userHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if !IsWalkthrough() {
		return home, nil
	}
	sandbox := filepath.Join(gaeaconfig.WalkthroughRootDir(), "home")
	if err := os.MkdirAll(sandbox, 0o755); err != nil {
		slog.Error("创建走查沙箱 home 失败（配置面拒绝回落真实主目录）", "error", err)
		return "", fmt.Errorf("创建走查沙箱 home 失败: %w", err)
	}
	return sandbox, nil
}

// ApplyNovelsDirWalkthrough 书架数据面隔离：走查态把 NovelsDir 强制指向沙箱
// （Temp\gaea-walkthrough\novels），覆盖硬编码默认与任何用户配置（隔离优先于
// 一切，同 workspace 口径）。建档 containment 护栏（CreateProject 只许书架内）
// 随之天然落沙箱。
func ApplyNovelsDirWalkthrough(cfg *Config) {
	if cfg == nil || !IsWalkthrough() {
		return
	}
	dir := filepath.Join(gaeaconfig.WalkthroughRootDir(), "novels")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Warn("创建走查沙箱书架失败（书架数据面未隔离，建档将命中真实书架！）", "error", err)
		return
	}
	if cfg.NovelsDir != dir {
		slog.Warn("GAEA_WALKTHROUGH=1：书架已强制指向走查沙箱", "from", cfg.NovelsDir, "to", dir)
	}
	cfg.NovelsDir = dir
}
