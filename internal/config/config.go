// config.go — internal/config 核心骨架（P4 纯结构拆分；2026-10-04 Load 三层
// 装配体迁 config_load.go，Load 本体只做装配）。
// 本文件仅保留：Load 装配器、资源目录解析（resolveResourceDir /
// ResolveResourceDirForTest / DataRoot / dirExists）与 funcMu。
// 域拆分：配置键常量 → config_keys.go；configFile / Config 类型 → config_types.go；
// 功能级模型绑定 → config_features.go；布尔偏好 → config_prefs.go；
// 实时语音 Realtime → config_realtime.go；Save / saveSetters → config_save.go；
// Load 三层（默认值/环境变量/配置文件）→ config_load.go。
// 纯结构拆分：逻辑 / 字段 / 函数签名 / 默认值零改动，仅文件重组。

package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// funcMu 保护功能级模型绑定字段（GetFeatureModel/SetFeatureModel 并发读写）
var funcMu sync.RWMutex

// Load 加载配置（只应调用一次）。
// 优先级：config 文件 > 环境变量 > 默认值；逐层装配见 config_load.go。
func Load() *Config {
	// v4.441.1 走查第三面：走查态主目录重定向到沙箱（.gaea_config.json/
	// .gaea_token.json 全部落 Temp\gaea-walkthrough\home，真实主目录配置不可
	// 读不可写）；非走查态与 os.UserHomeDir 等价。
	home, err := userHome()
	if err != nil {
		slog.Warn("获取用户主目录失败", "error", err)
	}

	// 1. 硬编码默认值（最低优先级）
	cfg := loadDefaults(home)

	// 2. 环境变量覆盖（中优先级）
	applyEnvOverrides(cfg)

	// 3. Config 文件覆盖（最高优先级）
	applyConfigFile(cfg, home)

	// 4. 解析资源目录（prompts/ skills/ 等）
	cfg.ResourceDir = resolveResourceDir()

	// 5. 走查书架隔离（v4.441.1 第三数据面）：最后一步强制覆盖——硬编码默认
	//    （C:\AI\xiaoshuo）、WUBI_NOVELS_DIR、配置文件三个来源在走查态一律失效，
	//    建档只能落 Temp\gaea-walkthrough\novels（隔离优先于一切）。
	ApplyNovelsDirWalkthrough(cfg)

	return cfg
}

// resolveResourceDir 找到 prompts/ 和 skills/ 所在的资源根目录。
// 优先基于 os.Executable() 向上查找，回退到 CWD。
func resolveResourceDir() string {
	// 环境变量优先：部署/开发可显式指定资源根，避免桌面副本找不到 prompts
	// 导致数据目录分裂（统计/引擎状态落在 exe 所在目录）。
	if v := os.Getenv("GAEA_RESOURCE_DIR"); v != "" {
		if dirExists(filepath.Join(v, "prompts")) {
			return v
		}
	}
	// 尝试从可执行文件路径向上查找
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for range 4 {
			if dirExists(filepath.Join(dir, "prompts")) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	// 用户级数据根（%APPDATA%/gaea）：exe 放在任意位置（如桌面）也能找到资源与数据。
	if ud, err := os.UserConfigDir(); err == nil {
		userRoot := filepath.Join(ud, "gaea")
		if dirExists(filepath.Join(userRoot, "prompts")) {
			return userRoot
		}
	}
	// 回退：当前工作目录
	if cwd, err := os.Getwd(); err == nil {
		if dirExists(filepath.Join(cwd, "prompts")) {
			return cwd
		}
	}
	return "."
}

// ResolveResourceDirForTest 暴露资源目录解析结果（仅测试/诊断用）。
func ResolveResourceDirForTest() string {
	return resolveResourceDir()
}

// DataRoot 返回用户级数据根目录（引擎状态/模型统计/轻语/聊天/角色库等）。
// 与 exe 位置无关：桌面副本或任意路径启动都读写同一份数据，避免统计/状态
// 因 ResourceDir 解析差异而分裂。优先级：GAEA_DATA_ROOT > 用户配置目录/gaea
// > 回退 ResourceDir（历史行为，防止取不到用户目录时数据丢失）。
func DataRoot() string {
	if v := os.Getenv("GAEA_DATA_ROOT"); v != "" {
		return v
	}
	if ud, err := os.UserConfigDir(); err == nil && ud != "" {
		return filepath.Join(ud, "gaea")
	}
	return resolveResourceDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
