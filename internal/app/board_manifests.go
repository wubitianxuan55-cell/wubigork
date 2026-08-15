package app

import (
	"github.com/gaea/gaea/internal/app/board"
)

// GetBoardManifests 返回 canonical 板块 manifest 清单（§5.2）：
//   - 前端 MainLayout/ModuleLauncher 数据驱动（菜单/快捷键/页面映射/启动器
//     全部由清单生成，附 B 的 12 个硬编码点收敛）；
//   - 启动自检与审计 dump（与 GetBoardManifests 之外的「实际装配结果」对应，
//     见 09 报告 §6.3 的 DSH dump-config 借鉴）。
//
// Wails 绑定挂 CoreB（gen_bindings explicitOverrides：GetBoardManifests→core）。
func (a *App) GetBoardManifests() []board.Manifest {
	return board.BuiltinManifests()
}
