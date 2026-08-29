package session

// space.go — S2 会话空间（work/play）的路径归属推导。
// 设计权威：docs/gaea-space-dimension-design.md §2.2/§3/§8（风险 3）。
// 空间归属以「路径」为唯一真相源：恢复链（resumeLastSession→ResumeFromDisk→
// Restore）不读 meta，space 只能从会话目录推导——目录分区方案的决定性优势。
// 旧平铺会话恒按 work 兼容（读端降级单点在 spaces.Normalize/SpaceOr）。

import (
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/spaces"
)

// SpaceForPath 从会话文件路径推导所属空间：
//   - <root>/.gaea/sessions/play[/archive]/<file> → play
//   - 其余（work 分区、平铺目录、平铺 archive、临时目录）→ work 兼容
//
// 空路径返回 work（防御）。
func SpaceForPath(sessionPath string) string {
	if sessionPath == "" {
		return spaces.SpaceWork
	}
	return spaces.SpaceForDir(filepath.Dir(sessionPath))
}
