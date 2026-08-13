package app

// Herdsman LAN 暴露安全检测（H0-4）：解析 herdsman config.yaml 的 api 段
// （lan_accessible/port），暴露时返回结构化告警与中文处置指引。
// 仅提示、不改 herdsman 配置。

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/herdsman"
)

// HerdsmanSecurityCheck 检测 Herdsman API 的局域网暴露风险。
//
// config 路径定位：HERDSMAN_CONFIG 环境变量优先，否则回退
// %USERPROFILE%\.herdsman\config.yaml（os.UserHomeDir）。结果结构
// 见 herdsman.LanExposure，供前端在启动/模型中心展示告警与处置指引。
func (a *App) HerdsmanSecurityCheck() herdsman.LanExposure {
	return herdsman.CheckLanExposure(herdsmanConfigPath())
}

// herdsmanConfigPath 定位 herdsman 配置路径（HERDSMAN_CONFIG 优先，回退用户目录）。
func herdsmanConfigPath() string {
	if p := strings.TrimSpace(os.Getenv("HERDSMAN_CONFIG")); p != "" {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".herdsman", "config.yaml")
	}
	return ""
}
